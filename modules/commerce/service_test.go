package commerce

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/anzhiyu-c/anheyu-app/ent/enttest"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	"github.com/anzhiyu-c/anheyu-app/pkg/integration/dp7575"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

type bindingRepositoryStub struct {
	binding     MemberBindingDTO
	err         error
	upserted    *MemberBindingDTO
	upsertErr   error
	findCalls   int
	upsertCalls int
}

func (s *bindingRepositoryStub) FindByUserID(_ context.Context, userID int64) (MemberBindingDTO, error) {
	s.findCalls++
	if s.err != nil {
		return MemberBindingDTO{}, s.err
	}
	binding := s.binding
	binding.UserID = userID
	return binding, nil
}

func (s *bindingRepositoryStub) Upsert(_ context.Context, dto MemberBindingDTO) error {
	s.upsertCalls++
	copy := dto
	s.upserted = &copy
	return s.upsertErr
}

type memberStatusClientStub struct {
	status                 MemberStatusDTO
	err                    error
	req                    dp7575.MemberStatusRequest
	profileResp            dp7575.MemberProfileResponse
	profileErr             error
	profileReq             dp7575.MemberProfileRequest
	probe                  dp7575.HealthProbeResult
	queryResp              dp7575.UserMapQueryResponse
	queryErr               error
	queryReq               dp7575.UserMapQueryRequest
	queryCalls             int
	ensureResp             dp7575.UserMapEnsureResponse
	ensureErr              error
	ensureReq              dp7575.UserMapEnsureRequest
	ensureCalls            int
	catalogResp            dp7575.MemberProductsCatalogResponse
	catalogErr             error
	orderResp              dp7575.OrderCreateResponse
	orderErr               error
	orderReq               dp7575.OrderCreateRequest
	paymentResp            dp7575.OrderPaymentDetailResponse
	paymentErr             error
	paymentReq             dp7575.OrderPaymentDetailRequest
	precheckResp           dp7575.CardRedeemPrecheckResponse
	precheckErr            error
	precheckReq            dp7575.CardRedeemPrecheckRequest
	redeemResp             dp7575.CardRedeemCreateResponse
	redeemErr              error
	redeemReq              dp7575.CardRedeemCreateRequest
	orderListResp          dp7575.OrderListResponse
	orderListResps         []dp7575.OrderListResponse
	orderListErr           error
	orderListReq           dp7575.OrderListRequest
	orderDetailResp        dp7575.OrderDetailResponse
	orderDetailResps       map[string]dp7575.OrderDetailResponse
	orderDetailErr         error
	orderDetailReq         dp7575.OrderDetailRequest
	orderStatusResp        dp7575.OrderStatusResponse
	orderStatusErr         error
	orderStatusReq         dp7575.OrderStatusRequest
	orderListCalls         int
	adminOrderMappingsResp dp7575.AdminOrderMappingListResponse
	adminOrderMappingsErr  error
	adminOrderMappingsReq  dp7575.AdminOrderMappingListRequest
	adminCardsResp         dp7575.AdminCardListResponse
	adminCardsErr          error
	adminCardsReq          dp7575.AdminCardListRequest
}

type resourceRepositoryStub struct {
	resourceByID            ResourceRecordDTO
	resourceByHost          ResourceRecordDTO
	adminResources          []AdminResourceListItemDTO
	adminResourceDetail     AdminResourceDetailDTO
	createdAdminResource    AdminResourceDetailDTO
	updatedAdminResource    AdminResourceDetailDTO
	boundResourceID         string
	boundArticleID          string
	deletedResourceID       string
	resourceOrderCount      int
	articleHostOptions      []AdminArticleHostOptionDTO
	resolvedArticleID       string
	hasGrant                bool
	hasGrantBySourceOrderNo bool
	createGrantReq          *ResourceAccessGrantCreateDTO
	createGrantCalls        int
	err                     error
}

func (s *resourceRepositoryStub) FindResourceByID(_ context.Context, resourceID string) (ResourceRecordDTO, error) {
	if s.err != nil {
		return ResourceRecordDTO{}, s.err
	}
	resource := s.resourceByID
	if resource.ResourceID == "" {
		resource.ResourceID = resourceID
	}
	return resource, nil
}

func (s *resourceRepositoryStub) ListAdminResources(_ context.Context, _ AdminResourceListQueryDTO) (AdminResourceListDTO, error) {
	if s.err != nil {
		return AdminResourceListDTO{}, s.err
	}
	return AdminResourceListDTO{List: s.adminResources, Total: len(s.adminResources), Page: 1, PageSize: len(s.adminResources)}, nil
}

func (s *resourceRepositoryStub) GetAdminResourceDetail(_ context.Context, resourceID string) (AdminResourceDetailDTO, error) {
	if s.err != nil {
		return AdminResourceDetailDTO{}, s.err
	}
	resource := s.adminResourceDetail
	if resource.ResourceID == "" {
		resource.ResourceID = resourceID
	}
	return resource, nil
}

func (s *resourceRepositoryStub) CreateAdminResource(_ context.Context, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error) {
	if s.err != nil {
		return AdminResourceDetailDTO{}, s.err
	}
	s.createdAdminResource = input
	if input.ResourceID == "" {
		input.ResourceID = "res_created"
	}
	return input, nil
}

func (s *resourceRepositoryStub) UpdateAdminResource(_ context.Context, resourceID string, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error) {
	if s.err != nil {
		return AdminResourceDetailDTO{}, s.err
	}
	input.ResourceID = resourceID
	s.updatedAdminResource = input
	return input, nil
}

func (s *resourceRepositoryStub) BindAdminResourceToArticle(_ context.Context, resourceID string, articleID string) (AdminResourceDetailDTO, error) {
	if s.err != nil {
		return AdminResourceDetailDTO{}, s.err
	}
	s.boundResourceID = resourceID
	s.boundArticleID = articleID
	return AdminResourceDetailDTO{ResourceID: resourceID, HostType: "article", HostID: articleID}, nil
}

func (s *resourceRepositoryStub) DeleteAdminResource(_ context.Context, resourceID string) error {
	if s.err != nil {
		return s.err
	}
	s.deletedResourceID = resourceID
	return nil
}

func (s *resourceRepositoryStub) CountResourceOrders(_ context.Context, resourceID string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.resourceOrderCount, nil
}

func (s *resourceRepositoryStub) SearchArticleHosts(_ context.Context, _ string) ([]AdminArticleHostOptionDTO, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.articleHostOptions, nil
}

func (s *resourceRepositoryStub) FindResourceByArticleHost(_ context.Context, articleID string) (AdminResourceDetailDTO, error) {
	if s.err != nil {
		return AdminResourceDetailDTO{}, s.err
	}
	if s.adminResourceDetail.HostID != "" && s.adminResourceDetail.HostID != articleID {
		return AdminResourceDetailDTO{}, ErrResourceNotFound
	}
	return s.adminResourceDetail, nil
}

func (s *resourceRepositoryStub) FindResourceByHost(_ context.Context, hostType, hostID string) (ResourceRecordDTO, error) {
	if s.err != nil {
		return ResourceRecordDTO{}, s.err
	}
	if s.resourceByHost.HostID != "" && hostID != s.resourceByHost.HostID {
		return ResourceRecordDTO{}, ErrResourceNotFound
	}
	resource := s.resourceByHost
	if resource.HostType == "" {
		resource.HostType = hostType
	}
	if resource.HostID == "" {
		resource.HostID = hostID
	}
	return resource, nil
}

func (s *resourceRepositoryStub) ResolveArticleIDByAbbrlink(_ context.Context, abbrlink string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.resolvedArticleID == "" {
		return "", ErrResourceNotFound
	}
	return s.resolvedArticleID, nil
}

func (s *resourceRepositoryStub) HasActiveGrant(_ context.Context, _ int64, _, _ string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.hasGrant, nil
}

func (s *resourceRepositoryStub) HasGrantBySourceOrderNo(_ context.Context, _ int64, _ string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.hasGrantBySourceOrderNo, nil
}

func (s *resourceRepositoryStub) CreateGrant(_ context.Context, input ResourceAccessGrantCreateDTO) error {
	if s.err != nil {
		return s.err
	}
	s.createGrantCalls++
	copy := input
	s.createGrantReq = &copy
	return nil
}

type resourceOrderRepositoryStub struct {
	created                ResourceOrderCreateDTO
	existing               ResourceOrderRecordDTO
	byExternalOrderNo      map[string]ResourceOrderRecordDTO
	err                    error
	markPaidUpdated        bool
	updatedBusinessOrderNo string
	updatedExternalOrderNo string
}

func (s *resourceOrderRepositoryStub) Create(_ context.Context, input ResourceOrderCreateDTO) (ResourceOrderRecordDTO, error) {
	if s.err != nil {
		return ResourceOrderRecordDTO{}, s.err
	}
	s.created = input
	return ResourceOrderRecordDTO(input), nil
}

func (s *resourceOrderRepositoryStub) MarkPaid(_ context.Context, businessOrderNo string, externalOrderNo string, paidAt *time.Time) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	if s.existing.Status == "paid" {
		return false, nil
	}
	s.existing.BusinessOrderNo = businessOrderNo
	s.existing.ExternalOrderNo = externalOrderNo
	s.existing.PaidAt = paidAt
	s.existing.Status = "paid"
	s.markPaidUpdated = true
	return true, nil
}

func (s *resourceOrderRepositoryStub) UpdateExternalOrderNo(_ context.Context, businessOrderNo string, externalOrderNo string) error {
	if s.err != nil {
		return s.err
	}
	s.updatedBusinessOrderNo = businessOrderNo
	s.updatedExternalOrderNo = externalOrderNo
	s.existing.BusinessOrderNo = businessOrderNo
	s.existing.ExternalOrderNo = externalOrderNo
	return nil
}

func (s *resourceOrderRepositoryStub) FindLatestPendingByUserAndResource(_ context.Context, userID int64, resourceID string) (ResourceOrderRecordDTO, error) {
	if s.err != nil {
		return ResourceOrderRecordDTO{}, s.err
	}
	if s.existing.BusinessOrderNo != "" && s.existing.UserID == userID && s.existing.ResourceID == resourceID && s.existing.Status == "pending" {
		return s.existing, nil
	}
	return ResourceOrderRecordDTO{}, ErrResourceOrderNotFound
}

func (s *resourceOrderRepositoryStub) FindByBusinessOrderNo(_ context.Context, businessOrderNo string) (ResourceOrderRecordDTO, error) {
	if s.err != nil {
		return ResourceOrderRecordDTO{}, s.err
	}
	if s.existing.BusinessOrderNo == "" {
		s.existing.BusinessOrderNo = businessOrderNo
	}
	return s.existing, nil
}

func (s *resourceOrderRepositoryStub) FindByExternalOrderNo(_ context.Context, externalOrderNo string) (ResourceOrderRecordDTO, error) {
	if s.err != nil {
		return ResourceOrderRecordDTO{}, s.err
	}
	if s.byExternalOrderNo != nil {
		if order, ok := s.byExternalOrderNo[externalOrderNo]; ok {
			return order, nil
		}
	}
	if s.existing.ExternalOrderNo == externalOrderNo {
		return s.existing, nil
	}
	return ResourceOrderRecordDTO{}, ErrResourceOrderNotFound
}

func newStubBindingRepo() *bindingRepositoryStub {
	return &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
}

func newStubMemberClient() *memberStatusClientStub {
	return &memberStatusClientStub{}
}

func (s *memberStatusClientStub) MemberStatus(_ context.Context, req dp7575.MemberStatusRequest) (dp7575.MemberStatusResponse, error) {
	s.req = req
	if s.err != nil {
		return dp7575.MemberStatusResponse{}, s.err
	}
	return dp7575.MemberStatusResponse{
		IsMember:        s.status.IsMember,
		MemberLevel:     []byte(`"` + s.status.Level + `"`),
		MemberLevelName: s.status.Level,
		MemberExpireAt:  s.status.ExpiresAt,
	}, nil
}

func (s *memberStatusClientStub) MemberProfile(_ context.Context, req dp7575.MemberProfileRequest) (dp7575.MemberProfileResponse, error) {
	s.profileReq = req
	if s.profileErr != nil {
		return dp7575.MemberProfileResponse{}, s.profileErr
	}
	return s.profileResp, nil
}

func (s *memberStatusClientStub) ConfigComplete() bool {
	return true
}

func (s *memberStatusClientStub) HealthProbe(_ context.Context) (dp7575.HealthProbeResult, error) {
	return s.probe, nil
}

func (s *memberStatusClientStub) QueryUserMapping(_ context.Context, req dp7575.UserMapQueryRequest) (dp7575.UserMapQueryResponse, error) {
	s.queryCalls++
	s.queryReq = req
	if s.queryErr != nil {
		return dp7575.UserMapQueryResponse{}, s.queryErr
	}
	return s.queryResp, nil
}

func (s *memberStatusClientStub) EnsureUserMapping(_ context.Context, req dp7575.UserMapEnsureRequest) (dp7575.UserMapEnsureResponse, error) {
	s.ensureCalls++
	s.ensureReq = req
	if s.ensureErr != nil {
		return dp7575.UserMapEnsureResponse{}, s.ensureErr
	}
	return s.ensureResp, nil
}

func (s *memberStatusClientStub) MemberProductsCatalog(_ context.Context) (dp7575.MemberProductsCatalogResponse, error) {
	if s.catalogErr != nil {
		return dp7575.MemberProductsCatalogResponse{}, s.catalogErr
	}
	return s.catalogResp, nil
}

func (s *memberStatusClientStub) CreateOrder(_ context.Context, req dp7575.OrderCreateRequest) (dp7575.OrderCreateResponse, error) {
	s.orderReq = req
	if s.orderErr != nil {
		return dp7575.OrderCreateResponse{}, s.orderErr
	}
	return s.orderResp, nil
}

func (s *memberStatusClientStub) OrderPaymentDetail(_ context.Context, req dp7575.OrderPaymentDetailRequest) (dp7575.OrderPaymentDetailResponse, error) {
	s.paymentReq = req
	if s.paymentErr != nil {
		return dp7575.OrderPaymentDetailResponse{}, s.paymentErr
	}
	return s.paymentResp, nil
}

func (s *memberStatusClientStub) CardRedeemPrecheck(_ context.Context, req dp7575.CardRedeemPrecheckRequest) (dp7575.CardRedeemPrecheckResponse, error) {
	s.precheckReq = req
	if s.precheckErr != nil {
		return dp7575.CardRedeemPrecheckResponse{}, s.precheckErr
	}
	return s.precheckResp, nil
}

func (s *memberStatusClientStub) CardRedeemCreate(_ context.Context, req dp7575.CardRedeemCreateRequest) (dp7575.CardRedeemCreateResponse, error) {
	s.redeemReq = req
	if s.redeemErr != nil {
		return dp7575.CardRedeemCreateResponse{}, s.redeemErr
	}
	return s.redeemResp, nil
}

func (s *memberStatusClientStub) OrderList(_ context.Context, req dp7575.OrderListRequest) (dp7575.OrderListResponse, error) {
	s.orderListCalls++
	s.orderListReq = req
	if s.orderListErr != nil {
		return dp7575.OrderListResponse{}, s.orderListErr
	}
	if len(s.orderListResps) >= s.orderListCalls {
		return s.orderListResps[s.orderListCalls-1], nil
	}
	return s.orderListResp, nil
}

func TestCheckResourceAccess_UsesStoredResourceForGuest(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{
		resourceByID: ResourceRecordDTO{
			ResourceID:    "res_real_1",
			Title:         "前端项目源码包",
			ResourceType:  "download_bundle",
			Status:        "published",
			SaleEnabled:   true,
			Price:         29.9,
			OriginalPrice: 49.9,
			MemberFree:    false,
		},
	}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.NoError(t, err)
	require.Equal(t, "login_required", result.Reason)
	require.Equal(t, "前端项目源码包", result.ResourceMeta.Title)
	require.Equal(t, 29.9, result.Price)
}

func TestCheckResourceAccess_ResolvesResourceFromArticleHost(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{
		resourceByHost: ResourceRecordDTO{
			ResourceID:   "res_article_1",
			HostType:     "article",
			HostID:       "art_public_1",
			Title:        "文章配套资源",
			ResourceType: "download_bundle",
			Status:       "published",
			SaleEnabled:  true,
			Price:        9.9,
		},
	}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{ArticleID: "art_public_1"})
	require.NoError(t, err)
	require.Equal(t, "res_article_1", result.ResourceMeta.ResourceID)
}

func TestCheckResourceAccess_ResolvesResourceFromAbbrlinkViaArticleID(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{
		resolvedArticleID: "art_public_1",
		resourceByHost: ResourceRecordDTO{
			ResourceID:   "res_article_1",
			HostType:     "article",
			HostID:       "art_public_1",
			Title:        "文章配套资源",
			ResourceType: "download_bundle",
			Status:       "published",
			SaleEnabled:  true,
			Price:        9.9,
		},
	}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{Abbrlink: "hello-world"})
	require.NoError(t, err)
	require.Equal(t, "res_article_1", result.ResourceMeta.ResourceID)
	require.Equal(t, "login_required", result.Reason)
}

func TestCheckResourceAccess_UsesLocalGrantBeforeRemoteOrderFallback(t *testing.T) {
	client := newStubMemberClient()
	svc := NewService(newStubBindingRepo(), client)
	svc.resourceRepo = &resourceRepositoryStub{
		resourceByID: ResourceRecordDTO{
			ResourceID:   "res_paid_1",
			Title:        "已购资源",
			ResourceType: "download_bundle",
			Status:       "published",
			SaleEnabled:  true,
			Price:        19.9,
		},
		hasGrant: true,
	}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_paid_1"})
	require.NoError(t, err)
	require.Equal(t, "already_purchased", result.Reason)
	require.True(t, result.AccessGranted)
	require.Zero(t, client.orderListCalls)
}

func TestCheckResourceAccess_UsesLocalGrantBeforeBindingResolution(t *testing.T) {
	repo := &bindingRepositoryStub{err: errors.New("binding missing")}
	client := newStubMemberClient()
	svc := NewService(repo, client)
	svc.resourceRepo = &resourceRepositoryStub{
		resourceByID: ResourceRecordDTO{
			ResourceID:   "res_paid_1",
			Title:        "已购资源",
			ResourceType: "download_bundle",
			Status:       "published",
			SaleEnabled:  true,
			Price:        19.9,
		},
		hasGrant: true,
	}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_paid_1"})
	require.NoError(t, err)
	require.True(t, result.AccessGranted)
	require.Equal(t, "already_purchased", result.Reason)
	require.Zero(t, client.ensureCalls)
	require.Zero(t, repo.upsertCalls)
}

func TestBuildPaidAccessDecisionForPremium_LoginRequired(t *testing.T) {
	svc := NewService(&bindingRepositoryStub{}, &memberStatusClientStub{})

	decision := svc.buildPaidAccessDecision(paidAccessDecisionInput{
		Kind:     paidAccessKindPremium,
		LoggedIn: false,
	})

	require.Equal(t, PaidAccessStateLoginRequired, decision.State)
	require.False(t, decision.Allowed)
}

func TestBuildPaidAccessDecisionForPremium_MemberRequired(t *testing.T) {
	svc := NewService(&bindingRepositoryStub{}, &memberStatusClientStub{})

	decision := svc.buildPaidAccessDecision(paidAccessDecisionInput{
		Kind:         paidAccessKindPremium,
		LoggedIn:     true,
		PremiumReady: false,
	})

	require.Equal(t, PaidAccessStateMemberRequired, decision.State)
	require.False(t, decision.Allowed)
}

func TestBuildPaidAccessDecisionForResource_PurchaseRequired(t *testing.T) {
	svc := NewService(&bindingRepositoryStub{}, &memberStatusClientStub{})

	decision := svc.buildPaidAccessDecision(paidAccessDecisionInput{
		Kind:          paidAccessKindResource,
		LoggedIn:      true,
		HasPurchase:   false,
		MemberFree:    false,
		AccessGranted: false,
	})

	require.Equal(t, PaidAccessStatePurchaseRequired, decision.State)
	require.False(t, decision.Allowed)
}

func (s *memberStatusClientStub) OrderDetail(_ context.Context, req dp7575.OrderDetailRequest) (dp7575.OrderDetailResponse, error) {
	s.orderDetailReq = req
	if s.orderDetailErr != nil {
		return dp7575.OrderDetailResponse{}, s.orderDetailErr
	}
	if s.orderDetailResps != nil {
		if resp, ok := s.orderDetailResps[req.ZibOrderNum]; ok {
			return resp, nil
		}
	}
	return s.orderDetailResp, nil
}

func (s *memberStatusClientStub) OrderStatus(_ context.Context, req dp7575.OrderStatusRequest) (dp7575.OrderStatusResponse, error) {
	s.orderStatusReq = req
	if s.orderStatusErr != nil {
		return dp7575.OrderStatusResponse{}, s.orderStatusErr
	}
	return s.orderStatusResp, nil
}

func (s *memberStatusClientStub) AdminOrderMappings(_ context.Context, req dp7575.AdminOrderMappingListRequest) (dp7575.AdminOrderMappingListResponse, error) {
	s.adminOrderMappingsReq = req
	if s.adminOrderMappingsErr != nil {
		return dp7575.AdminOrderMappingListResponse{}, s.adminOrderMappingsErr
	}
	return s.adminOrderMappingsResp, nil
}

func (s *memberStatusClientStub) AdminCards(_ context.Context, req dp7575.AdminCardListRequest) (dp7575.AdminCardListResponse, error) {
	s.adminCardsReq = req
	if s.adminCardsErr != nil {
		return dp7575.AdminCardListResponse{}, s.adminCardsErr
	}
	return s.adminCardsResp, nil
}

func TestMemberBindingRepository_BindAndFind(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:memberbinding?mode=memory&_fk=1", enttest.WithMigrateOptions(schema.WithGlobalUniqueID(false)))
	defer client.Close()

	repo := NewBindingRepository(client)
	err := repo.Upsert(ctx, MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	})
	require.NoError(t, err)

	binding, err := repo.FindByUserID(ctx, 1001)
	require.NoError(t, err)
	require.Equal(t, "dp-user-001", binding.ExternalUserID)
}

func TestCheckResourceAccess_EnsureBindingThenPurchaseRequired(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{
		status:     MemberStatusDTO{IsMember: false},
		ensureResp: dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
	}
	svc := NewService(repo, client)
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "演示资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 9.9, OriginalPrice: 9.9, MemberFree: false}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})

	require.NoError(t, err)
	require.False(t, result.AccessGranted)
	require.Equal(t, "purchase_required", result.Reason)
	require.True(t, result.RequiresPurchase)
	require.True(t, result.Payable)
	require.Equal(t, "resource_purchase", result.BusinessType)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.req.ExternalUserID)
	require.NotNil(t, repo.upserted)
	require.Equal(t, "oerx", repo.upserted.ExternalUserID)
}

func TestCheckResourceAccess_LoginRequiredWhenNotLoggedIn(t *testing.T) {
	svc := NewService(&bindingRepositoryStub{err: ErrMemberBindingNotFound}, &memberStatusClientStub{})
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "演示资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 9.9, OriginalPrice: 9.9, MemberFree: false}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})

	require.NoError(t, err)
	require.False(t, result.AccessGranted)
	require.True(t, result.RequiresLogin)
	require.Equal(t, "login_required", result.Reason)
}

func TestCheckResourceAccess_ArticleWithoutBoundResourceReturnsNoResourceState(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	svc := NewService(repo, &memberStatusClientStub{})
	svc.resourceRepo = &resourceRepositoryStub{err: ErrResourceNotFound}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{Abbrlink: "Y4hH"})

	require.NoError(t, err)
	require.False(t, result.AccessGranted)
	require.False(t, result.RequiresPurchase)
	require.False(t, result.Payable)
	require.Equal(t, "resource_not_found", result.Reason)
	require.Empty(t, result.ResourceMeta.ResourceID)
}

func TestCheckResourceAccess_UnavailableWhenSaleDisabled(t *testing.T) {
	svc := NewService(&bindingRepositoryStub{err: ErrMemberBindingNotFound}, &memberStatusClientStub{})
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "演示资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: false, Price: 9.9, OriginalPrice: 9.9, MemberFree: false}}

	_, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{LoggedIn: false}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})
	require.ErrorIs(t, err, ErrResourceUnavailable)
}

func TestCheckResourceAccess_MemberFreeForMember(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{status: MemberStatusDTO{IsMember: true}}
	svc := NewService(repo, client)
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "会员免费资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 9.9, OriginalPrice: 9.9, MemberFree: true}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})

	require.NoError(t, err)
	require.True(t, result.AccessGranted)
	require.Equal(t, "member_free", result.Reason)
	require.True(t, result.MemberFree)
	require.True(t, result.UserIsMember)
}

func TestCheckResourceAccess_FreeResourceDoesNotMarkLoggedInUserAsMember(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_free_1", Title: "免费资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 0, OriginalPrice: 9.9, MemberFree: false, ResourceItems: []ResourceAccessItemDTO{{ID: "item_1", Title: "夸克网盘", URL: "https://pan.example.com/x", ExtractionCode: "ABCD"}}}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_free_1"})
	require.NoError(t, err)
	require.True(t, result.AccessGranted)
	require.False(t, result.UserIsMember)
	require.Len(t, result.ResourceItems, 1)
	require.Equal(t, "夸克网盘", result.ResourceItems[0].Title)
}

func TestCheckResourceAccess_AlreadyPurchasedWhenPaidSnapshotMatchesResource(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{
		status:          MemberStatusDTO{IsMember: false},
		orderListResp:   dp7575.OrderListResponse{List: []dp7575.OrderListItem{{OrderNum: "2604190047071040704", Status: "paid", BusinessType: "resource_purchase"}}},
		orderDetailResp: dp7575.OrderDetailResponse{Snapshot: map[string]any{"product": map[string]any{"resource_id": "res_demo_1"}}},
	}
	svc := NewService(repo, client)
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:    "res_demo_1",
		Title:         "已购资源",
		ResourceType:  "download_bundle",
		Status:        "published",
		SaleEnabled:   true,
		Price:         9.9,
		OriginalPrice: 9.9,
		MemberFree:    false,
		ResourceItems: []ResourceAccessItemDTO{{ID: "item_1", Title: "百度网盘", URL: "https://pan.example.com/x"}},
	}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})

	require.NoError(t, err)
	require.True(t, result.AccessGranted)
	require.Equal(t, "already_purchased", result.Reason)
	require.True(t, result.AlreadyPurchased)
	require.Len(t, result.ResourceItems, 1)
	require.Equal(t, "百度网盘", result.ResourceItems[0].Title)
}

func TestCheckResourceAccess_PaginatesRemoteOrderFallback(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{
		status: MemberStatusDTO{IsMember: false},
		orderListResps: []dp7575.OrderListResponse{
			{List: []dp7575.OrderListItem{{OrderNum: "P1", Status: "paid", BusinessType: "resource_purchase"}}, Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 20, Total: 21}},
			{List: []dp7575.OrderListItem{{OrderNum: "P2", Status: "paid", BusinessType: "resource_purchase"}}, Pagination: dp7575.OrderListPagination{Page: 2, PageSize: 20, Total: 21}},
		},
		orderDetailResps: map[string]dp7575.OrderDetailResponse{
			"P1": {Snapshot: map[string]any{"product": map[string]any{"resource_id": "res_other"}}},
			"P2": {Snapshot: map[string]any{"product": map[string]any{"resource_id": "res_demo_1"}}},
		},
	}
	svc := NewService(repo, client)
	svc.resourceRepo = &resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "已购资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 9.9, OriginalPrice: 9.9, MemberFree: false}}

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})
	require.NoError(t, err)
	require.True(t, result.AccessGranted)
	require.Equal(t, 2, client.orderListCalls)
	require.Equal(t, 2, client.orderListReq.Page)
}

type premiumArticleRepositoryStub struct {
	articleContentHTML string
	err                error
}

func (s *premiumArticleRepositoryStub) FindContentHTMLByPremiumContentID(_ context.Context, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.articleContentHTML, nil
}

func TestService_GetPremiumMemberBlockContent_ReturnsPreviewHTML(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{status: MemberStatusDTO{IsMember: true, Level: "2"}}
	svc := NewService(repo, client)
	svc.SetArticleRepository(&premiumArticleRepositoryStub{articleContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-10"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>会员正文内容</p></div></div></div>`})

	html, err := svc.GetPremiumMemberBlockContent(context.Background(), 1001, "premium-10")
	require.NoError(t, err)
	require.Equal(t, `<p>会员正文内容</p>`, html)
}

func TestService_GetPremiumMemberBlockContentForActor_UsesResolvedExternalIdentity(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{
		status:     MemberStatusDTO{IsMember: true, Level: "2"},
		ensureResp: dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
	}
	svc := NewService(repo, client)
	svc.SetArticleRepository(&premiumArticleRepositoryStub{articleContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-10"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>会员正文内容</p></div></div></div>`})

	html, err := svc.GetPremiumMemberBlockContentForActor(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, "premium-10")

	require.NoError(t, err)
	require.Equal(t, `<p>会员正文内容</p>`, html)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.req.ExternalUserID)
}

func TestCheckResourceAccess_StillReturnsPurchaseRequiredAfterDecisionRefactor(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{status: MemberStatusDTO{IsMember: false}}
	svc := NewService(repo, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "资源包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 19.9}}, &resourceOrderRepositoryStub{})

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})
	require.NoError(t, err)
	require.Equal(t, "purchase_required", result.Reason)
	require.False(t, result.AccessGranted)
	require.True(t, result.RequiresPurchase)
}

func TestGetPremiumMemberBlockContentForActor_StillRejectsNonPremiumMember(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{status: MemberStatusDTO{IsMember: true, Level: "1"}}
	svc := NewService(repo, client)
	svc.SetArticleRepository(&premiumArticleRepositoryStub{articleContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-1"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>hidden</p></div></div></div>`})

	_, err := svc.GetPremiumMemberBlockContentForActor(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "oerx", LoggedIn: true}, "premium-1")
	require.ErrorIs(t, err, ErrResourcePurchaseNotRequired)
}

func TestCreateResourcePurchaseOrder_PersistsLocalOrderSnapshot(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		HostType:     "article",
		HostID:       "art_public_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}}
	client := newStubMemberClient()
	client.orderResp = dp7575.OrderCreateResponse{ZibOrderNum: "ZGZ_EXT_001", PayURL: "https://pay.example.com/1", OrderPrice: 29.9, OrderStatus: "pending"}

	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	result, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.NoError(t, err)
	require.Equal(t, "res_real_1", orderRepo.created.ResourceID)
	product := orderRepo.created.Snapshot["product"].(map[string]any)
	require.Equal(t, "article", product["host_type"])
	require.Equal(t, "res_real_1", result.ResourceID)
	require.Equal(t, "https://pay.example.com/1", result.PayURL)
	require.Equal(t, "resource_purchase", client.orderReq.BusinessType)
	require.Equal(t, result.BusinessOrderNo, orderRepo.updatedBusinessOrderNo)
	require.Equal(t, "ZGZ_EXT_001", orderRepo.updatedExternalOrderNo)
}

func TestCreateResourcePurchaseOrder_UsesRequestedPaymentMethod(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}}
	client := newStubMemberClient()
	client.orderResp = dp7575.OrderCreateResponse{ZibOrderNum: "ZGZ_EXT_001", PayURL: "https://pay.example.com/wechat/1", OrderPrice: 29.9, OrderStatus: "pending"}

	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	_, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1", PaymentMethod: "wechat"})
	require.NoError(t, err)
	require.Equal(t, "wechat", client.orderReq.PaymentMethod)
}

func TestCreateResourcePurchaseOrder_ReusesExistingPendingOrderWithoutCreatingLocalDuplicate(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{
		existing: ResourceOrderRecordDTO{
			UserID:          1001,
			ResourceID:      "res_real_1",
			BusinessOrderNo: "YGZ_RES_OLD_001",
			Amount:          29.9,
			Status:          "pending",
			Snapshot:        map[string]any{"product": map[string]any{"resource_id": "res_real_1"}},
		},
	}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}}
	client := newStubMemberClient()
	client.orderResp = dp7575.OrderCreateResponse{ZibOrderNum: "ZGZ_EXT_001", PayURL: "https://pay.example.com/1", OrderPrice: 29.9, OrderStatus: "pending"}

	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	result, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.NoError(t, err)
	require.Empty(t, orderRepo.created.BusinessOrderNo)
	require.Equal(t, "YGZ_RES_OLD_001", client.orderReq.BusinessOrderNo)
	require.Equal(t, "YGZ_RES_OLD_001", result.BusinessOrderNo)
	require.Equal(t, "ZGZ_EXT_001", orderRepo.updatedExternalOrderNo)
}

func TestCreateResourcePurchaseOrder_ReusesExistingPendingOrderWithExternalOrderNo(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{
		existing: ResourceOrderRecordDTO{
			UserID:          1001,
			ResourceID:      "res_real_1",
			BusinessOrderNo: "YGZ_RES_OLD_002",
			ExternalOrderNo: "ZGZ_EXT_OLD_002",
			Amount:          29.9,
			Status:          "pending",
		},
	}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}}
	client := newStubMemberClient()

	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	result, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.NoError(t, err)
	require.Empty(t, orderRepo.created.BusinessOrderNo)
	require.Empty(t, client.orderReq.BusinessOrderNo)
	require.Equal(t, "YGZ_RES_OLD_002", result.BusinessOrderNo)
	require.Empty(t, result.PayURL)
}

func TestCreateResourcePurchaseOrder_RejectsSaleDisabledResource(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		HostType:     "article",
		HostID:       "art_public_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  false,
		Price:        29.9,
	}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	_, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.ErrorIs(t, err, ErrResourceUnavailable)
}

func TestCreateResourcePurchaseOrder_RejectsAlreadyPurchasedResource(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}, hasGrant: true}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	_, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.ErrorIs(t, err, ErrResourcePurchaseNotRequired)
}

func TestCreateResourcePurchaseOrder_RejectsMemberFreeResourceForMember(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "会员免费资源",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
		MemberFree:   true,
	}}
	client := newStubMemberClient()
	client.status = MemberStatusDTO{IsMember: true}
	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	_, err := svc.CreateResourcePurchaseOrder(context.Background(), 1001, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.ErrorIs(t, err, ErrResourcePurchaseNotRequired)
}

func TestCreateResourcePurchaseOrderForActor_UsesResolvedExternalIdentity(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{}
	resourceRepo := &resourceRepositoryStub{resourceByID: ResourceRecordDTO{
		ResourceID:   "res_real_1",
		Title:        "前端项目源码包",
		ResourceType: "download_bundle",
		Status:       "published",
		SaleEnabled:  true,
		Price:        29.9,
	}}
	client := &memberStatusClientStub{
		status:     MemberStatusDTO{IsMember: false},
		ensureResp: dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
		orderResp:  dp7575.OrderCreateResponse{ZibOrderNum: "ZGZ_EXT_001", PayURL: "https://pay.example.com/1", OrderPrice: 29.9, OrderStatus: "pending"},
	}
	svc := NewService(&bindingRepositoryStub{err: ErrMemberBindingNotFound}, client)
	svc.SetResourceRepositories(resourceRepo, orderRepo)

	result, err := svc.CreateResourcePurchaseOrderForActor(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_real_1"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.req.ExternalUserID)
	require.Equal(t, "oerx", client.orderReq.ExternalUserID)
	require.Equal(t, "res_real_1", result.ResourceID)
}

func TestMarkResourceOrderPaid_CreatesAccessGrant(t *testing.T) {
	grantRepo := &resourceRepositoryStub{}
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_001",
		UserID:          1001,
		ResourceID:      "res_real_1",
		Status:          "pending",
	}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(grantRepo, orderRepo)

	err := svc.MarkResourceOrderPaid(context.Background(), "YGZ_RES_001", "2604190047071040704")
	require.NoError(t, err)
	require.NotNil(t, grantRepo.createGrantReq)
	require.Equal(t, "res_real_1", grantRepo.createGrantReq.ResourceID)
	require.Equal(t, "purchase", grantRepo.createGrantReq.GrantType)
	require.Equal(t, "YGZ_RES_001", grantRepo.createGrantReq.SourceOrderNo)
}

func TestMarkResourceOrderPaid_SkipsDuplicateGrantForSameSourceOrder(t *testing.T) {
	grantRepo := &resourceRepositoryStub{hasGrantBySourceOrderNo: true}
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_001",
		UserID:          1001,
		ResourceID:      "res_real_1",
		Status:          "pending",
	}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(grantRepo, orderRepo)

	err := svc.MarkResourceOrderPaid(context.Background(), "YGZ_RES_001", "2604190047071040704")
	require.NoError(t, err)
	require.Nil(t, grantRepo.createGrantReq)
	require.Equal(t, 0, grantRepo.createGrantCalls)
}

func TestMarkResourceOrderPaid_SkipsGrantWhenOrderAlreadyPaid(t *testing.T) {
	grantRepo := &resourceRepositoryStub{}
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_001",
		UserID:          1001,
		ResourceID:      "res_real_1",
		Status:          "paid",
	}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(grantRepo, orderRepo)

	err := svc.MarkResourceOrderPaid(context.Background(), "YGZ_RES_001", "2604190047071040704")
	require.NoError(t, err)
	require.False(t, orderRepo.markPaidUpdated)
	require.Nil(t, grantRepo.createGrantReq)
	require.Equal(t, 0, grantRepo.createGrantCalls)
}

func TestGetResourcePurchaseOrderStatus_MarksGrantWhenRemoteStatusPaid(t *testing.T) {
	grantRepo := &resourceRepositoryStub{}
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_002",
		UserID:          1001,
		ResourceID:      "res_real_1",
		ExternalOrderNo: "ZGZ_EXT_002",
		Status:          "pending",
	}}
	client := newStubMemberClient()
	client.orderStatusResp = dp7575.OrderStatusResponse{ZibOrderNum: "ZGZ_EXT_002", OrderStatus: "paid", OrderStatusLabel: "已支付", OrderPrice: 29.9, PayPrice: 29.9, PayType: "alipay", CreatedAt: "2026-04-19 12:00:00", PaidAt: "2026-04-19 12:01:00"}
	svc := NewService(newStubBindingRepo(), client)
	svc.SetResourceRepositories(grantRepo, orderRepo)

	result, err := svc.GetResourcePurchaseOrderStatus(context.Background(), 1001, "YGZ_RES_002")
	require.NoError(t, err)
	require.Equal(t, "paid", result.Status)
	require.NotNil(t, grantRepo.createGrantReq)
	require.Equal(t, "res_real_1", result.ResourceID)
}

func TestGetResourcePurchaseOrderStatus_NotFoundForOtherUser(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_002",
		UserID:          2002,
		ResourceID:      "res_real_1",
		ExternalOrderNo: "ZGZ_EXT_002",
		Status:          "pending",
	}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.SetResourceRepositories(&resourceRepositoryStub{}, orderRepo)

	_, err := svc.GetResourcePurchaseOrderStatus(context.Background(), 1001, "YGZ_RES_002")
	require.ErrorIs(t, err, ErrResourceOrderNotFound)
}

func TestGetResourcePurchaseOrderStatusForActor_UsesResolvedExternalIdentity(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_002",
		UserID:          1001,
		ResourceID:      "res_real_1",
		ExternalOrderNo: "ZGZ_EXT_002",
		Status:          "pending",
	}}
	client := &memberStatusClientStub{
		ensureResp:      dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
		orderStatusResp: dp7575.OrderStatusResponse{ZibOrderNum: "ZGZ_EXT_002", OrderStatus: "pending"},
	}
	svc := NewService(&bindingRepositoryStub{err: ErrMemberBindingNotFound}, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{}, orderRepo)

	result, err := svc.GetResourcePurchaseOrderStatusForActor(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, "YGZ_RES_002")
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.orderStatusReq.ExternalUserID)
	require.Equal(t, "YGZ_RES_002", result.BusinessOrderNo)
}

func TestGetResourcePurchasePaymentDetail_UsesExternalOrderNoForOwnedOrder(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_002",
		UserID:          1001,
		ResourceID:      "res_real_1",
		ExternalOrderNo: "ZGZ_EXT_002",
		Status:          "pending",
	}}
	client := &memberStatusClientStub{paymentResp: dp7575.OrderPaymentDetailResponse{
		OrderNum:   "ZGZ_EXT_002",
		Amount:     29.9,
		PayType:    "alipay",
		PayChannel: "alipay",
		PayDetail: map[string]any{
			"url_qrcode": "data:image/png;base64,ALIPAYQR",
			"pay_url":    "https://pay.example.com/alipay/ZGZ_EXT_002",
		},
	}}

	svc := NewService(repo, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{}, orderRepo)

	result, err := svc.GetResourcePaymentDetail(context.Background(), 1001, ResourceOrderPaymentDetailDTO{BusinessOrderNo: "YGZ_RES_002"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.paymentReq.ExternalUserID)
	require.Equal(t, "ZGZ_EXT_002", client.paymentReq.ZibOrderNum)
	require.Equal(t, "YGZ_RES_002", result.BusinessOrderNo)
	require.Equal(t, "res_real_1", result.ResourceID)
	require.Equal(t, "alipay", result.PayType)
	require.Equal(t, "data:image/png;base64,ALIPAYQR", result.PayDetail["url_qrcode"])
}

func TestGetResourcePaymentDetailForActor_UsesResolvedExternalIdentity(t *testing.T) {
	orderRepo := &resourceOrderRepositoryStub{existing: ResourceOrderRecordDTO{
		BusinessOrderNo: "YGZ_RES_002",
		UserID:          1001,
		ResourceID:      "res_real_1",
		ExternalOrderNo: "ZGZ_EXT_002",
		Status:          "pending",
	}}
	client := &memberStatusClientStub{
		ensureResp: dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
		paymentResp: dp7575.OrderPaymentDetailResponse{
			OrderNum:   "ZGZ_EXT_002",
			Amount:     29.9,
			PayType:    "alipay",
			PayChannel: "alipay",
		},
	}
	svc := NewService(&bindingRepositoryStub{err: ErrMemberBindingNotFound}, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{}, orderRepo)

	result, err := svc.GetResourcePaymentDetailForActor(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceOrderPaymentDetailDTO{BusinessOrderNo: "YGZ_RES_002"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.paymentReq.ExternalUserID)
	require.Equal(t, "YGZ_RES_002", result.BusinessOrderNo)
}

func TestListAdminResources_ReturnsBoundArticleMeta(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{
		adminResources: []AdminResourceListItemDTO{{
			ResourceID:  "res_1",
			Title:       "前端源码包",
			Status:      "published",
			SaleEnabled: true,
			Price:       29.9,
			HostType:    "article",
			HostID:      "art_1",
			HostTitle:   "文章 A",
		}},
	}

	result, err := svc.ListAdminResources(context.Background(), AdminResourceListQueryDTO{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	require.Equal(t, "文章 A", result.List[0].HostTitle)
}

func TestGetAdminResourceDetail_ReturnsItemsAndBinding(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{
		adminResourceDetail: AdminResourceDetailDTO{
			ResourceID: "res_1",
			Title:      "前端源码包",
			HostType:   "article",
			HostID:     "art_1",
			HostTitle:  "文章 A",
			Items:      []AdminResourceItemDTO{{Title: "百度网盘", URL: "https://pan.example.com/x"}},
		},
	}

	result, err := svc.GetAdminResourceDetail(context.Background(), "res_1")
	require.NoError(t, err)
	require.Equal(t, "article", result.HostType)
	require.Len(t, result.Items, 1)
}

func TestCreateAdminResource_StoresArticleBinding(t *testing.T) {
	repo := &resourceRepositoryStub{}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = repo

	_, err := svc.CreateAdminResource(context.Background(), AdminResourceDetailDTO{
		Title:       "资源 A",
		Status:      "published",
		SaleEnabled: true,
		Price:       0,
		HostType:    "article",
		HostID:      "art_1",
		Items:       []AdminResourceItemDTO{{Title: "百度网盘", ItemType: "link", URL: "https://pan.example.com/x", Status: "active"}},
	})

	require.NoError(t, err)
	require.Equal(t, "article", repo.createdAdminResource.HostType)
	require.Equal(t, "art_1", repo.createdAdminResource.HostID)
}

func TestCreateAdminResource_RejectsNegativePrice(t *testing.T) {
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = &resourceRepositoryStub{}

	_, err := svc.CreateAdminResource(context.Background(), AdminResourceDetailDTO{Title: "资源 A", Price: -1})
	require.Error(t, err)
}

func TestBindAdminResourceToArticle_UsesDedicatedBindingAction(t *testing.T) {
	repo := &resourceRepositoryStub{}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = repo

	_, err := svc.BindAdminResourceToArticle(context.Background(), "res_2", "art_1")

	require.NoError(t, err)
	require.Equal(t, "res_2", repo.boundResourceID)
	require.Equal(t, "art_1", repo.boundArticleID)
	require.Empty(t, repo.updatedAdminResource.ResourceID)
}

func TestDeleteAdminResource_RejectsBoundResource(t *testing.T) {
	repo := &resourceRepositoryStub{adminResourceDetail: AdminResourceDetailDTO{ResourceID: "res_1", HostType: "article", HostID: "art_1"}}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = repo

	err := svc.DeleteAdminResource(context.Background(), "res_1")

	require.ErrorIs(t, err, ErrResourceBoundToArticle)
}

func TestDeleteAdminResource_RejectsResourceWithOrders(t *testing.T) {
	repo := &resourceRepositoryStub{adminResourceDetail: AdminResourceDetailDTO{ResourceID: "res_1"}, resourceOrderCount: 1}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = repo

	err := svc.DeleteAdminResource(context.Background(), "res_1")

	require.ErrorIs(t, err, ErrResourceHasOrders)
}

func TestDeleteAdminResource_AllowsPlainResource(t *testing.T) {
	repo := &resourceRepositoryStub{adminResourceDetail: AdminResourceDetailDTO{ResourceID: "res_1"}, resourceOrderCount: 0}
	svc := NewService(newStubBindingRepo(), newStubMemberClient())
	svc.resourceRepo = repo

	err := svc.DeleteAdminResource(context.Background(), "res_1")

	require.NoError(t, err)
	require.Equal(t, "res_1", repo.deletedResourceID)
}

func TestMemberBindingRepository_UpsertUpdatesExistingBinding(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:memberbinding-upsert?mode=memory&_fk=1", enttest.WithMigrateOptions(schema.WithGlobalUniqueID(false)))
	defer client.Close()

	repo := NewBindingRepository(client)
	require.NoError(t, repo.Upsert(ctx, MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}))

	require.NoError(t, repo.Upsert(ctx, MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-002",
		SiteID:         "yangguangzhan",
		Status:         "inactive",
	}))

	binding, err := repo.FindByUserID(ctx, 1001)
	require.NoError(t, err)
	require.Equal(t, "dp-user-002", binding.ExternalUserID)
	require.Equal(t, "inactive", binding.Status)
}

func TestMemberBindingRepository_RejectsDuplicateExternalIdentityPerSite(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:memberbinding-duplicate?mode=memory&_fk=1", enttest.WithMigrateOptions(schema.WithGlobalUniqueID(false)))
	defer client.Close()

	repo := NewBindingRepository(client)
	require.NoError(t, repo.Upsert(ctx, MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}))

	err := repo.Upsert(ctx, MemberBindingDTO{
		UserID:         1002,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	})
	require.Error(t, err)
}

func TestService_GetMemberStatus(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{status: MemberStatusDTO{
		IsMember:  true,
		Level:     "annual",
		ExpiresAt: "2027-04-18T00:00:00Z",
	}}

	svc := NewService(repo, client)
	status, err := svc.GetMemberStatus(context.Background(), 1001)
	require.NoError(t, err)
	require.True(t, status.IsMember)
	require.Equal(t, "annual", status.Level)
	require.Equal(t, "dp-user-001", client.req.ExternalUserID)
	require.Equal(t, "yangguangzhan", client.req.SiteID)
	require.Equal(t, "ready", status.State)
}

func TestService_GetMemberStatus_WhenBindingMissing_ReturnsPendingState(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{}

	svc := NewService(repo, client)
	status, err := svc.GetMemberStatus(context.Background(), 1001)
	require.NoError(t, err)
	require.False(t, status.IsMember)
	require.Equal(t, "pending", status.State)
	require.Contains(t, status.Message, "当前账号尚未完成会员映射")
	require.Contains(t, status.Message, "external_user_id=")
	require.Zero(t, client.ensureCalls)
}

func TestService_GetMemberStatus_ReturnsRepositoryErrorWhenBindingLookupFails(t *testing.T) {
	repo := &bindingRepositoryStub{err: context.DeadlineExceeded}
	client := &memberStatusClientStub{}

	svc := NewService(repo, client)
	_, err := svc.GetMemberStatus(context.Background(), 1001)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, client.ensureCalls)
}

func TestService_IsPremiumMember_WhenBindingMissing_EnsuresMapping(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())

	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{
		ensureResp: dp7575.UserMapEnsureResponse{
			SiteID:         "yangguangzhan",
			ExternalUserID: memberExternalUserID(1001),
			WPUserID:       7,
			IsMapped:       true,
			Mapped:         true,
			Action:         "existing",
		},
		status: MemberStatusDTO{IsMember: true, Level: "2", ExpiresAt: "2027-04-18T00:00:00Z"},
	}

	svc := NewService(repo, client)
	allowed, err := svc.IsPremiumMember(context.Background(), 1001)

	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, 1, client.ensureCalls)
	require.Equal(t, memberExternalUserID(1001), client.ensureReq.ExternalUserID)
	require.NotNil(t, repo.upserted)
	require.Equal(t, "yangguangzhan", repo.upserted.SiteID)
	require.Equal(t, memberExternalUserID(1001), repo.upserted.ExternalUserID)
}

func TestService_CheckResourceAccess_ReconcilesLegacyBindingBeforeMemberLookup(t *testing.T) {
	publicUserID := memberExternalUserID(1001)
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{ExternalUserID: publicUserID, SiteID: "yangguangzhan", Status: "active"}}
	client := &memberStatusClientStub{
		status:     MemberStatusDTO{IsMember: false},
		ensureResp: dp7575.UserMapEnsureResponse{SiteID: "yangguangzhan", ExternalUserID: "oerx", IsMapped: true},
	}
	svc := NewService(repo, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "资源包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 19.9}}, &resourceOrderRepositoryStub{})

	result, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: publicUserID, LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})
	require.NoError(t, err)
	require.False(t, result.AccessGranted)
	require.Equal(t, publicUserID, client.ensureReq.ExternalUserID)
	require.Equal(t, "oerx", client.req.ExternalUserID)
	require.NotNil(t, repo.upserted)
	require.Equal(t, "oerx", repo.upserted.ExternalUserID)
}

func TestService_CheckResourceAccess_ReturnsRepositoryErrorWhenBindingLookupFails(t *testing.T) {
	repo := &bindingRepositoryStub{err: context.DeadlineExceeded}
	client := &memberStatusClientStub{}
	svc := NewService(repo, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{resourceByID: ResourceRecordDTO{ResourceID: "res_demo_1", Title: "资源包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 19.9}}, &resourceOrderRepositoryStub{})

	_, err := svc.CheckResourceAccess(context.Background(), &ResourceAccessCheckActorDTO{UserID: 1001, ExternalUserID: "user_public_123", LoggedIn: true}, ResourceAccessCheckRequestDTO{ResourceID: "res_demo_1"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, client.ensureCalls)
}

func TestService_GetMemberProfile(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{
		profileResp: dp7575.MemberProfileResponse{
			IsMember:        true,
			MemberLevelName: "年度会员",
			MemberExpireAt:  "2027-04-18T00:00:00Z",
			HistorySummary: dp7575.MemberHistorySummary{
				LatestOrderNo:     "VIP20260418001",
				LatestOrderStatus: "paid",
				LatestOrderAmount: "99.00",
				LatestOrderTime:   "2026-04-18 09:32:00",
			},
		},
	}

	svc := NewService(repo, client)
	profile, err := svc.GetMemberProfile(context.Background(), 1001)
	require.NoError(t, err)
	require.True(t, profile.IsMember)
	require.Equal(t, "年度会员", profile.Level)
	require.Len(t, profile.RecentOrders, 1)
	require.Equal(t, "VIP20260418001", profile.RecentOrders[0].OrderNo)
}

func TestService_GetHealthCheck(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "dp-user-001",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{
		status: MemberStatusDTO{IsMember: true, Level: "会员", ExpiresAt: "2027-04-18T00:00:00Z"},
		probe:  dp7575.HealthProbeResult{Connected: true, SignatureValid: true, Detail: "success"},
	}

	svc := NewService(repo, client)
	result, err := svc.GetHealthCheck(context.Background(), 1001)
	require.NoError(t, err)
	require.Len(t, result.Items, 6)
	require.Equal(t, "接入配置是否完整", result.Items[0].Name)
	require.Equal(t, "通过", result.Items[0].Result)
	require.Equal(t, "用户映射是否存在", result.Items[3].Name)
	require.Equal(t, "已绑定", result.Items[3].Result)
	require.Equal(t, "当前会员状态", result.Items[5].Name)
	require.Equal(t, "已开通会员", result.Items[5].Result)
}

func TestService_GetHealthCheck_WhenBindingMissing_MarksDependentItemsAsWarning(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())

	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{
		probe: dp7575.HealthProbeResult{Connected: true, SignatureValid: true, Detail: "success"},
	}

	svc := NewService(repo, client)
	result, err := svc.GetHealthCheck(context.Background(), 1001)
	require.NoError(t, err)
	require.Len(t, result.Items, 6)
	require.Equal(t, "未绑定", result.Items[3].Result)
	require.Contains(t, result.Items[3].Detail, "external_user_id")
	require.Equal(t, "待处理", result.Items[4].Result)
	require.Contains(t, result.Items[4].Detail, "请先在极光库完成用户映射")
	require.Equal(t, "待处理", result.Items[5].Result)
	require.Contains(t, result.Items[5].Detail, "请先在极光库完成用户映射")
}

func TestService_AutoBindAfterLogin_SkipsWhenBindingExists(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "existing",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{}

	svc := NewService(repo, client)
	externalUserID, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.NoError(t, err)
	require.Equal(t, "existing", externalUserID)
	require.Zero(t, client.ensureCalls)
	require.Equal(t, 0, repo.upsertCalls)
}

func TestService_AutoBindAfterLogin_ReconcilesLegacyBindingWithEnsureResult(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{ensureResp: dp7575.UserMapEnsureResponse{
		SiteID:         "yangguangzhan",
		ExternalUserID: "oerx",
		WPUserID:       7788,
		IsMapped:       true,
		Mapped:         true,
		Action:         "existing",
	}}

	svc := NewService(repo, client)
	externalUserID, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.NoError(t, err)
	require.Equal(t, "oerx", externalUserID)
	require.Equal(t, 1, client.ensureCalls)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.NotNil(t, repo.upserted)
	require.Equal(t, "oerx", repo.upserted.ExternalUserID)
}

func TestService_AutoBindAfterLogin_PersistsRemoteMappingWhenFound(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{ensureResp: dp7575.UserMapEnsureResponse{
		SiteID:         "yangguangzhan",
		ExternalUserID: "user_public_123",
		WPUserID:       7788,
		IsMapped:       true,
		Mapped:         true,
		Action:         "existing",
	}}

	svc := NewService(repo, client)
	externalUserID, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.NoError(t, err)
	require.Equal(t, "user_public_123", externalUserID)
	require.Equal(t, 1, client.ensureCalls)
	require.Equal(t, "user_public_123", client.ensureReq.ExternalUserID)
	require.NotNil(t, repo.upserted)
	require.Equal(t, int64(1001), repo.upserted.UserID)
	require.Equal(t, "user_public_123", repo.upserted.ExternalUserID)
	require.Equal(t, "yangguangzhan", repo.upserted.SiteID)
	require.Equal(t, "active", repo.upserted.Status)
	require.NotNil(t, repo.upserted.LastSyncedAt)
}

func TestService_AutoBindAfterLogin_CreatesBindingViaEnsureWhenMissing(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{ensureResp: dp7575.UserMapEnsureResponse{
		SiteID:         "yangguangzhan",
		ExternalUserID: "user_public_123",
		WPUserID:       7788,
		IsMapped:       true,
		Mapped:         true,
		Action:         "created",
	}}

	svc := NewService(repo, client)
	externalUserID, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.NoError(t, err)
	require.Equal(t, "user_public_123", externalUserID)
	require.Equal(t, 1, client.ensureCalls)
	require.NotNil(t, repo.upserted)
	require.Equal(t, "user_public_123", repo.upserted.ExternalUserID)
	require.Equal(t, "yangguangzhan", repo.upserted.SiteID)
	require.Equal(t, int64(1001), repo.upserted.UserID)
}

func TestService_AutoBindAfterLogin_ReturnsErrorWhenEnsureFails(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{ensureErr: context.DeadlineExceeded}

	svc := NewService(repo, client)
	_, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, repo.upserted)
}

func TestService_AutoBindAfterLogin_SkipsWhenEnsureClientNotConfigured(t *testing.T) {
	repo := &bindingRepositoryStub{err: ErrMemberBindingNotFound}
	client := &memberStatusClientStub{ensureErr: dp7575.ErrNotConfigured}

	svc := NewService(repo, client)
	externalUserID, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.NoError(t, err)
	require.Equal(t, "user_public_123", externalUserID)
	require.Equal(t, 1, client.ensureCalls)
	require.Nil(t, repo.upserted)
}

func TestService_AutoBindAfterLogin_ReturnsRepositoryErrorWhenBindingLookupFails(t *testing.T) {
	repo := &bindingRepositoryStub{err: context.DeadlineExceeded}
	client := &memberStatusClientStub{}

	svc := NewService(repo, client)
	_, err := svc.AutoBindAfterLogin(context.Background(), 1001, "user_public_123")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, client.ensureCalls)
	require.Nil(t, repo.upserted)
}

func TestService_GetMemberPurchaseCatalog_GroupsProductsByMemberLevel(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{catalogResp: dp7575.MemberProductsCatalogResponse{
		Products: []dp7575.MemberProduct{
			{ProductID: "vip_2_1_upgrade", MemberLevel: 2, MemberLevelName: "钻石会员", Title: "钻石会员升级", Description: "永久", Price: 199, OriginalPrice: 299, ActionType: "upgrade", Meta: dp7575.MemberProductMeta{Tag: "补差价"}},
			{ProductID: "vip_1_0_pay", MemberLevel: 1, MemberLevelName: "普通会员", Title: "普通会员购买", Description: "12个月", Price: 99, OriginalPrice: 199, ActionType: "pay", Meta: dp7575.MemberProductMeta{Tag: "首购"}},
			{ProductID: "vip_1_1_renew", MemberLevel: 1, MemberLevelName: "普通会员", Title: "普通会员续费", Description: "12个月", Price: 79, OriginalPrice: 99, ActionType: "renew", Meta: dp7575.MemberProductMeta{Tag: "续费优惠"}},
		},
	}}

	svc := NewService(repo, client)
	result, err := svc.GetMemberPurchaseCatalog(context.Background(), 1001)
	require.NoError(t, err)
	require.Len(t, result.MemberTypes, 2)
	require.Equal(t, "1", result.MemberTypes[0].Level)
	require.Equal(t, "普通会员", result.MemberTypes[0].Name)
	require.Len(t, result.MemberTypes[0].PriceOptions, 2)
	require.Equal(t, "vip_1_0_pay", result.MemberTypes[0].PriceOptions[0].ProductID)
	require.Equal(t, "首购", result.MemberTypes[0].PriceOptions[0].Tag)
	require.Equal(t, "2", result.MemberTypes[1].Level)
	require.Equal(t, "vip_2_1_upgrade", result.MemberTypes[1].PriceOptions[0].ProductID)
	require.Equal(t, []string{"wechat", "alipay", "card"}, result.PaymentMethods)
}

func TestService_GetMemberPurchaseCatalog_ReturnsRepositoryErrorWhenBindingLookupFails(t *testing.T) {
	repo := &bindingRepositoryStub{err: context.DeadlineExceeded}
	client := &memberStatusClientStub{}

	svc := NewService(repo, client)
	_, err := svc.GetMemberPurchaseCatalog(context.Background(), 1001)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestService_CreateMemberOrder_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{orderResp: dp7575.OrderCreateResponse{
		ZibOrderNum: "VIP20260418001",
		OrderStatus: "pending",
		PayType:     "wechat",
		PayURL:      "https://pay.example.com/wechat/VIP20260418001",
	}}

	svc := NewService(repo, client)
	result, err := svc.CreateMemberOrder(context.Background(), 1001, MemberOrderCreateDTO{
		ProductID:     "vip_1_0_pay",
		PaymentMethod: "wechat",
	})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.orderReq.ExternalUserID)
	require.Equal(t, "vip", client.orderReq.ProductType)
	require.Equal(t, "vip_1_0_pay", client.orderReq.ProductID)
	require.Equal(t, "wechat", client.orderReq.PaymentMethod)
	require.Equal(t, "VIP20260418001", result.OrderNo)
	require.Equal(t, "wechat", result.PayType)
}

func TestService_GetMemberPaymentDetail_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{paymentResp: dp7575.OrderPaymentDetailResponse{
		OrderNum:   "VIP20260418001",
		Amount:     199,
		PayType:    "alipay",
		PayChannel: "alipay",
		PayDetail: map[string]any{
			"url_qrcode": "data:image/png;base64,ALIPAYQR",
			"qr_code":    "https://qr.example.com/alipay/VIP20260418001",
		},
	}}

	svc := NewService(repo, client)
	result, err := svc.GetMemberPaymentDetail(context.Background(), 1001, MemberOrderPaymentDetailDTO{OrderNo: "VIP20260418001"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.paymentReq.ExternalUserID)
	require.Equal(t, "VIP20260418001", client.paymentReq.ZibOrderNum)
	require.Equal(t, "VIP20260418001", result.OrderNo)
	require.Equal(t, "alipay", result.PayType)
	require.Equal(t, "data:image/png;base64,ALIPAYQR", result.PayDetail["url_qrcode"])
}

func TestService_RedeemMemberCard_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{redeemResp: dp7575.CardRedeemCreateResponse{
		RedeemStatus:  "success",
		TargetType:    "member",
		TargetSummary: "会员兑换：等级 1",
		OrderNum:      "26041820010001",
		EffectSummary: "兑换成功",
	}}

	svc := NewService(repo, client)
	result, err := svc.RedeemMemberCard(context.Background(), 1001, MemberCardRedeemDTO{
		CardCode:     "VIP-CARD-001",
		CardPassword: "SECURE-001",
	})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.redeemReq.ExternalUserID)
	require.Equal(t, "VIP-CARD-001", client.redeemReq.CardCode)
	require.Equal(t, "SECURE-001", client.redeemReq.CardPassword)
	require.Equal(t, "success", result.Status)
	require.Equal(t, "26041820010001", result.OrderNo)
	require.Equal(t, "兑换成功", result.Message)
}

func TestService_GetMemberOrders_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{orderListResp: dp7575.OrderListResponse{
		List: []dp7575.OrderListItem{
			{OrderNum: "VIP20260418001", BusinessType: "member", ProductType: "vip", Status: "paid", Amount: 99, PayType: "wechat", CreateTime: "2026-04-18 09:32:00"},
			{OrderNum: "RES20260418002", BusinessType: "resource", ProductType: "resource", Status: "pending", Amount: 29.9, PayType: "alipay", CreateTime: "2026-04-18 10:00:00"},
		},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 10, Total: 2},
	}}

	svc := NewService(repo, client)
	result, err := svc.GetMemberOrders(context.Background(), 1001)
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.orderListReq.ExternalUserID)
	require.Len(t, result.List, 2)
	require.Equal(t, "VIP20260418001", result.List[0].OrderNo)
	require.Equal(t, "member", result.List[0].BusinessType)
	require.Equal(t, "RES20260418002", result.List[1].OrderNo)
	require.Equal(t, "resource", result.List[1].BusinessType)
}

func TestService_GetMemberOrders_UsesLocalResourceSnapshotTitle(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{orderListResp: dp7575.OrderListResponse{
		List:       []dp7575.OrderListItem{{OrderNum: "RES20260418002", BusinessType: "resource_purchase", ProductType: "resource", Status: "pending", Amount: 29.9, PayType: "wechat", CreateTime: "2026-04-18 10:00:00"}},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 10, Total: 1},
	}}
	orderRepo := &resourceOrderRepositoryStub{byExternalOrderNo: map[string]ResourceOrderRecordDTO{
		"RES20260418002": {
			BusinessOrderNo: "YGZ_RES_002",
			ExternalOrderNo: "RES20260418002",
			ResourceID:      "res_real_1",
			Snapshot:        map[string]any{"product": map[string]any{"title": "前端项目源码包"}},
		},
	}}

	svc := NewService(repo, client)
	svc.SetResourceRepositories(&resourceRepositoryStub{}, orderRepo)

	result, err := svc.GetMemberOrders(context.Background(), 1001)
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	require.Equal(t, "前端项目源码包", result.List[0].ProductTitle)
}

func TestService_ListAdminOrderMappings(t *testing.T) {
	client := &memberStatusClientStub{adminOrderMappingsResp: dp7575.AdminOrderMappingListResponse{
		Summary: dp7575.AdminOrderMappingSummary{Total: 2, SiteIDZeroCount: 1, LatestCreatedAt: "2026-04-19 15:58:07"},
		List: []dp7575.AdminOrderMappingItem{{
			ZibOrderNum:     "2604191558069513850",
			ExternalUserID:  "oerx",
			WpUserID:        7,
			ProductType:     "custom_amount",
			PostID:          0,
			ResourceID:      0,
			CreatedAt:       "2026-04-19 15:58:07",
			StoredSiteID:    "yangguangzhan",
			ResolvedSiteID:  "yangguangzhan",
			SnapshotSiteID:  "yangguangzhan",
			ContextSource:   "snapshot",
			IsDirty:         false,
			RequestSnapshot: map[string]any{"_ygz_site": map[string]any{"site_id": "yangguangzhan"}},
		}},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 20, Total: 2},
	}}
	svc := NewService(newStubBindingRepo(), client)

	result, err := svc.ListAdminOrderMappings(context.Background(), AdminOrderMappingListQueryDTO{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 2, result.Summary.Total)
	require.Len(t, result.List, 1)
	require.Equal(t, "2604191558069513850", result.List[0].ZibOrderNum)
	require.Equal(t, 20, client.adminOrderMappingsReq.PageSize)
}

func TestService_ListAdminCards(t *testing.T) {
	client := &memberStatusClientStub{adminCardsResp: dp7575.AdminCardListResponse{
		List: []dp7575.AdminCardItem{{
			CardCode:     "23282285775808818733",
			CardPassword: "Uu5WKelcw4Z86SfdonM9Kz1l09y6FBPrR7v",
			CardType:     "vip_exchange",
			Status:       "used",
			Remark:       "cardpass_20260418222640",
			CreatedAt:    "2026-04-18 22:27:00",
			UpdatedAt:    "2026-04-18 22:33:43",
		}},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 20, Total: 1},
	}}
	svc := NewService(newStubBindingRepo(), client)

	result, err := svc.ListAdminCards(context.Background(), AdminCardListQueryDTO{Status: "used", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, "used", client.adminCardsReq.Status)
	require.Len(t, result.List, 1)
	require.Equal(t, "23282285775808818733", result.List[0].CardCode)
	require.Equal(t, 1, result.Pagination.Total)
}

func TestService_GetMemberOrderDetail_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{orderDetailResp: dp7575.OrderDetailResponse{
		OrderNum: "VIP20260418001", BusinessType: "member", ProductType: "vip", ProductID: "vip_1_0_pay",
		Status: "paid", Amount: 99, PayType: "wechat", PayTime: "2026-04-18 09:35:00", CreateTime: "2026-04-18 09:32:00",
		Snapshot: map[string]any{"product": map[string]any{"title": "年度会员"}},
		ZibOrder: map[string]any{"order_id": 12345},
	}}

	svc := NewService(repo, client)
	result, err := svc.GetMemberOrderDetail(context.Background(), 1001, MemberOrderDetailRequestDTO{OrderNo: "VIP20260418001"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.orderDetailReq.ExternalUserID)
	require.Equal(t, "VIP20260418001", client.orderDetailReq.ZibOrderNum)
	require.Equal(t, "VIP20260418001", result.OrderNo)
	require.Equal(t, "member", result.BusinessType)
	require.Equal(t, "年度会员", result.Snapshot["product"].(map[string]any)["title"])
}

func TestService_GetMemberOrderStatus_UsesBindingExternalUserID(t *testing.T) {
	repo := &bindingRepositoryStub{binding: MemberBindingDTO{
		UserID:         1001,
		ExternalUserID: "user_public_123",
		SiteID:         "yangguangzhan",
		Status:         "active",
	}}
	client := &memberStatusClientStub{orderStatusResp: dp7575.OrderStatusResponse{
		BusinessType: "member", ZibOrderNum: "VIP20260418001", ProductType: "vip",
		OrderStatus: "paid", OrderStatusLabel: "已支付", OrderPrice: 99, PayPrice: 99,
		PayType: "wechat", CreatedAt: "2026-04-18 09:32:00", PaidAt: "2026-04-18 09:35:00",
	}}

	svc := NewService(repo, client)
	result, err := svc.GetMemberOrderStatus(context.Background(), 1001, MemberOrderStatusRequestDTO{OrderNo: "VIP20260418001"})
	require.NoError(t, err)
	require.Equal(t, "user_public_123", client.orderStatusReq.ExternalUserID)
	require.Equal(t, "VIP20260418001", client.orderStatusReq.ZibOrderNum)
	require.Equal(t, "member", result.BusinessType)
	require.Equal(t, "paid", result.Status)
	require.Equal(t, "已支付", result.StatusLabel)
}
