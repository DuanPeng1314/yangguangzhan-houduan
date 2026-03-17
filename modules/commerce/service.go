package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/model"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/repository"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
)

type AccessContext struct {
	Authorization       string
	ArticleGuestToken   string
	ResourceGuestTokens map[string]string
	GuestEmail          string
}

type PurchaseTarget struct {
	ArticleID      string
	ArticleTitle   string
	ResourceType   string
	ResourceID     string
	PriceCent      int
	MembershipFree bool
	Subject        string
	PurchaseKey    string
}

type PurchaseOrderResult struct {
	OrderNo    string `json:"order_no"`
	QRCode     string `json:"qr_code"`
	Amount     int    `json:"amount"`
	GuestToken string `json:"guest_token,omitempty"`
}

type GuestOrderStatusResult struct {
	OrderNo       string `json:"order_no"`
	Status        string `json:"status"`
	Amount        int    `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	PaidAt        string `json:"paid_at,omitempty"`
}

type MemberAuthorizeResult struct {
	AuthorizeURL string `json:"authorize_url"`
}

type MemberUserProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Verified  bool   `json:"verified"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type MemberMembershipProfile struct {
	TierID          string `json:"tier_id,omitempty"`
	TierName        string `json:"tier_name,omitempty"`
	TierDisplayName string `json:"tier_display_name,omitempty"`
	IsActive        bool   `json:"is_active"`
	ExpiresAt       string `json:"expires_at,omitempty"`
}

type MemberDashboardResult struct {
	User            MemberUserProfile       `json:"user"`
	Membership      MemberMembershipProfile `json:"membership"`
	RecentOrders    []interface{}           `json:"recent_orders"`
	RecentPurchases []interface{}           `json:"recent_purchases"`
	Stats           struct {
		TotalOrders     int `json:"total_orders"`
		PaidOrders      int `json:"paid_orders"`
		TotalPurchases  int `json:"total_purchases"`
		ActivePurchases int `json:"active_purchases"`
	} `json:"stats"`
}

type MemberSessionResult struct {
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token,omitempty"`
	TokenType    string                `json:"token_type"`
	ExpiresIn    int                   `json:"expires_in"`
	ExpiresAt    string                `json:"expires_at"`
	Dashboard    MemberDashboardResult `json:"dashboard"`
}

type MemberTier struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	PriceMonthly  int    `json:"price_monthly"`
	PriceYearly   int    `json:"price_yearly"`
	PriceLifetime int    `json:"price_lifetime"`
	SortOrder     int    `json:"sort_order"`
	Active        bool   `json:"active"`
}

type MemberTierListResult struct {
	Tiers []MemberTier `json:"tiers"`
	Total int          `json:"total"`
}

type MemberRedeemResult struct {
	Success   bool                  `json:"success"`
	Dashboard MemberDashboardResult `json:"dashboard"`
}

type Service struct {
	settingSvc  setting.SettingService
	articleRepo repository.ArticleRepository
}

func NewService(settingSvc setting.SettingService, articleRepo repository.ArticleRepository) *Service {
	return &Service{settingSvc: settingSvc, articleRepo: articleRepo}
}

func (s *Service) DecoratePublicArticle(ctx context.Context, detail *model.ArticleDetailResponse, accessCtx AccessContext) *model.ArticleDetailResponse {
	if detail == nil {
		return nil
	}
	articleCfg := s.normalizeArticleConfig(detail)
	resourceCfgs := s.normalizeResourceConfigs(detail)
	if articleCfg == nil && len(resourceCfgs) == 0 {
		return detail
	}
	state := &model.ArticleCommerceState{Enabled: s.isEnabled(), PurchaseEnabled: s.purchaseConfigured()}
	articleState := s.resolveArticleState(ctx, articleCfg, accessCtx)
	if articleState != nil {
		state.Article = articleState
		if articleState.AccessType == "paid" && !articleState.HasAccess {
			detail.ContentHTML = s.buildPreviewHTML(detail, articleState)
		}
	}
	resourceStates := s.resolveResourceStates(ctx, resourceCfgs, articleState, accessCtx)
	if len(resourceStates) > 0 {
		state.Resources = resourceStates
	}
	detail.ExtraConfig = s.sanitizeExtraConfig(detail.ExtraConfig, articleCfg, resourceCfgs, resourceStates)
	detail.Commerce = state
	return detail
}

func (s *Service) ResolvePurchaseTarget(ctx context.Context, articleID string, resourceID string) (*PurchaseTarget, error) {
	article, err := s.articleRepo.GetBySlugOrIDForPreview(ctx, articleID)
	if err != nil {
		return nil, err
	}
	articleCfg := s.normalizeArticleConfigFromArticle(article)
	resourceCfgs := s.normalizeResourceConfigsFromArticle(article)
	if resourceID == "" {
		if articleCfg == nil || articleCfg.AccessType != "paid" || articleCfg.PriceCent <= 0 {
			return nil, fmt.Errorf("文章未启用付费购买")
		}
		return &PurchaseTarget{
			ArticleID:      article.ID,
			ArticleTitle:   article.Title,
			ResourceType:   articleCfg.ResourceType,
			ResourceID:     articleCfg.ResourceID,
			PriceCent:      articleCfg.PriceCent,
			MembershipFree: boolValue(articleCfg.MembershipFree),
			Subject:        fmt.Sprintf("购买文章《%s》", article.Title),
			PurchaseKey:    articleCfg.ResourceID,
		}, nil
	}
	for _, resource := range resourceCfgs {
		if resource == nil {
			continue
		}
		if resource.ID != resourceID && resource.ResourceID != resourceID {
			continue
		}
		if boolValue(resource.GrantWithArticle) {
			if articleCfg == nil || articleCfg.AccessType != "paid" || articleCfg.PriceCent <= 0 {
				return nil, fmt.Errorf("当前资源随文章解锁，但文章未启用付费购买")
			}
			return &PurchaseTarget{
				ArticleID:      article.ID,
				ArticleTitle:   article.Title,
				ResourceType:   articleCfg.ResourceType,
				ResourceID:     articleCfg.ResourceID,
				PriceCent:      articleCfg.PriceCent,
				MembershipFree: boolValue(articleCfg.MembershipFree),
				Subject:        fmt.Sprintf("购买文章《%s》", article.Title),
				PurchaseKey:    articleCfg.ResourceID,
			}, nil
		}
		if resource.AccessType != "paid" || resource.PriceCent <= 0 {
			return nil, fmt.Errorf("当前资源未启用单独购买")
		}
		subject := strings.TrimSpace(resource.Title)
		if subject == "" {
			subject = fmt.Sprintf("购买文章资源《%s》", article.Title)
		}
		return &PurchaseTarget{
			ArticleID:      article.ID,
			ArticleTitle:   article.Title,
			ResourceType:   resource.ResourceType,
			ResourceID:     resource.ResourceID,
			PriceCent:      resource.PriceCent,
			MembershipFree: boolValue(resource.MembershipFree),
			Subject:        subject,
			PurchaseKey:    resource.ID,
		}, nil
	}
	return nil, fmt.Errorf("未找到对应的付费目标")
}

func (s *Service) CreateUserPurchase(ctx context.Context, token string, target *PurchaseTarget, paymentMethod string) (*PurchaseOrderResult, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("缺少统一登录令牌")
	}
	payload := map[string]interface{}{
		"product_type":   "resource",
		"product_id":     target.ResourceID,
		"amount":         target.PriceCent,
		"subject":        target.Subject,
		"payment_method": paymentMethod,
		"extra": map[string]interface{}{
			"site_id":         s.siteID(),
			"resource_type":   target.ResourceType,
			"resource_id":     target.ResourceID,
			"membership_free": target.MembershipFree,
		},
	}
	var result PurchaseOrderResult
	if err := s.requestJSON(ctx, http.MethodPost, "/payments/create", token, payload, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) CreateGuestPurchase(ctx context.Context, email string, target *PurchaseTarget, paymentMethod string) (*PurchaseOrderResult, error) {
	if strings.TrimSpace(email) == "" {
		return nil, fmt.Errorf("游客购买需要邮箱")
	}
	payload := map[string]interface{}{
		"product_type":   "resource",
		"product_id":     target.ResourceID,
		"amount":         target.PriceCent,
		"subject":        target.Subject,
		"payment_method": paymentMethod,
		"email":          email,
		"extra": map[string]interface{}{
			"site_id":         s.siteID(),
			"resource_type":   target.ResourceType,
			"resource_id":     target.ResourceID,
			"membership_free": target.MembershipFree,
		},
	}
	var result PurchaseOrderResult
	if err := s.requestJSON(ctx, http.MethodPost, "/payments/guest-create", "", payload, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetGuestOrderStatus(ctx context.Context, orderNo string, guestToken string) (*GuestOrderStatusResult, error) {
	if strings.TrimSpace(orderNo) == "" || strings.TrimSpace(guestToken) == "" {
		return nil, fmt.Errorf("缺少订单号或游客凭证")
	}
	path := fmt.Sprintf("/payments/guest/%s/status?guest_token=%s", orderNo, guestToken)
	var result GuestOrderStatusResult
	if err := s.requestJSON(ctx, http.MethodGet, path, "", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) BuildMemberAuthorizeURL(redirectURI string, state string) (*MemberAuthorizeResult, error) {
	if !s.memberAuthConfigured() {
		return nil, fmt.Errorf("统一登录配置未完成")
	}
	if strings.TrimSpace(redirectURI) == "" || strings.TrimSpace(state) == "" {
		return nil, fmt.Errorf("缺少回调地址或状态参数")
	}
	if _, err := url.ParseRequestURI(redirectURI); err != nil {
		return nil, fmt.Errorf("回调地址无效")
	}
	query := url.Values{}
	query.Set("client_id", s.clientID())
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("state", state)
	authorizeURL := fmt.Sprintf("%s/oauth/authorize?%s", s.rootURL(), query.Encode())
	return &MemberAuthorizeResult{AuthorizeURL: authorizeURL}, nil
}

func (s *Service) ExchangeMemberToken(ctx context.Context, code string, redirectURI string) (*MemberSessionResult, error) {
	if !s.memberAuthConfigured() {
		return nil, fmt.Errorf("统一登录配置未完成")
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(redirectURI) == "" {
		return nil, fmt.Errorf("缺少授权码或回调地址")
	}
	payload := map[string]interface{}{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": redirectURI,
		"client_id":    s.clientID(),
	}
	var tokenResult remoteOAuthTokenResult
	if err := s.requestJSONRoot(ctx, http.MethodPost, "/oauth/token", "", payload, nil, &tokenResult); err != nil {
		return nil, err
	}
	return s.buildMemberSession(ctx, tokenResult)
}

func (s *Service) RefreshMemberToken(ctx context.Context, refreshToken string) (*MemberSessionResult, error) {
	if !s.memberAuthConfigured() {
		return nil, fmt.Errorf("统一登录配置未完成")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("缺少刷新令牌")
	}
	payload := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	var tokenResult remoteOAuthTokenResult
	if err := s.requestJSONRoot(ctx, http.MethodPost, "/oauth/token", "", payload, nil, &tokenResult); err != nil {
		return nil, err
	}
	tokenResult.RefreshToken = refreshToken
	return s.buildMemberSession(ctx, tokenResult)
}

func (s *Service) GetMemberDashboard(ctx context.Context, token string) (*MemberDashboardResult, error) {
	bearerToken := normalizeBearerToken(token)
	if bearerToken == "" {
		return nil, fmt.Errorf("缺少会员登录令牌")
	}
	var meResult remoteUserMeResponse
	if err := s.requestJSON(ctx, http.MethodGet, "/users/me", bearerToken, nil, nil, &meResult); err != nil {
		return nil, err
	}
	var permissionResult remotePermissionVerifyResponse
	if err := s.requestJSON(ctx, http.MethodGet, "/permissions/verify", bearerToken, nil, nil, &permissionResult); err != nil {
		permissionResult = remotePermissionVerifyResponse{}
	}
	result := &MemberDashboardResult{
		User: MemberUserProfile{
			ID:        meResult.ID,
			Email:     meResult.Email,
			Name:      strings.TrimSpace(meResult.Name),
			Verified:  meResult.Verified,
			CreatedAt: meResult.Created,
			UpdatedAt: meResult.Updated,
		},
		Membership: MemberMembershipProfile{
			TierID:          meResult.CurrentTier,
			TierName:        firstNonEmpty(permissionResult.TierName, permissionResult.Tier, meResult.TierInfo.Name),
			TierDisplayName: firstNonEmpty(meResult.TierInfo.DisplayName, permissionResult.TierName, permissionResult.Tier),
			IsActive:        permissionResult.IsActive,
			ExpiresAt:       firstNonEmpty(permissionResult.ExpiresAt, meResult.TierExpiresAt),
		},
		RecentOrders:    make([]interface{}, 0),
		RecentPurchases: make([]interface{}, 0),
	}
	if result.User.Name == "" {
		result.User.Name = result.User.Email
	}
	if result.Membership.TierDisplayName == "" && result.Membership.TierName == "" {
		result.Membership.TierDisplayName = "免费用户"
	}
	return result, nil
}

func (s *Service) GetMemberTiers(ctx context.Context) (*MemberTierListResult, error) {
	var result MemberTierListResult
	if err := s.requestJSON(ctx, http.MethodGet, "/tiers", "", nil, nil, &result); err != nil {
		return nil, err
	}
	sort.SliceStable(result.Tiers, func(i, j int) bool {
		return result.Tiers[i].SortOrder < result.Tiers[j].SortOrder
	})
	return &result, nil
}

func (s *Service) RedeemMemberKey(ctx context.Context, token string, key string) (*MemberRedeemResult, error) {
	bearerToken := normalizeBearerToken(token)
	if bearerToken == "" {
		return nil, fmt.Errorf("缺少会员登录令牌")
	}
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("请输入卡密")
	}
	payload := map[string]interface{}{
		"key": strings.TrimSpace(key),
	}
	var remoteResult struct {
		Success bool `json:"success"`
	}
	if err := s.requestJSON(ctx, http.MethodPost, "/keys/redeem", bearerToken, payload, nil, &remoteResult); err != nil {
		return nil, err
	}
	dashboard, err := s.GetMemberDashboard(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	return &MemberRedeemResult{
		Success:   remoteResult.Success,
		Dashboard: *dashboard,
	}, nil
}

func (s *Service) purchaseConfigured() bool {
	return s.isEnabled() && s.baseURL() != "" && s.siteID() != ""
}

func (s *Service) accessConfigured() bool {
	return s.purchaseConfigured() && s.siteAPIKey() != ""
}

func (s *Service) memberAuthConfigured() bool {
	return s.isEnabled() && s.baseURL() != "" && s.clientID() != ""
}

func (s *Service) isEnabled() bool {
	return s.settingSvc.GetBool(constant.KeyCommerceEnabled.String())
}

func (s *Service) baseURL() string {
	return strings.TrimRight(strings.TrimSpace(s.settingSvc.Get(constant.KeyCommerceAuthCenterURL.String())), "/")
}

func (s *Service) siteID() string {
	return strings.TrimSpace(s.settingSvc.Get(constant.KeyCommerceSiteID.String()))
}

func (s *Service) clientID() string {
	return strings.TrimSpace(s.settingSvc.Get(constant.KeyCommerceClientID.String()))
}

func (s *Service) siteAPIKey() string {
	return strings.TrimSpace(s.settingSvc.Get(constant.KeyCommerceSiteAPIKey.String()))
}

func (s *Service) timeout() time.Duration {
	value := strings.TrimSpace(s.settingSvc.Get(constant.KeyCommerceRequestTimeoutMs.String()))
	if value == "" {
		return 10 * time.Second
	}
	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return 10 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *Service) rootURL() string {
	base := s.baseURL()
	for _, suffix := range []string{"/api/v1", "/api"} {
		if strings.HasSuffix(base, suffix) {
			return strings.TrimSuffix(base, suffix)
		}
	}
	return base
}

func (s *Service) resolveArticleState(ctx context.Context, cfg *model.ArticleCommerceArticleConfig, accessCtx AccessContext) *model.ArticleCommerceArticleState {
	if cfg == nil {
		return nil
	}
	state := &model.ArticleCommerceArticleState{
		ResourceType:   cfg.ResourceType,
		ResourceID:     cfg.ResourceID,
		AccessType:     cfg.AccessType,
		PriceCent:      cfg.PriceCent,
		PreviewMode:    cfg.PreviewMode,
		PreviewText:    cfg.PreviewText,
		PreviewLimit:   cfg.PreviewLimit,
		MembershipFree: boolValue(cfg.MembershipFree),
	}
	if cfg.AccessType != "paid" {
		state.HasAccess = true
		state.AccessSource = "free"
		return state
	}
	state.RequiresPurchase = true
	if !s.accessConfigured() {
		return state
	}
	if strings.TrimSpace(accessCtx.Authorization) != "" {
		results, err := s.checkResourcesBatch(ctx, accessCtx.Authorization, []resourceRef{{Type: cfg.ResourceType, ID: cfg.ResourceID}})
		if err != nil {
			log.Printf("[commerce] 批量校验文章权限失败: %v", err)
			return state
		}
		if result, ok := results[cfg.ResourceID]; ok && result.HasAccess {
			state.HasAccess = true
			state.RequiresPurchase = false
			state.AccessSource = result.AccessType
		}
		return state
	}
	if strings.TrimSpace(accessCtx.ArticleGuestToken) == "" {
		return state
	}
	result, err := s.checkResourceSingle(ctx, cfg.ResourceType, cfg.ResourceID, accessCtx.ArticleGuestToken)
	if err != nil {
		log.Printf("[commerce] 游客校验文章权限失败: %v", err)
		return state
	}
	if result.HasAccess {
		state.HasAccess = true
		state.RequiresPurchase = false
		if result.IsMember {
			state.AccessSource = "membership"
		} else {
			state.AccessSource = "purchase"
		}
	}
	return state
}

func (s *Service) resolveResourceStates(ctx context.Context, resources []*model.ArticleCommerceResourceConfig, articleState *model.ArticleCommerceArticleState, accessCtx AccessContext) []*model.ArticleCommerceResourceState {
	if len(resources) == 0 {
		return nil
	}
	states := make([]*model.ArticleCommerceResourceState, 0, len(resources))
	batchTargets := make([]resourceRef, 0)
	batchIndex := make(map[string]*model.ArticleCommerceResourceState)
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		state := &model.ArticleCommerceResourceState{
			ID:               resource.ID,
			Title:            resource.Title,
			Description:      resource.Description,
			Provider:         resource.Provider,
			ResourceType:     resource.ResourceType,
			ResourceID:       resource.ResourceID,
			AccessType:       resource.AccessType,
			PriceCent:        resource.PriceCent,
			GrantWithArticle: boolValue(resource.GrantWithArticle),
			MembershipFree:   boolValue(resource.MembershipFree),
			Sort:             resource.Sort,
		}
		switch {
		case resource.AccessType != "paid":
			state.HasAccess = true
			state.AccessSource = "free"
			state.URL = resource.URL
			state.ExtractCode = resource.ExtractCode
		case boolValue(resource.GrantWithArticle) && articleState != nil && articleState.HasAccess:
			state.HasAccess = true
			state.AccessSource = "article"
			state.URL = resource.URL
			state.ExtractCode = resource.ExtractCode
		case strings.TrimSpace(accessCtx.Authorization) != "" && s.accessConfigured():
			batchTargets = append(batchTargets, resourceRef{Type: resource.ResourceType, ID: resource.ResourceID})
			batchIndex[resource.ResourceID] = state
		case strings.TrimSpace(accessCtx.ResourceGuestTokens[resource.ID]) != "":
			result, err := s.checkResourceSingle(ctx, resource.ResourceType, resource.ResourceID, accessCtx.ResourceGuestTokens[resource.ID])
			if err == nil && result.HasAccess {
				state.HasAccess = true
				state.AccessSource = "purchase"
				state.URL = resource.URL
				state.ExtractCode = resource.ExtractCode
			}
		}
		states = append(states, state)
	}
	if len(batchTargets) > 0 {
		results, err := s.checkResourcesBatch(ctx, accessCtx.Authorization, batchTargets)
		if err != nil {
			log.Printf("[commerce] 批量校验资源权限失败: %v", err)
			return states
		}
		for resourceID, state := range batchIndex {
			result, ok := results[resourceID]
			if !ok || !result.HasAccess {
				continue
			}
			state.HasAccess = true
			state.AccessSource = result.AccessType
			for _, resource := range resources {
				if resource != nil && resource.ResourceID == resourceID {
					state.URL = resource.URL
					state.ExtractCode = resource.ExtractCode
					break
				}
			}
		}
	}
	return states
}

func (s *Service) buildPreviewHTML(detail *model.ArticleDetailResponse, state *model.ArticleCommerceArticleState) string {
	if strings.TrimSpace(state.PreviewMode) == "custom" && strings.TrimSpace(state.PreviewText) != "" {
		return s.wrapParagraphs(state.PreviewText)
	}
	if len(detail.Summaries) > 0 {
		return s.wrapParagraphs(strings.Join(detail.Summaries, "\n\n"))
	}
	return ""
}

func (s *Service) wrapParagraphs(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "\n")
	builder := strings.Builder{}
	builder.WriteString(`<div class="article-commerce-preview">`)
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		builder.WriteString("<p>")
		builder.WriteString(html.EscapeString(piece))
		builder.WriteString("</p>")
	}
	builder.WriteString("</div>")
	return builder.String()
}

func (s *Service) sanitizeExtraConfig(extra *model.ArticleExtraConfig, articleCfg *model.ArticleCommerceArticleConfig, resourceCfgs []*model.ArticleCommerceResourceConfig, resourceStates []*model.ArticleCommerceResourceState) *model.ArticleExtraConfig {
	if extra == nil {
		return nil
	}
	cloned := *extra
	if articleCfg == nil && len(resourceCfgs) == 0 {
		return &cloned
	}
	safeCommerce := &model.ArticleCommerceConfig{}
	if articleCfg != nil {
		articleCopy := *articleCfg
		safeCommerce.Article = &articleCopy
	}
	if len(resourceCfgs) > 0 {
		accessMap := make(map[string]bool, len(resourceStates))
		for _, state := range resourceStates {
			if state != nil {
				accessMap[state.ID] = state.HasAccess
			}
		}
		safeResources := make([]*model.ArticleCommerceResourceConfig, 0, len(resourceCfgs))
		for _, resource := range resourceCfgs {
			if resource == nil {
				continue
			}
			copyItem := *resource
			if !accessMap[resource.ID] {
				copyItem.URL = ""
				copyItem.ExtractCode = ""
			}
			safeResources = append(safeResources, &copyItem)
		}
		safeCommerce.Resources = safeResources
	}
	cloned.Commerce = safeCommerce
	return &cloned
}

func (s *Service) normalizeArticleConfig(detail *model.ArticleDetailResponse) *model.ArticleCommerceArticleConfig {
	if detail == nil || detail.ExtraConfig == nil || detail.ExtraConfig.Commerce == nil || detail.ExtraConfig.Commerce.Article == nil {
		return nil
	}
	copyItem := *detail.ExtraConfig.Commerce.Article
	if copyItem.AccessType == "" {
		copyItem.AccessType = "free"
	}
	if copyItem.ResourceType == "" {
		copyItem.ResourceType = "article"
	}
	if copyItem.ResourceID == "" {
		copyItem.ResourceID = detail.ID
	}
	if copyItem.PreviewMode == "" {
		copyItem.PreviewMode = "summary"
	}
	return &copyItem
}

func (s *Service) normalizeArticleConfigFromArticle(article *model.Article) *model.ArticleCommerceArticleConfig {
	if article == nil || article.ExtraConfig == nil || article.ExtraConfig.Commerce == nil || article.ExtraConfig.Commerce.Article == nil {
		return nil
	}
	copyItem := *article.ExtraConfig.Commerce.Article
	if copyItem.AccessType == "" {
		copyItem.AccessType = "free"
	}
	if copyItem.ResourceType == "" {
		copyItem.ResourceType = "article"
	}
	if copyItem.ResourceID == "" {
		copyItem.ResourceID = article.ID
	}
	if copyItem.PreviewMode == "" {
		copyItem.PreviewMode = "summary"
	}
	return &copyItem
}

func (s *Service) normalizeResourceConfigs(detail *model.ArticleDetailResponse) []*model.ArticleCommerceResourceConfig {
	if detail == nil || detail.ExtraConfig == nil || detail.ExtraConfig.Commerce == nil || len(detail.ExtraConfig.Commerce.Resources) == 0 {
		return nil
	}
	return normalizeResourceConfigs(detail.ID, detail.ExtraConfig.Commerce.Resources)
}

func (s *Service) normalizeResourceConfigsFromArticle(article *model.Article) []*model.ArticleCommerceResourceConfig {
	if article == nil || article.ExtraConfig == nil || article.ExtraConfig.Commerce == nil || len(article.ExtraConfig.Commerce.Resources) == 0 {
		return nil
	}
	return normalizeResourceConfigs(article.ID, article.ExtraConfig.Commerce.Resources)
}

func normalizeResourceConfigs(articleID string, resources []*model.ArticleCommerceResourceConfig) []*model.ArticleCommerceResourceConfig {
	normalized := make([]*model.ArticleCommerceResourceConfig, 0, len(resources))
	for index, resource := range resources {
		if resource == nil {
			continue
		}
		copyItem := *resource
		if copyItem.ID == "" {
			copyItem.ID = fmt.Sprintf("resource-%d", index+1)
		}
		if copyItem.ResourceType == "" {
			copyItem.ResourceType = "article_resource"
		}
		if copyItem.ResourceID == "" {
			copyItem.ResourceID = fmt.Sprintf("%s:%s", articleID, copyItem.ID)
		}
		if copyItem.AccessType == "" {
			copyItem.AccessType = "free"
		}
		normalized = append(normalized, &copyItem)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].Sort < normalized[j].Sort
	})
	return normalized
}

type resourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type batchAccessItem struct {
	HasAccess  bool   `json:"has_access"`
	AccessType string `json:"access_type"`
	PurchaseID string `json:"purchase_id"`
}

type batchAccessResponse struct {
	Code int `json:"code"`
	Data struct {
		Results map[string]batchAccessItem `json:"results"`
	} `json:"data"`
}

type singleAccessResponse struct {
	HasAccess bool `json:"has_access"`
	IsMember  bool `json:"is_member"`
}

type remoteOAuthTokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type remoteUserMeResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	CurrentTier   string `json:"current_tier"`
	TierExpiresAt string `json:"tier_expires_at"`
	Created       string `json:"created"`
	Updated       string `json:"updated"`
	Name          string `json:"name"`
	TierInfo      struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	} `json:"tier_info"`
}

type remotePermissionVerifyResponse struct {
	Tier      string `json:"tier"`
	TierName  string `json:"tier_name"`
	IsActive  bool   `json:"is_active"`
	ExpiresAt string `json:"expires_at"`
}

func (s *Service) checkResourcesBatch(ctx context.Context, token string, resources []resourceRef) (map[string]batchAccessItem, error) {
	payload := map[string]interface{}{"resources": resources}
	var result batchAccessResponse
	if err := s.requestJSON(ctx, http.MethodPost, fmt.Sprintf("/site/%s/check-resources", s.siteID()), token, payload, map[string]string{"X-API-Key": s.siteAPIKey()}, &result); err != nil {
		return nil, err
	}
	return result.Data.Results, nil
}

func (s *Service) checkResourceSingle(ctx context.Context, resourceType string, resourceID string, guestToken string) (*singleAccessResponse, error) {
	query := fmt.Sprintf("/site/%s/check-resource?resource_type=%s&resource_id=%s", s.siteID(), resourceType, resourceID)
	if strings.TrimSpace(guestToken) != "" {
		query = fmt.Sprintf("%s&guest_token=%s", query, guestToken)
	}
	var result singleAccessResponse
	if err := s.requestJSON(ctx, http.MethodGet, query, "", nil, map[string]string{"X-API-Key": s.siteAPIKey()}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) buildMemberSession(ctx context.Context, tokenResult remoteOAuthTokenResult) (*MemberSessionResult, error) {
	bearerToken := normalizeBearerToken(tokenResult.AccessToken)
	dashboard, err := s.GetMemberDashboard(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	expiresIn := tokenResult.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return &MemberSessionResult{
		AccessToken:  tokenResult.AccessToken,
		RefreshToken: tokenResult.RefreshToken,
		TokenType:    firstNonEmpty(tokenResult.TokenType, "Bearer"),
		ExpiresIn:    expiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339),
		Dashboard:    *dashboard,
	}, nil
}

func (s *Service) requestJSON(ctx context.Context, method string, path string, bearerToken string, body interface{}, headers map[string]string, target interface{}) error {
	return s.requestJSONToBase(ctx, s.baseURL(), method, path, bearerToken, body, headers, target)
}

func (s *Service) requestJSONRoot(ctx context.Context, method string, path string, bearerToken string, body interface{}, headers map[string]string, target interface{}) error {
	return s.requestJSONToBase(ctx, s.rootURL(), method, path, bearerToken, body, headers, target)
}

func (s *Service) requestJSONToBase(ctx context.Context, base string, method string, path string, bearerToken string, body interface{}, headers map[string]string, target interface{}) error {
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("统一认证中心地址未配置")
	}
	requestCtx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(requestCtx, method, strings.TrimRight(base, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(bearerToken) != "" {
		request.Header.Set("Authorization", bearerToken)
	}
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			request.Header.Set(key, value)
		}
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(extractRemoteError(responseBody))
		if message == "" {
			message = fmt.Sprintf("统一认证中心返回状态码 %d", response.StatusCode)
		}
		return fmt.Errorf(message)
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("解析统一认证中心响应失败: %w", err)
	}
	return nil
}

func extractRemoteError(payload []byte) string {
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return string(payload)
	}
	for _, key := range []string{"error", "message", "msg"} {
		if value, ok := body[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return string(payload)
}

func normalizeBearerToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return trimmed
	}
	return "Bearer " + trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func boolValue(value *bool) bool {
	return value != nil && *value
}
