package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	"github.com/anzhiyu-c/anheyu-app/pkg/integration/dp7575"
)

type bindingRepository interface {
	FindByUserID(ctx context.Context, userID int64) (MemberBindingDTO, error)
	Upsert(ctx context.Context, dto MemberBindingDTO) error
}

type memberStatusClient interface {
	MemberStatus(ctx context.Context, req dp7575.MemberStatusRequest) (dp7575.MemberStatusResponse, error)
	MemberProfile(ctx context.Context, req dp7575.MemberProfileRequest) (dp7575.MemberProfileResponse, error)
	MemberProductsCatalog(ctx context.Context) (dp7575.MemberProductsCatalogResponse, error)
	CreateOrder(ctx context.Context, req dp7575.OrderCreateRequest) (dp7575.OrderCreateResponse, error)
	OrderPaymentDetail(ctx context.Context, req dp7575.OrderPaymentDetailRequest) (dp7575.OrderPaymentDetailResponse, error)
	CardRedeemPrecheck(ctx context.Context, req dp7575.CardRedeemPrecheckRequest) (dp7575.CardRedeemPrecheckResponse, error)
	CardRedeemCreate(ctx context.Context, req dp7575.CardRedeemCreateRequest) (dp7575.CardRedeemCreateResponse, error)
	AdminCards(ctx context.Context, req dp7575.AdminCardListRequest) (dp7575.AdminCardListResponse, error)
	OrderList(ctx context.Context, req dp7575.OrderListRequest) (dp7575.OrderListResponse, error)
	AdminOrderMappings(ctx context.Context, req dp7575.AdminOrderMappingListRequest) (dp7575.AdminOrderMappingListResponse, error)
	OrderDetail(ctx context.Context, req dp7575.OrderDetailRequest) (dp7575.OrderDetailResponse, error)
	OrderStatus(ctx context.Context, req dp7575.OrderStatusRequest) (dp7575.OrderStatusResponse, error)
	ConfigComplete() bool
	HealthProbe(ctx context.Context) (dp7575.HealthProbeResult, error)
	EnsureUserMapping(ctx context.Context, req dp7575.UserMapEnsureRequest) (dp7575.UserMapEnsureResponse, error)
}

type Service struct {
	repo              bindingRepository
	client            memberStatusClient
	resourceRepo      resourceRepository
	resourceOrderRepo resourceOrderRepository
	memberZoneRepo    memberZoneRepository
	sanitizeHTML      func(string) string
}

func NewService(repo bindingRepository, client memberStatusClient) *Service {
	return &Service{
		repo:         repo,
		client:       client,
		sanitizeHTML: func(content string) string { return content },
	}
}

type paidAccessKind string

const (
	paidAccessKindPremium  paidAccessKind = "premium"
	paidAccessKindResource paidAccessKind = "resource"
)

type paidAccessDecisionInput struct {
	Kind          paidAccessKind
	LoggedIn      bool
	PremiumReady  bool
	AccessGranted bool
	HasPurchase   bool
	MemberFree    bool
	UserIsMember  bool
}

func (s *Service) buildPaidAccessDecision(input paidAccessDecisionInput) PaidAccessDecisionDTO {
	if !input.LoggedIn {
		return PaidAccessDecisionDTO{State: PaidAccessStateLoginRequired, Allowed: false}
	}

	if input.Kind == paidAccessKindPremium {
		if input.PremiumReady {
			return PaidAccessDecisionDTO{State: PaidAccessStateAllowed, Allowed: true}
		}
		return PaidAccessDecisionDTO{State: PaidAccessStateMemberRequired, Allowed: false}
	}

	if input.AccessGranted || input.HasPurchase || (input.MemberFree && input.UserIsMember) {
		return PaidAccessDecisionDTO{State: PaidAccessStateAllowed, Allowed: true}
	}

	return PaidAccessDecisionDTO{State: PaidAccessStatePurchaseRequired, Allowed: false}
}

func paidAccessReasonFromState(state PaidAccessState) string {
	switch state {
	case PaidAccessStateAllowed:
		return "allowed"
	case PaidAccessStateLoginRequired:
		return "login_required"
	case PaidAccessStateMemberRequired:
		return "member_required"
	case PaidAccessStatePurchaseRequired:
		return "purchase_required"
	case PaidAccessStateNotFound:
		return "not_found"
	default:
		return "unavailable"
	}
}

func (s *Service) buildResourceAccessCheckDTO(
	decision PaidAccessDecisionDTO,
	resource ResourceMetaDTO,
	pricing resourcePricing,
	resourceItems []ResourceAccessItemDTO,
	userIsMember bool,
	alreadyPurchased bool,
	reason string,
) ResourceAccessCheckDTO {
	requiresLogin := decision.State == PaidAccessStateLoginRequired
	requiresPurchase := decision.State == PaidAccessStatePurchaseRequired
	payable := requiresPurchase
	price := pricing.Price
	items := []ResourceAccessItemDTO(nil)
	if decision.Allowed {
		items = resourceItems
	}
	if decision.Allowed {
		price = 0
	}

	return ResourceAccessCheckDTO{
		AccessGranted:    decision.Allowed,
		Reason:           reason,
		RequiresLogin:    requiresLogin,
		RequiresPurchase: requiresPurchase,
		MemberFree:       pricing.MemberFree,
		UserIsMember:     userIsMember,
		AlreadyPurchased: alreadyPurchased,
		Price:            price,
		OriginalPrice:    pricing.OriginalPrice,
		BusinessType:     "resource_purchase",
		BusinessPreview:  s.buildResourceOrderPreview(resource, pricing),
		Payable:          payable,
		ResourceMeta:     resource,
		ResourceItems:    items,
	}
}

func (s *Service) SetResourceRepositories(resourceRepo resourceRepository, resourceOrderRepo resourceOrderRepository) {
	s.resourceRepo = resourceRepo
	s.resourceOrderRepo = resourceOrderRepo
}

func (s *Service) SetMemberZoneRepository(memberZoneRepo memberZoneRepository) {
	s.memberZoneRepo = memberZoneRepo
}

func (s *Service) SetHTMLSanitizer(sanitizer func(string) string) {
	if sanitizer == nil {
		s.sanitizeHTML = func(content string) string { return content }
		return
	}
	s.sanitizeHTML = sanitizer
}

var (
	ErrMemberBindingNotFound       = errors.New("member binding not found")
	ErrResourceNotFound            = errors.New("resource not found")
	ErrResourceUnavailable         = errors.New("resource unavailable")
	ErrResourcePurchaseNotRequired = errors.New("resource purchase not required")
	ErrInvalidResourceLocator      = errors.New("invalid resource locator")
	ErrResourceLocatorRequired     = errors.New("resource locator required")
	ErrResourceOrderNotFound       = errors.New("resource order not found")
	ErrResourceBoundToArticle      = errors.New("resource bound to article")
	ErrResourceHasOrders           = errors.New("resource has orders")
	ErrMemberZoneNotFound          = errors.New("member zone not found")
	ErrMemberZoneUnavailable       = errors.New("member zone unavailable")
	ErrMemberZoneAccessDenied      = errors.New("member zone access denied")
	ErrMemberZoneInvalidInput      = errors.New("member zone invalid input")
	ErrMemberZoneConflict          = errors.New("member zone conflict")
)

func (s *Service) AutoBindAfterLogin(ctx context.Context, userID int64, publicUserID string) (string, error) {
	if publicUserID == "" {
		return "", nil
	}

	binding, found, err := s.findMemberBinding(ctx, userID)
	if err != nil {
		return "", err
	}
	if found && !shouldRefreshMemberBinding(binding, publicUserID) {
		return binding.ExternalUserID, nil
	}

	result, err := s.client.EnsureUserMapping(ctx, dp7575.UserMapEnsureRequest{ExternalUserID: publicUserID})
	if err != nil {
		if errors.Is(err, dp7575.ErrNotConfigured) {
			return publicUserID, nil
		}
		return "", err
	}
	if !result.IsMapped {
		return publicUserID, nil
	}

	now := time.Now()
	if err := s.repo.Upsert(ctx, MemberBindingDTO{
		UserID:         userID,
		ExternalUserID: result.ExternalUserID,
		SiteID:         result.SiteID,
		Status:         "active",
		LastSyncedAt:   &now,
	}); err != nil {
		return "", err
	}

	return result.ExternalUserID, nil
}

func (s *Service) findMemberBinding(ctx context.Context, userID int64) (MemberBindingDTO, bool, error) {
	binding, err := s.repo.FindByUserID(ctx, userID)
	if err == nil {
		return binding, true, nil
	}
	if errors.Is(err, ErrMemberBindingNotFound) {
		return MemberBindingDTO{}, false, nil
	}
	return MemberBindingDTO{}, false, err
}

func shouldRefreshMemberBinding(binding MemberBindingDTO, publicUserID string) bool {
	if strings.TrimSpace(binding.ExternalUserID) == "" || strings.TrimSpace(binding.SiteID) == "" {
		return true
	}
	return publicUserID != "" && binding.ExternalUserID == publicUserID
}

func memberBindingHint(userID int64) string {
	return fmt.Sprintf("当前账号尚未完成会员映射，请先使用 external_user_id=%s 建立映射", memberExternalUserID(userID))
}

func (s *Service) requireMemberBinding(ctx context.Context, userID int64) (MemberBindingDTO, error) {
	binding, found, err := s.findMemberBinding(ctx, userID)
	if err != nil {
		return MemberBindingDTO{}, err
	}
	if !found {
		return MemberBindingDTO{}, errors.New(memberBindingHint(userID))
	}
	return binding, nil
}

func (s *Service) GetMemberStatus(ctx context.Context, userID int64) (MemberStatusDTO, error) {
	binding, found, err := s.findMemberBinding(ctx, userID)
	if err != nil {
		return MemberStatusDTO{}, err
	}
	if !found {
		return MemberStatusDTO{
			IsMember: false,
			State:    "pending",
			Message:  memberBindingHint(userID),
		}, nil
	}

	status, err := s.client.MemberStatus(ctx, dp7575.MemberStatusRequest{
		ExternalUserID: binding.ExternalUserID,
		SiteID:         binding.SiteID,
	})
	if err != nil {
		return MemberStatusDTO{}, err
	}

	return MemberStatusDTO{
		IsMember:  status.IsMember,
		Level:     normalizeMemberLevel(status),
		ExpiresAt: status.MemberExpireAt,
		State:     "ready",
	}, nil
}

func (s *Service) ListAdminOrderMappings(ctx context.Context, query AdminOrderMappingListQueryDTO) (AdminOrderMappingListDTO, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	resp, err := s.client.AdminOrderMappings(ctx, dp7575.AdminOrderMappingListRequest{
		SiteID:      query.SiteID,
		ZibOrderNum: query.ZibOrderNum,
		Page:        query.Page,
		PageSize:    query.PageSize,
	})
	if err != nil {
		return AdminOrderMappingListDTO{}, err
	}

	items := make([]AdminOrderMappingItemDTO, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, AdminOrderMappingItemDTO{
			ZibOrderNum:     item.ZibOrderNum,
			ExternalUserID:  item.ExternalUserID,
			WpUserID:        item.WpUserID,
			ProductType:     item.ProductType,
			PostID:          item.PostID,
			ResourceID:      item.ResourceID,
			CreatedAt:       item.CreatedAt,
			StoredSiteID:    item.StoredSiteID,
			ResolvedSiteID:  item.ResolvedSiteID,
			SnapshotSiteID:  item.SnapshotSiteID,
			ContextSource:   item.ContextSource,
			IsDirty:         item.IsDirty,
			RequestSnapshot: item.RequestSnapshot,
		})
	}

	return AdminOrderMappingListDTO{
		Summary: AdminOrderMappingSummaryDTO{
			Total:           resp.Summary.Total,
			SiteIDZeroCount: resp.Summary.SiteIDZeroCount,
			LatestCreatedAt: resp.Summary.LatestCreatedAt,
		},
		List: items,
		Pagination: AdminOrderMappingPaginationDTO{
			Page:     resp.Pagination.Page,
			PageSize: resp.Pagination.PageSize,
			Total:    resp.Pagination.Total,
		},
	}, nil
}

func (s *Service) ListAdminCards(ctx context.Context, query AdminCardListQueryDTO) (AdminCardListDTO, error) {
	resp, err := s.client.AdminCards(ctx, dp7575.AdminCardListRequest{
		CardType: query.CardType,
		Status:   query.Status,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		return AdminCardListDTO{}, err
	}

	items := make([]AdminCardItemDTO, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, AdminCardItemDTO{
			CardCode:     item.CardCode,
			CardPassword: item.CardPassword,
			CardType:     item.CardType,
			Status:       item.Status,
			Remark:       item.Remark,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}

	return AdminCardListDTO{
		List: items,
		Pagination: AdminOrderMappingPaginationDTO{
			Page:     resp.Pagination.Page,
			PageSize: resp.Pagination.PageSize,
			Total:    resp.Pagination.Total,
		},
	}, nil
}

type resourcePricing struct {
	Price         float64
	OriginalPrice float64
	MemberFree    bool
}

func (s *Service) ListAdminResources(ctx context.Context, query AdminResourceListQueryDTO) (AdminResourceListDTO, error) {
	return s.resourceRepo.ListAdminResources(ctx, query)
}

func (s *Service) GetAdminResourceDetail(ctx context.Context, resourceID string) (AdminResourceDetailDTO, error) {
	return s.resourceRepo.GetAdminResourceDetail(ctx, resourceID)
}

func (s *Service) CreateAdminResource(ctx context.Context, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error) {
	if err := validateAdminResourceInput(input); err != nil {
		return AdminResourceDetailDTO{}, err
	}
	return s.resourceRepo.CreateAdminResource(ctx, input)
}

func (s *Service) UpdateAdminResource(ctx context.Context, resourceID string, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error) {
	if err := validateAdminResourceInput(input); err != nil {
		return AdminResourceDetailDTO{}, err
	}
	return s.resourceRepo.UpdateAdminResource(ctx, resourceID, input)
}

func (s *Service) BindAdminResourceToArticle(ctx context.Context, resourceID string, articleID string) (AdminResourceDetailDTO, error) {
	if strings.TrimSpace(resourceID) == "" || strings.TrimSpace(articleID) == "" {
		return AdminResourceDetailDTO{}, fmt.Errorf("resource id and article id are required")
	}
	return s.resourceRepo.BindAdminResourceToArticle(ctx, resourceID, articleID)
}

func (s *Service) SearchAdminArticleHosts(ctx context.Context, query string) ([]AdminArticleHostOptionDTO, error) {
	return s.resourceRepo.SearchArticleHosts(ctx, query)
}

func (s *Service) GetAdminResourceByArticle(ctx context.Context, articleID string) (AdminResourceDetailDTO, error) {
	return s.resourceRepo.FindResourceByArticleHost(ctx, articleID)
}

func (s *Service) DeleteAdminResource(ctx context.Context, resourceID string) error {
	if strings.TrimSpace(resourceID) == "" {
		return ErrResourceNotFound
	}
	detail, err := s.resourceRepo.GetAdminResourceDetail(ctx, resourceID)
	if err != nil {
		return err
	}
	if detail.HostID != "" || detail.HostType != "" {
		return ErrResourceBoundToArticle
	}
	count, err := s.resourceRepo.CountResourceOrders(ctx, resourceID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrResourceHasOrders
	}
	return s.resourceRepo.DeleteAdminResource(ctx, resourceID)
}

func validateAdminResourceInput(input AdminResourceDetailDTO) error {
	if input.Title == "" {
		return fmt.Errorf("resource title is required")
	}
	if input.Price < 0 || input.OriginalPrice < 0 {
		return fmt.Errorf("resource price must be non-negative")
	}
	for _, item := range input.Items {
		if item.Title == "" {
			return fmt.Errorf("resource item title is required")
		}
		if item.ItemType == "link" && item.URL == "" {
			return fmt.Errorf("resource item url is required")
		}
	}
	if input.HostType != "" && input.HostType != "article" {
		return fmt.Errorf("resource host type must be article")
	}
	return nil
}

func (s *Service) ListAdminMemberZones(ctx context.Context, query AdminMemberZoneListQueryDTO) (AdminMemberZoneListDTO, error) {
	if s.memberZoneRepo == nil {
		return AdminMemberZoneListDTO{}, ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.ListAdminMemberZones(ctx, query)
}

func (s *Service) GetAdminMemberZoneDetail(ctx context.Context, contentID string) (AdminMemberZoneDetailDTO, error) {
	if s.memberZoneRepo == nil {
		return AdminMemberZoneDetailDTO{}, ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.GetAdminMemberZoneDetail(ctx, contentID)
}

func (s *Service) CreateAdminMemberZone(ctx context.Context, input AdminMemberZoneDetailDTO) (AdminMemberZoneDetailDTO, error) {
	sanitizedInput, err := s.prepareAdminMemberZoneInput(ctx, input)
	if err != nil {
		return AdminMemberZoneDetailDTO{}, err
	}
	if s.memberZoneRepo == nil {
		return AdminMemberZoneDetailDTO{}, ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.CreateAdminMemberZone(ctx, sanitizedInput)
}

func (s *Service) UpdateAdminMemberZone(ctx context.Context, contentID string, input AdminMemberZoneDetailDTO) (AdminMemberZoneDetailDTO, error) {
	if strings.TrimSpace(contentID) == "" {
		return AdminMemberZoneDetailDTO{}, ErrMemberZoneNotFound
	}
	if s.memberZoneRepo == nil {
		return AdminMemberZoneDetailDTO{}, ErrMemberZoneNotFound
	}
	sanitizedInput, err := s.prepareAdminMemberZoneInput(ctx, input)
	if err != nil {
		return AdminMemberZoneDetailDTO{}, err
	}
	return s.memberZoneRepo.UpdateAdminMemberZone(ctx, contentID, sanitizedInput)
}

func (s *Service) DeleteAdminMemberZone(ctx context.Context, contentID string) error {
	if strings.TrimSpace(contentID) == "" {
		return ErrMemberZoneNotFound
	}
	if s.memberZoneRepo == nil {
		return ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.DeleteAdminMemberZone(ctx, contentID)
}

func (s *Service) GetAdminMemberZoneByArticle(ctx context.Context, articleID string) (*AdminMemberZoneDetailDTO, error) {
	if strings.TrimSpace(articleID) == "" || s.memberZoneRepo == nil {
		return nil, nil
	}
	detail, err := s.memberZoneRepo.FindAdminMemberZoneByArticle(ctx, articleID)
	if errors.Is(err, ErrMemberZoneNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (s *Service) prepareAdminMemberZoneInput(ctx context.Context, input AdminMemberZoneDetailDTO) (AdminMemberZoneDetailDTO, error) {
	if err := validateAdminMemberZoneInput(input); err != nil {
		return AdminMemberZoneDetailDTO{}, err
	}
	if input.SourceArticleID != "" && s.resourceRepo != nil {
		exists, err := s.resourceRepo.ArticleHostExists(ctx, input.SourceArticleID)
		if err != nil {
			return AdminMemberZoneDetailDTO{}, err
		}
		if !exists {
			return AdminMemberZoneDetailDTO{}, fmt.Errorf("%w: member zone article id does not exist", ErrMemberZoneInvalidInput)
		}
	}

	input.ContentHTML = strings.TrimSpace(s.sanitizeHTML(input.ContentHTML))
	if input.ContentHTML == "" {
		return AdminMemberZoneDetailDTO{}, fmt.Errorf("%w: member zone html content is invalid", ErrMemberZoneInvalidInput)
	}

	return input, nil
}

func validateAdminMemberZoneInput(input AdminMemberZoneDetailDTO) error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("%w: member zone title is required", ErrMemberZoneInvalidInput)
	}
	if strings.TrimSpace(input.Slug) == "" {
		return fmt.Errorf("%w: member zone slug is required", ErrMemberZoneInvalidInput)
	}
	if strings.TrimSpace(input.ContentMD) == "" {
		return fmt.Errorf("%w: member zone markdown content is required", ErrMemberZoneInvalidInput)
	}
	if strings.TrimSpace(input.ContentHTML) == "" {
		return fmt.Errorf("%w: member zone html content is required", ErrMemberZoneInvalidInput)
	}
	switch input.Status {
	case "draft", "published", "archived":
	default:
		return fmt.Errorf("%w: member zone status is invalid", ErrMemberZoneInvalidInput)
	}
	switch input.AccessLevel {
	case "member", "premium":
	default:
		return fmt.Errorf("%w: member zone access level is invalid", ErrMemberZoneInvalidInput)
	}
	if input.SourceArticleID != "" {
		_, entityType, err := idgen.DecodePublicID(input.SourceArticleID)
		if err != nil {
			return fmt.Errorf("%w: member zone article id is invalid", ErrMemberZoneInvalidInput)
		}
		if entityType != idgen.EntityTypeArticle {
			return fmt.Errorf("%w: member zone article id is invalid", ErrMemberZoneInvalidInput)
		}
	}
	return nil
}

func (s *Service) ListPublishedMemberZones(ctx context.Context) ([]MemberZoneListItemDTO, error) {
	if s.memberZoneRepo == nil {
		return nil, nil
	}
	return s.memberZoneRepo.ListPublishedMemberZones(ctx)
}

func (s *Service) GetPublishedMemberZoneMetaBySlug(ctx context.Context, slug string) (MemberZoneMetaDTO, error) {
	if strings.TrimSpace(slug) == "" || s.memberZoneRepo == nil {
		return MemberZoneMetaDTO{}, ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.GetPublishedMemberZoneMetaBySlug(ctx, strings.TrimSpace(slug))
}

func (s *Service) GetPublishedMemberZoneByArticle(ctx context.Context, articleID string) (*MemberZoneMetaDTO, error) {
	if strings.TrimSpace(articleID) == "" || s.memberZoneRepo == nil {
		return nil, nil
	}
	meta, err := s.memberZoneRepo.GetPublishedMemberZoneByArticle(ctx, articleID)
	if errors.Is(err, ErrMemberZoneNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Service) CheckMemberZoneAccess(ctx context.Context, actor *ResourceAccessCheckActorDTO, slug string) (MemberZoneAccessCheckDTO, error) {
	meta, err := s.GetPublishedMemberZoneMetaBySlug(ctx, slug)
	if err != nil {
		return MemberZoneAccessCheckDTO{}, err
	}

	if actor == nil || !actor.LoggedIn {
		return MemberZoneAccessCheckDTO{
			Allowed:        false,
			Reason:         "login_required",
			RequiresLogin:  true,
			RequiresMember: false,
			RequiredLevel:  meta.AccessLevel,
			MemberZoneMeta: meta,
		}, nil
	}

	identity, err := s.resolvePaidIdentity(ctx, actor)
	if err != nil {
		return MemberZoneAccessCheckDTO{}, err
	}
	if !identity.BindingReady {
		return MemberZoneAccessCheckDTO{
			Allowed:        false,
			Reason:         "member_required",
			RequiresLogin:  false,
			RequiresMember: true,
			RequiredLevel:  meta.AccessLevel,
			MemberZoneMeta: meta,
		}, nil
	}

	status, err := s.client.MemberStatus(ctx, dp7575.MemberStatusRequest{
		ExternalUserID: identity.ExternalUserID,
		SiteID:         identity.SiteID,
	})
	if err != nil {
		return MemberZoneAccessCheckDTO{}, err
	}

	userLevel := normalizeMemberLevel(status)
	if !status.IsMember {
		return MemberZoneAccessCheckDTO{
			Allowed:        false,
			Reason:         "member_required",
			RequiresLogin:  false,
			RequiresMember: true,
			RequiredLevel:  meta.AccessLevel,
			UserIsMember:   false,
			UserLevel:      userLevel,
			MemberZoneMeta: meta,
		}, nil
	}

	if meta.AccessLevel == "premium" && memberLevelNumber(status) < 2 {
		return MemberZoneAccessCheckDTO{
			Allowed:        false,
			Reason:         "member_required",
			RequiresLogin:  false,
			RequiresMember: true,
			RequiredLevel:  meta.AccessLevel,
			UserIsMember:   true,
			UserLevel:      userLevel,
			MemberZoneMeta: meta,
		}, nil
	}

	return MemberZoneAccessCheckDTO{
		Allowed:        true,
		Reason:         "allowed",
		RequiresLogin:  false,
		RequiresMember: false,
		RequiredLevel:  meta.AccessLevel,
		UserIsMember:   true,
		UserLevel:      userLevel,
		MemberZoneMeta: meta,
	}, nil
}

func (s *Service) GetMemberZoneContentForActor(ctx context.Context, actor *ResourceAccessCheckActorDTO, slug string) (MemberZoneContentDTO, error) {
	access, err := s.CheckMemberZoneAccess(ctx, actor, slug)
	if err != nil {
		return MemberZoneContentDTO{}, err
	}
	if !access.Allowed {
		if access.RequiresLogin {
			return MemberZoneContentDTO{}, ErrMemberZoneUnavailable
		}
		return MemberZoneContentDTO{}, ErrMemberZoneAccessDenied
	}
	if s.memberZoneRepo == nil {
		return MemberZoneContentDTO{}, ErrMemberZoneNotFound
	}
	return s.memberZoneRepo.GetPublishedMemberZoneContentBySlug(ctx, strings.TrimSpace(slug))
}

func (s *Service) CheckResourceAccess(ctx context.Context, actor *ResourceAccessCheckActorDTO, input ResourceAccessCheckRequestDTO) (ResourceAccessCheckDTO, error) {
	resourceRecord, err := s.resolveResource(ctx, input)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) && input.ResourceID == "" {
			return ResourceAccessCheckDTO{
				AccessGranted:    false,
				Reason:           "resource_not_found",
				RequiresLogin:    false,
				RequiresPurchase: false,
				MemberFree:       false,
				UserIsMember:     false,
				AlreadyPurchased: false,
				Price:            0,
				OriginalPrice:    0,
				BusinessType:     "resource_purchase",
				Payable:          false,
			}, nil
		}
		return ResourceAccessCheckDTO{}, err
	}
	if resourceRecord.Status != "published" || !resourceRecord.SaleEnabled {
		return ResourceAccessCheckDTO{}, ErrResourceUnavailable
	}

	resource := ResourceMetaDTO{ResourceID: resourceRecord.ResourceID, Title: resourceRecord.Title, Type: resourceRecord.ResourceType}
	pricing := resourcePricing{Price: resourceRecord.Price, OriginalPrice: resourceRecord.OriginalPrice, MemberFree: resourceRecord.MemberFree}
	resourceItems := resourceRecord.ResourceItems

	if pricing.Price <= 0 {
		return s.buildResourceAccessCheckDTO(
			PaidAccessDecisionDTO{State: PaidAccessStateAllowed, Allowed: true},
			resource,
			pricing,
			resourceItems,
			false,
			false,
			"free",
		), nil
	}

	if actor == nil || !actor.LoggedIn {
		decision := s.buildPaidAccessDecision(paidAccessDecisionInput{Kind: paidAccessKindResource, LoggedIn: false})
		return s.buildResourceAccessCheckDTO(decision, resource, pricing, nil, false, false, paidAccessReasonFromState(decision.State)), nil
	}

	hasGrant, err := s.hasActiveResourceGrant(ctx, actor.UserID, resource.ResourceID, "")
	if err != nil {
		return ResourceAccessCheckDTO{}, err
	}
	if hasGrant {
		decision := s.buildPaidAccessDecision(paidAccessDecisionInput{Kind: paidAccessKindResource, LoggedIn: true, AccessGranted: true})
		return s.buildResourceAccessCheckDTO(decision, resource, pricing, resourceItems, false, true, "already_purchased"), nil
	}

	binding, err := s.resolveResourceAccessBinding(ctx, actor)
	if err != nil {
		return ResourceAccessCheckDTO{}, err
	}

	memberStatus, err := s.client.MemberStatus(ctx, dp7575.MemberStatusRequest{ExternalUserID: binding.ExternalUserID, SiteID: binding.SiteID})
	if err != nil {
		return ResourceAccessCheckDTO{}, err
	}

	if pricing.MemberFree && memberStatus.IsMember {
		decision := s.buildPaidAccessDecision(paidAccessDecisionInput{Kind: paidAccessKindResource, LoggedIn: true, MemberFree: true, UserIsMember: true})
		return s.buildResourceAccessCheckDTO(decision, resource, pricing, resourceItems, true, false, "member_free"), nil
	}

	hasPurchased := s.hasPurchasedResource(ctx, binding, resource.ResourceID)
	if hasPurchased {
		decision := s.buildPaidAccessDecision(paidAccessDecisionInput{Kind: paidAccessKindResource, LoggedIn: true, HasPurchase: true, UserIsMember: memberStatus.IsMember, MemberFree: pricing.MemberFree})
		return s.buildResourceAccessCheckDTO(decision, resource, pricing, resourceItems, memberStatus.IsMember, true, "already_purchased"), nil
	}

	decision := s.buildPaidAccessDecision(paidAccessDecisionInput{Kind: paidAccessKindResource, LoggedIn: true, HasPurchase: false, MemberFree: pricing.MemberFree, UserIsMember: memberStatus.IsMember})
	return s.buildResourceAccessCheckDTO(decision, resource, pricing, nil, memberStatus.IsMember, false, paidAccessReasonFromState(decision.State)), nil
}

func (s *Service) GetMemberProfile(ctx context.Context, userID int64) (MemberProfileDTO, error) {
	binding, found, err := s.findMemberBinding(ctx, userID)
	if err != nil {
		return MemberProfileDTO{}, err
	}
	if !found {
		return MemberProfileDTO{
			IsMember: false,
			State:    "pending",
			Message:  memberBindingHint(userID),
		}, nil
	}

	profile, err := s.client.MemberProfile(ctx, dp7575.MemberProfileRequest{
		ExternalUserID: binding.ExternalUserID,
		SiteID:         binding.SiteID,
	})
	if err != nil {
		return MemberProfileDTO{}, err
	}

	orders := make([]MemberOrderSummaryDTO, 0, 1)
	if profile.HistorySummary.LatestOrderNo != "" {
		orders = append(orders, MemberOrderSummaryDTO{
			OrderNo:   profile.HistorySummary.LatestOrderNo,
			Status:    profile.HistorySummary.LatestOrderStatus,
			Amount:    profile.HistorySummary.LatestOrderAmount,
			CreatedAt: profile.HistorySummary.LatestOrderTime,
		})
	}

	return MemberProfileDTO{
		IsMember:     profile.IsMember,
		Level:        normalizeMemberLevel(dp7575.MemberStatusResponse{MemberLevel: profile.MemberLevel, MemberLevelName: profile.MemberLevelName, MemberExpireAt: profile.MemberExpireAt}),
		ExpiresAt:    profile.MemberExpireAt,
		State:        "ready",
		RecentOrders: orders,
	}, nil
}

func (s *Service) GetMemberPurchaseCatalog(ctx context.Context, userID int64) (MemberPurchaseCatalogDTO, error) {
	if _, err := s.requireMemberBinding(ctx, userID); err != nil {
		return MemberPurchaseCatalogDTO{}, err
	}

	catalog, err := s.client.MemberProductsCatalog(ctx)
	if err != nil {
		return MemberPurchaseCatalogDTO{}, err
	}

	grouped := make(map[int]*MemberPurchaseMemberTypeDTO)
	levels := make([]int, 0)
	for _, product := range catalog.Products {
		group, exists := grouped[product.MemberLevel]
		if !exists {
			group = &MemberPurchaseMemberTypeDTO{
				Level: strconv.Itoa(product.MemberLevel),
				Name:  product.MemberLevelName,
			}
			grouped[product.MemberLevel] = group
			levels = append(levels, product.MemberLevel)
		}

		group.PriceOptions = append(group.PriceOptions, MemberPurchasePriceOptionDTO{
			ProductID:     product.ProductID,
			Title:         product.Title,
			Description:   product.Description,
			Price:         product.Price,
			OriginalPrice: product.OriginalPrice,
			ActionType:    product.ActionType,
			Tag:           product.Meta.Tag,
		})
	}

	sort.Ints(levels)
	memberTypes := make([]MemberPurchaseMemberTypeDTO, 0, len(levels))
	for _, level := range levels {
		group := grouped[level]
		sort.SliceStable(group.PriceOptions, func(i, j int) bool {
			return memberPurchaseActionRank(group.PriceOptions[i].ActionType) < memberPurchaseActionRank(group.PriceOptions[j].ActionType)
		})
		memberTypes = append(memberTypes, *group)
	}

	return MemberPurchaseCatalogDTO{
		MemberTypes:    memberTypes,
		PaymentMethods: []string{"wechat", "alipay", "card"},
	}, nil
}

func (s *Service) CreateMemberOrder(ctx context.Context, userID int64, input MemberOrderCreateDTO) (MemberOrderDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberOrderDTO{}, err
	}

	result, err := s.client.CreateOrder(ctx, dp7575.OrderCreateRequest{
		ExternalUserID: binding.ExternalUserID,
		ProductType:    "vip",
		ProductID:      input.ProductID,
		PaymentMethod:  input.PaymentMethod,
	})
	if err != nil {
		return MemberOrderDTO{}, err
	}

	return MemberOrderDTO{
		OrderNo:    result.ZibOrderNum,
		Status:     result.OrderStatus,
		PayType:    result.PayType,
		PayURL:     result.PayURL,
		OrderPrice: result.OrderPrice,
		CreatedAt:  result.CreatedAt,
	}, nil
}

func (s *Service) CreateResourcePurchaseOrder(ctx context.Context, userID int64, input ResourceAccessCheckRequestDTO) (ResourcePurchaseOrderDTO, error) {
	return s.CreateResourcePurchaseOrderForActor(ctx, &ResourceAccessCheckActorDTO{UserID: userID, ExternalUserID: memberExternalUserID(userID), LoggedIn: true}, input)
}

func (s *Service) CreateResourcePurchaseOrderForActor(ctx context.Context, actor *ResourceAccessCheckActorDTO, input ResourceAccessCheckRequestDTO) (ResourcePurchaseOrderDTO, error) {
	if s.resourceOrderRepo == nil {
		return ResourcePurchaseOrderDTO{}, fmt.Errorf("resource order repository not configured")
	}

	access, err := s.CheckResourceAccess(ctx, actor, input)
	if err != nil {
		return ResourcePurchaseOrderDTO{}, err
	}
	if access.AccessGranted || !access.RequiresPurchase || !access.Payable {
		return ResourcePurchaseOrderDTO{}, ErrResourcePurchaseNotRequired
	}
	binding, err := s.resolveResourceAccessBinding(ctx, actor)
	if err != nil {
		publicUserID := ""
		if actor != nil {
			publicUserID = memberExternalUserID(actor.UserID)
		}
		return ResourcePurchaseOrderDTO{}, fmt.Errorf("当前账号尚未完成会员映射，请先使用 external_user_id=%s 建立映射", publicUserID)
	}

	resourceRecord, err := s.resolveResource(ctx, input)
	if err != nil {
		return ResourcePurchaseOrderDTO{}, err
	}
	if resourceRecord.Status != "published" || !resourceRecord.SaleEnabled || resourceRecord.Price <= 0 {
		return ResourcePurchaseOrderDTO{}, ErrResourceUnavailable
	}

	pendingOrder, err := s.resourceOrderRepo.FindLatestPendingByUserAndResource(ctx, actor.UserID, resourceRecord.ResourceID)
	if err != nil && !errors.Is(err, ErrResourceOrderNotFound) {
		return ResourcePurchaseOrderDTO{}, err
	}
	if err == nil && pendingOrder.ExternalOrderNo != "" {
		return ResourcePurchaseOrderDTO{
			BusinessOrderNo: pendingOrder.BusinessOrderNo,
			PayURL:          "",
			Amount:          pendingOrder.Amount,
			ResourceID:      pendingOrder.ResourceID,
		}, nil
	}

	businessOrderNo := fmt.Sprintf("YGZ_RES_%d", time.Now().UnixNano())
	orderAmount := resourceRecord.Price
	snapshot := buildResourcePurchaseSnapshot(resourceRecord)
	if err == nil {
		businessOrderNo = pendingOrder.BusinessOrderNo
		orderAmount = pendingOrder.Amount
		if len(pendingOrder.Snapshot) > 0 {
			snapshot = pendingOrder.Snapshot
		}
	}
	attach, _ := snapshot["product"].(map[string]any)

	if err != nil {
		_, err = s.resourceOrderRepo.Create(ctx, ResourceOrderCreateDTO{
			UserID:          actor.UserID,
			ResourceID:      resourceRecord.ResourceID,
			BusinessOrderNo: businessOrderNo,
			Amount:          orderAmount,
			Status:          "pending",
			Snapshot:        snapshot,
		})
		if err != nil {
			return ResourcePurchaseOrderDTO{}, err
		}
	}

	result, err := s.client.CreateOrder(ctx, dp7575.OrderCreateRequest{
		ExternalUserID:  binding.ExternalUserID,
		PaymentMethod:   normalizeResourcePaymentMethod(input.PaymentMethod),
		BusinessType:    "resource_purchase",
		BusinessOrderNo: businessOrderNo,
		Subject:         "资源购买：" + resourceRecord.Title,
		Amount:          orderAmount,
		Attach:          attach,
	})
	if err != nil {
		return ResourcePurchaseOrderDTO{}, err
	}
	if result.ZibOrderNum != "" {
		if err := s.resourceOrderRepo.UpdateExternalOrderNo(ctx, businessOrderNo, result.ZibOrderNum); err != nil {
			return ResourcePurchaseOrderDTO{}, err
		}
	}

	return ResourcePurchaseOrderDTO{
		BusinessOrderNo: businessOrderNo,
		PayURL:          result.PayURL,
		Amount:          orderAmount,
		ResourceID:      resourceRecord.ResourceID,
	}, nil
}

func (s *Service) MarkResourceOrderPaid(ctx context.Context, businessOrderNo string, externalOrderNo string) error {
	if s.resourceRepo == nil || s.resourceOrderRepo == nil {
		return fmt.Errorf("resource repositories not configured")
	}

	order, err := s.resourceOrderRepo.FindByBusinessOrderNo(ctx, businessOrderNo)
	if err != nil {
		return err
	}

	now := time.Now()
	updated, err := s.resourceOrderRepo.MarkPaid(ctx, businessOrderNo, externalOrderNo, &now)
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	hasGrant, err := s.resourceRepo.HasGrantBySourceOrderNo(ctx, order.UserID, businessOrderNo)
	if err != nil {
		return err
	}
	if hasGrant {
		return nil
	}

	return s.resourceRepo.CreateGrant(ctx, ResourceAccessGrantCreateDTO{
		UserID:         order.UserID,
		ResourceID:     order.ResourceID,
		ResourceItemID: order.ResourceItemID,
		GrantType:      "purchase",
		SourceOrderNo:  businessOrderNo,
		Status:         "active",
		GrantedAt:      &now,
	})
}

func (s *Service) GetResourcePurchaseOrderStatus(ctx context.Context, userID int64, businessOrderNo string) (ResourcePurchaseOrderStatusDTO, error) {
	return s.GetResourcePurchaseOrderStatusForActor(ctx, &ResourceAccessCheckActorDTO{UserID: userID, ExternalUserID: memberExternalUserID(userID), LoggedIn: true}, businessOrderNo)
}

func (s *Service) GetResourcePurchaseOrderStatusForActor(ctx context.Context, actor *ResourceAccessCheckActorDTO, businessOrderNo string) (ResourcePurchaseOrderStatusDTO, error) {
	if s.resourceOrderRepo == nil {
		return ResourcePurchaseOrderStatusDTO{}, fmt.Errorf("resource order repository not configured")
	}
	order, err := s.resourceOrderRepo.FindByBusinessOrderNo(ctx, businessOrderNo)
	if err != nil {
		return ResourcePurchaseOrderStatusDTO{}, err
	}
	if actor == nil || order.UserID != actor.UserID {
		return ResourcePurchaseOrderStatusDTO{}, ErrResourceOrderNotFound
	}
	if order.ExternalOrderNo == "" {
		return ResourcePurchaseOrderStatusDTO{BusinessOrderNo: order.BusinessOrderNo, ResourceID: order.ResourceID, Status: order.Status}, nil
	}
	binding, err := s.resolveResourceAccessBinding(ctx, actor)
	if err != nil {
		publicUserID := memberExternalUserID(actor.UserID)
		return ResourcePurchaseOrderStatusDTO{}, fmt.Errorf("当前账号尚未完成会员映射，请先使用 external_user_id=%s 建立映射", publicUserID)
	}

	status, err := s.client.OrderStatus(ctx, dp7575.OrderStatusRequest{ExternalUserID: binding.ExternalUserID, ZibOrderNum: order.ExternalOrderNo})
	if err != nil {
		return ResourcePurchaseOrderStatusDTO{}, err
	}
	if status.OrderStatus == "paid" && order.Status != "paid" {
		if err := s.MarkResourceOrderPaid(ctx, order.BusinessOrderNo, order.ExternalOrderNo); err != nil {
			return ResourcePurchaseOrderStatusDTO{}, err
		}
	}

	return ResourcePurchaseOrderStatusDTO{
		BusinessOrderNo: order.BusinessOrderNo,
		ExternalOrderNo: order.ExternalOrderNo,
		ResourceID:      order.ResourceID,
		Status:          status.OrderStatus,
		StatusLabel:     status.OrderStatusLabel,
		OrderPrice:      status.OrderPrice,
		PayPrice:        status.PayPrice,
		PayType:         status.PayType,
		CreatedAt:       status.CreatedAt,
		PaidAt:          status.PaidAt,
	}, nil
}

func (s *Service) GetResourcePaymentDetail(ctx context.Context, userID int64, input ResourceOrderPaymentDetailDTO) (ResourcePaymentDetailDTO, error) {
	return s.GetResourcePaymentDetailForActor(ctx, &ResourceAccessCheckActorDTO{UserID: userID, ExternalUserID: memberExternalUserID(userID), LoggedIn: true}, input)
}

func (s *Service) GetResourcePaymentDetailForActor(ctx context.Context, actor *ResourceAccessCheckActorDTO, input ResourceOrderPaymentDetailDTO) (ResourcePaymentDetailDTO, error) {
	if s.resourceOrderRepo == nil {
		return ResourcePaymentDetailDTO{}, fmt.Errorf("resource order repository not configured")
	}
	order, err := s.resourceOrderRepo.FindByBusinessOrderNo(ctx, input.BusinessOrderNo)
	if err != nil {
		return ResourcePaymentDetailDTO{}, err
	}
	if actor == nil || order.UserID != actor.UserID {
		return ResourcePaymentDetailDTO{}, ErrResourceOrderNotFound
	}
	if order.ExternalOrderNo == "" {
		return ResourcePaymentDetailDTO{}, fmt.Errorf("resource external order not found")
	}
	binding, err := s.resolveResourceAccessBinding(ctx, actor)
	if err != nil {
		publicUserID := memberExternalUserID(actor.UserID)
		return ResourcePaymentDetailDTO{}, fmt.Errorf("当前账号尚未完成会员映射，请先使用 external_user_id=%s 建立映射", publicUserID)
	}

	result, err := s.client.OrderPaymentDetail(ctx, dp7575.OrderPaymentDetailRequest{
		ExternalUserID: binding.ExternalUserID,
		ZibOrderNum:    order.ExternalOrderNo,
	})
	if err != nil {
		return ResourcePaymentDetailDTO{}, err
	}

	return ResourcePaymentDetailDTO{
		BusinessOrderNo: order.BusinessOrderNo,
		ResourceID:      order.ResourceID,
		Amount:          result.Amount,
		PayType:         result.PayType,
		PayChannel:      result.PayChannel,
		PayTime:         result.PayTime,
		PayDetail:       result.PayDetail,
	}, nil
}

func (s *Service) GetMemberPaymentDetail(ctx context.Context, userID int64, input MemberOrderPaymentDetailDTO) (MemberPaymentDetailDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberPaymentDetailDTO{}, err
	}

	result, err := s.client.OrderPaymentDetail(ctx, dp7575.OrderPaymentDetailRequest{
		ExternalUserID: binding.ExternalUserID,
		ZibOrderNum:    input.OrderNo,
	})
	if err != nil {
		return MemberPaymentDetailDTO{}, err
	}

	return MemberPaymentDetailDTO{
		OrderNo:    result.OrderNum,
		Amount:     result.Amount,
		PayType:    result.PayType,
		PayChannel: result.PayChannel,
		PayTime:    result.PayTime,
		PayDetail:  result.PayDetail,
	}, nil
}

func (s *Service) RedeemMemberCard(ctx context.Context, userID int64, input MemberCardRedeemDTO) (MemberCardRedeemResultDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberCardRedeemResultDTO{}, err
	}

	result, err := s.client.CardRedeemCreate(ctx, dp7575.CardRedeemCreateRequest{
		ExternalUserID: binding.ExternalUserID,
		CardCode:       input.CardCode,
		CardPassword:   input.CardPassword,
	})
	if err != nil {
		return MemberCardRedeemResultDTO{}, err
	}

	return MemberCardRedeemResultDTO{
		Status:        result.RedeemStatus,
		TargetType:    result.TargetType,
		TargetSummary: result.TargetSummary,
		OrderNo:       result.OrderNum,
		Message:       result.EffectSummary,
	}, nil
}

func memberExternalUserID(userID int64) string {
	publicUserID, err := idgen.GeneratePublicID(uint(userID), idgen.EntityTypeUser)
	if err != nil {
		return fmt.Sprintf("user:%d", userID)
	}
	return publicUserID
}

func normalizeMemberLevel(status dp7575.MemberStatusResponse) string {
	if status.MemberLevelName != "" {
		return status.MemberLevelName
	}

	var text string
	if err := json.Unmarshal(status.MemberLevel, &text); err == nil {
		return text
	}

	var number int
	if err := json.Unmarshal(status.MemberLevel, &number); err == nil {
		return fmt.Sprintf("%d", number)
	}

	return ""
}

func memberLevelNumber(status dp7575.MemberStatusResponse) int {
	var text string
	if err := json.Unmarshal(status.MemberLevel, &text); err == nil {
		if n, convErr := strconv.Atoi(strings.TrimSpace(text)); convErr == nil {
			return n
		}
	}

	var number int
	if err := json.Unmarshal(status.MemberLevel, &number); err == nil {
		return number
	}

	return 0
}

func (s *Service) resolveResource(ctx context.Context, input ResourceAccessCheckRequestDTO) (ResourceRecordDTO, error) {
	if err := validateResourceLocator(input); err != nil {
		return ResourceRecordDTO{}, err
	}
	if s.resourceRepo == nil {
		return ResourceRecordDTO{}, ErrResourceNotFound
	}
	if input.ResourceID != "" {
		return s.resourceRepo.FindResourceByID(ctx, input.ResourceID)
	}
	if input.ArticleID != "" {
		return s.resourceRepo.FindResourceByHost(ctx, "article", input.ArticleID)
	}
	if input.Abbrlink != "" {
		articleID, err := s.resourceRepo.ResolveArticleIDByAbbrlink(ctx, input.Abbrlink)
		if err != nil {
			return ResourceRecordDTO{}, err
		}
		return s.resourceRepo.FindResourceByHost(ctx, "article", articleID)
	}
	return ResourceRecordDTO{}, ErrResourceNotFound
}

func (s *Service) buildResourceOrderPreview(meta ResourceMetaDTO, pricing resourcePricing) ResourceOrderPreviewDTO {
	return ResourceOrderPreviewDTO{
		Amount:       pricing.Price,
		Subject:      "资源购买：" + meta.Title,
		BusinessType: "resource_purchase",
		ResourceID:   meta.ResourceID,
	}
}

func (s *Service) hasActiveResourceGrant(ctx context.Context, userID int64, resourceID string, resourceItemID string) (bool, error) {
	if s.resourceRepo == nil {
		return false, nil
	}
	return s.resourceRepo.HasActiveGrant(ctx, userID, resourceID, resourceItemID)
}

func (s *Service) resolveResourceAccessBinding(ctx context.Context, actor *ResourceAccessCheckActorDTO) (MemberBindingDTO, error) {
	identity, err := s.resolvePaidIdentity(ctx, actor)
	if err != nil {
		return MemberBindingDTO{}, err
	}
	if !identity.BindingReady {
		return MemberBindingDTO{}, fmt.Errorf("当前账号尚未完成会员映射")
	}
	return MemberBindingDTO{UserID: identity.UserID, ExternalUserID: identity.ExternalUserID, SiteID: identity.SiteID, Status: "active"}, nil
}

func (s *Service) resolvePaidIdentity(ctx context.Context, actor *ResourceAccessCheckActorDTO) (PaidIdentityDTO, error) {
	if actor == nil || !actor.LoggedIn {
		return PaidIdentityDTO{}, nil
	}

	publicUserID := memberExternalUserID(actor.UserID)
	binding, found, err := s.findMemberBinding(ctx, actor.UserID)
	if err != nil {
		return PaidIdentityDTO{}, err
	}
	if found && !shouldRefreshMemberBinding(binding, publicUserID) {
		return PaidIdentityDTO{LoggedIn: true, UserID: actor.UserID, ExternalUserID: binding.ExternalUserID, SiteID: binding.SiteID, BindingReady: true}, nil
	}

	if actor.ExternalUserID == "" {
		return PaidIdentityDTO{LoggedIn: true, UserID: actor.UserID}, nil
	}

	binding, mapped, ensureErr := s.ensureMemberBinding(ctx, actor.UserID, actor.ExternalUserID)
	if ensureErr != nil {
		return PaidIdentityDTO{}, ensureErr
	}
	if !mapped {
		return PaidIdentityDTO{LoggedIn: true, UserID: actor.UserID, ExternalUserID: actor.ExternalUserID}, nil
	}

	return PaidIdentityDTO{LoggedIn: true, UserID: actor.UserID, ExternalUserID: binding.ExternalUserID, SiteID: binding.SiteID, BindingReady: true}, nil
}

func (s *Service) ensureMemberBinding(ctx context.Context, userID int64, externalUserID string) (MemberBindingDTO, bool, error) {
	result, ensureErr := s.client.EnsureUserMapping(ctx, dp7575.UserMapEnsureRequest{ExternalUserID: externalUserID})
	if ensureErr != nil {
		return MemberBindingDTO{}, false, ensureErr
	}
	if !result.IsMapped {
		return MemberBindingDTO{}, false, nil
	}

	now := time.Now()
	binding := MemberBindingDTO{
		UserID:         userID,
		ExternalUserID: result.ExternalUserID,
		SiteID:         result.SiteID,
		Status:         "active",
		LastSyncedAt:   &now,
	}
	if upsertErr := s.repo.Upsert(ctx, binding); upsertErr != nil {
		return MemberBindingDTO{}, false, upsertErr
	}

	return binding, true, nil
}

func (s *Service) hasPurchasedResource(ctx context.Context, binding MemberBindingDTO, resourceID string) bool {
	const pageSize = 20
	for page := 1; ; page++ {
		result, err := s.client.OrderList(ctx, dp7575.OrderListRequest{ExternalUserID: binding.ExternalUserID, BusinessType: "resource_purchase", Page: page, PageSize: pageSize})
		if err != nil {
			return false
		}

		for _, item := range result.List {
			if item.Status != "paid" {
				continue
			}
			detail, detailErr := s.client.OrderDetail(ctx, dp7575.OrderDetailRequest{ExternalUserID: binding.ExternalUserID, ZibOrderNum: item.OrderNum})
			if detailErr != nil {
				continue
			}
			if snapshotResourceID(detail.Snapshot) == resourceID {
				return true
			}
		}

		if len(result.List) == 0 || page*pageSize >= result.Pagination.Total {
			return false
		}
	}
}

func validateResourceLocator(input ResourceAccessCheckRequestDTO) error {
	count := 0
	if input.ResourceID != "" {
		count++
	}
	if input.ArticleID != "" {
		count++
	}
	if input.Abbrlink != "" {
		count++
	}
	if count == 0 || count > 1 {
		if count == 0 {
			return ErrResourceLocatorRequired
		}
		return ErrInvalidResourceLocator
	}
	return nil
}

func ValidateResourceLocatorForAPI(input ResourceAccessCheckRequestDTO) error {
	return validateResourceLocator(input)
}

func snapshotResourceID(snapshot map[string]any) string {
	product, ok := snapshot["product"].(map[string]any)
	if !ok {
		return ""
	}
	if product["resource_id"] != nil {
		return fmt.Sprint(product["resource_id"])
	}
	attach, ok := product["attach"].(map[string]any)
	if ok && attach["resource_id"] != nil {
		return fmt.Sprint(attach["resource_id"])
	}

	return ""
}

func buildResourcePurchaseSnapshot(resourceRecord ResourceRecordDTO) map[string]any {
	return map[string]any{
		"product": map[string]any{
			"resource_id":   resourceRecord.ResourceID,
			"host_type":     resourceRecord.HostType,
			"host_id":       resourceRecord.HostID,
			"title":         resourceRecord.Title,
			"resource_type": resourceRecord.ResourceType,
			"price":         resourceRecord.Price,
			"member_free":   resourceRecord.MemberFree,
		},
	}
}

func normalizeResourcePaymentMethod(paymentMethod string) string {
	if paymentMethod == "alipay" {
		return "alipay"
	}
	return "wechat"
}

func extractOrderSnapshotTitle(snapshot map[string]any) string {
	if snapshot == nil {
		return ""
	}
	product, ok := snapshot["product"].(map[string]any)
	if !ok {
		return ""
	}
	if title, ok := product["title"].(string); ok {
		return strings.TrimSpace(title)
	}
	if title, ok := product["resource_title"].(string); ok {
		return strings.TrimSpace(title)
	}
	return ""
}

func memberPurchaseActionRank(actionType string) int {
	switch actionType {
	case "pay":
		return 0
	case "renew":
		return 1
	case "upgrade":
		return 2
	default:
		return 3
	}
}

func (s *Service) GetMemberOrders(ctx context.Context, userID int64) (MemberOrdersDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberOrdersDTO{}, err
	}

	result, err := s.client.OrderList(ctx, dp7575.OrderListRequest{
		ExternalUserID: binding.ExternalUserID,
	})
	if err != nil {
		return MemberOrdersDTO{}, err
	}

	list := make([]MemberOrderListItemDTO, 0, len(result.List))
	for _, item := range result.List {
		productTitle := ""
		if item.BusinessType == "resource_purchase" && s.resourceOrderRepo != nil {
			resourceOrder, err := s.resourceOrderRepo.FindByExternalOrderNo(ctx, item.OrderNum)
			if err == nil {
				productTitle = extractOrderSnapshotTitle(resourceOrder.Snapshot)
			}
		}
		list = append(list, MemberOrderListItemDTO{
			OrderNo:      item.OrderNum,
			BusinessType: item.BusinessType,
			ProductType:  item.ProductType,
			ProductTitle: productTitle,
			Status:       item.Status,
			Amount:       item.Amount,
			PayType:      item.PayType,
			CreatedAt:    item.CreateTime,
		})
	}

	return MemberOrdersDTO{
		List: list,
		Pagination: MemberOrdersPaginationDTO{
			Page:     result.Pagination.Page,
			PageSize: result.Pagination.PageSize,
			Total:    result.Pagination.Total,
		},
	}, nil
}

func (s *Service) GetMemberOrderDetail(ctx context.Context, userID int64, input MemberOrderDetailRequestDTO) (MemberOrderDetailDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberOrderDetailDTO{}, err
	}

	result, err := s.client.OrderDetail(ctx, dp7575.OrderDetailRequest{
		ExternalUserID: binding.ExternalUserID,
		ZibOrderNum:    input.OrderNo,
	})
	if err != nil {
		return MemberOrderDetailDTO{}, err
	}

	return MemberOrderDetailDTO{
		OrderNo:      result.OrderNum,
		BusinessType: result.BusinessType,
		ProductType:  result.ProductType,
		ProductID:    result.ProductID,
		Status:       result.Status,
		Amount:       result.Amount,
		PayType:      result.PayType,
		PayTime:      result.PayTime,
		CreatedAt:    result.CreateTime,
		Snapshot:     result.Snapshot,
		ZibOrder:     result.ZibOrder,
	}, nil
}

func (s *Service) GetMemberOrderStatus(ctx context.Context, userID int64, input MemberOrderStatusRequestDTO) (MemberOrderStatusDTO, error) {
	binding, err := s.requireMemberBinding(ctx, userID)
	if err != nil {
		return MemberOrderStatusDTO{}, err
	}

	result, err := s.client.OrderStatus(ctx, dp7575.OrderStatusRequest{
		ExternalUserID: binding.ExternalUserID,
		ZibOrderNum:    input.OrderNo,
	})
	if err != nil {
		return MemberOrderStatusDTO{}, err
	}

	return MemberOrderStatusDTO{
		OrderNo:      result.ZibOrderNum,
		BusinessType: result.BusinessType,
		Status:       result.OrderStatus,
		StatusLabel:  result.OrderStatusLabel,
		OrderPrice:   result.OrderPrice,
		PayPrice:     result.PayPrice,
		PayType:      result.PayType,
		CreatedAt:    result.CreatedAt,
		PaidAt:       result.PaidAt,
	}, nil
}

func (s *Service) GetHealthCheck(ctx context.Context, userID int64) (HealthCheckDTO, error) {
	items := []HealthCheckItemDTO{
		{Key: "config", Name: "接入配置是否完整", Description: "检查是否已经填写极光库鉴权系统所需配置", Status: "pending", Result: "未检查"},
		{Key: "connectivity", Name: "极光库服务是否可连接", Description: "检查本站能否正常访问极光库服务", Status: "pending", Result: "未检查"},
		{Key: "signature", Name: "鉴权签名是否有效", Description: "检查当前鉴权信息是否被极光库识别", Status: "pending", Result: "未检查"},
		{Key: "binding", Name: "用户映射是否存在", Description: "检查当前登录用户是否已绑定极光库用户", Status: "pending", Result: "未检查"},
		{Key: "member_fetch", Name: "会员状态是否可获取", Description: "检查是否能够成功获取当前用户会员信息", Status: "pending", Result: "未检查"},
		{Key: "member_current", Name: "当前会员状态", Description: "展示当前会员是否有效", Status: "pending", Result: "未检查"},
	}

	if !s.client.ConfigComplete() {
		items[0].Status = "error"
		items[0].Result = "缺少配置"
		items[0].Detail = "还没有填写完整接入配置"
		return HealthCheckDTO{Items: items}, nil
	}
	items[0].Status = "success"
	items[0].Result = "通过"

	probe, err := s.client.HealthProbe(ctx)
	if err != nil {
		return HealthCheckDTO{}, err
	}
	if probe.Connected {
		items[1].Status = "success"
		items[1].Result = "通过"
	} else {
		items[1].Status = "error"
		items[1].Result = "连接失败"
		items[1].Detail = probe.Detail
		return HealthCheckDTO{Items: items}, nil
	}
	if probe.SignatureValid {
		items[2].Status = "success"
		items[2].Result = "通过"
	} else {
		items[2].Status = "error"
		items[2].Result = "签名无效"
		items[2].Detail = probe.Detail
		return HealthCheckDTO{Items: items}, nil
	}

	binding, found, err := s.findMemberBinding(ctx, userID)
	if err != nil {
		return HealthCheckDTO{}, err
	}
	if !found {
		publicUserID := memberExternalUserID(userID)
		items[3].Status = "warning"
		items[3].Result = "未绑定"
		items[3].Detail = fmt.Sprintf("当前账号还没有绑定极光库用户，请在极光库侧为 external_user_id=%s 建立映射", publicUserID)
		items[4].Status = "warning"
		items[4].Result = "待处理"
		items[4].Detail = fmt.Sprintf("请先在极光库完成用户映射（external_user_id=%s），再重新检查", publicUserID)
		items[5].Status = "warning"
		items[5].Result = "待处理"
		items[5].Detail = fmt.Sprintf("请先在极光库完成用户映射（external_user_id=%s），再重新检查", publicUserID)
		return HealthCheckDTO{Items: items}, nil
	}
	items[3].Status = "success"
	items[3].Result = "已绑定"

	status, err := s.client.MemberStatus(ctx, dp7575.MemberStatusRequest{ExternalUserID: binding.ExternalUserID, SiteID: binding.SiteID})
	if err != nil {
		items[4].Status = "error"
		items[4].Result = "获取失败"
		items[4].Detail = "暂时无法获取会员状态"
		return HealthCheckDTO{Items: items}, nil
	}
	items[4].Status = "success"
	items[4].Result = "正常"

	level := normalizeMemberLevel(status)
	items[5].Status = "success"
	if status.IsMember {
		items[5].Result = "已开通会员"
		items[5].Detail = fmt.Sprintf("当前会员等级：%s，到期时间：%s", level, status.MemberExpireAt)
	} else {
		items[5].Result = "未开通会员"
		items[5].Detail = "当前账号未处于会员有效状态"
	}

	return HealthCheckDTO{Items: items}, nil
}
