package member_handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalauth "github.com/anzhiyu-c/anheyu-app/internal/pkg/auth"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	"github.com/anzhiyu-c/anheyu-app/pkg/integration/dp7575"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type bindingRepositoryStub struct {
	binding commerce.MemberBindingDTO
	err     error
}

func (s *bindingRepositoryStub) FindByUserID(_ context.Context, userID int64) (commerce.MemberBindingDTO, error) {
	if s.err != nil {
		return commerce.MemberBindingDTO{}, s.err
	}
	binding := s.binding
	binding.UserID = userID
	return binding, nil
}

func (s *bindingRepositoryStub) Upsert(_ context.Context, dto commerce.MemberBindingDTO) error {
	_ = dto
	return nil
}

type memberClientStub struct {
	statusResp             dp7575.MemberStatusResponse
	statusErr              error
	statusReq              dp7575.MemberStatusRequest
	catalogResp            dp7575.MemberProductsCatalogResponse
	orderResp              dp7575.OrderCreateResponse
	paymentResp            dp7575.OrderPaymentDetailResponse
	redeemResp             dp7575.CardRedeemCreateResponse
	ensureResp             dp7575.UserMapEnsureResponse
	ensureErr              error
	ensureReq              dp7575.UserMapEnsureRequest
	orderReq               dp7575.OrderCreateRequest
	paymentReq             dp7575.OrderPaymentDetailRequest
	redeemReq              dp7575.CardRedeemCreateRequest
	orderListResp          dp7575.OrderListResponse
	orderDetailResp        dp7575.OrderDetailResponse
	orderStatusResp        dp7575.OrderStatusResponse
	adminOrderMappingsErr  error
	adminOrderMappingsResp dp7575.AdminOrderMappingListResponse
	adminCardsErr          error
	adminCardsResp         dp7575.AdminCardListResponse
}

type resourceRepositoryStub struct {
	resourceByID        commerce.ResourceRecordDTO
	resourceByHost      commerce.ResourceRecordDTO
	adminResources      []commerce.AdminResourceListItemDTO
	adminResourceDetail commerce.AdminResourceDetailDTO
	boundResourceID     string
	boundArticleID      string
	deletedResourceID   string
	resourceOrderCount  int
	articleHostOptions  []commerce.AdminArticleHostOptionDTO
	hasGrant            bool
	articleHostExists   bool
	err                 error
}

func (s *resourceRepositoryStub) FindResourceByID(_ context.Context, resourceID string) (commerce.ResourceRecordDTO, error) {
	if s.err != nil {
		return commerce.ResourceRecordDTO{}, s.err
	}
	resource := s.resourceByID
	if resource.ResourceID == "" {
		resource.ResourceID = resourceID
	}
	return resource, nil
}

func (s *resourceRepositoryStub) ListAdminResources(_ context.Context, _ commerce.AdminResourceListQueryDTO) (commerce.AdminResourceListDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceListDTO{}, s.err
	}
	return commerce.AdminResourceListDTO{List: s.adminResources, Total: len(s.adminResources), Page: 1, PageSize: len(s.adminResources)}, nil
}

func (s *resourceRepositoryStub) GetAdminResourceDetail(_ context.Context, resourceID string) (commerce.AdminResourceDetailDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceDetailDTO{}, s.err
	}
	resource := s.adminResourceDetail
	if resource.ResourceID == "" {
		resource.ResourceID = resourceID
	}
	return resource, nil
}

func (s *resourceRepositoryStub) CreateAdminResource(_ context.Context, input commerce.AdminResourceDetailDTO) (commerce.AdminResourceDetailDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceDetailDTO{}, s.err
	}
	if input.ResourceID == "" {
		input.ResourceID = "res_created"
	}
	return input, nil
}

func (s *resourceRepositoryStub) UpdateAdminResource(_ context.Context, resourceID string, input commerce.AdminResourceDetailDTO) (commerce.AdminResourceDetailDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceDetailDTO{}, s.err
	}
	input.ResourceID = resourceID
	return input, nil
}

func (s *resourceRepositoryStub) BindAdminResourceToArticle(_ context.Context, resourceID string, articleID string) (commerce.AdminResourceDetailDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceDetailDTO{}, s.err
	}
	s.boundResourceID = resourceID
	s.boundArticleID = articleID
	return commerce.AdminResourceDetailDTO{ResourceID: resourceID, HostType: "article", HostID: articleID}, nil
}

func (s *resourceRepositoryStub) DeleteAdminResource(_ context.Context, resourceID string) error {
	if s.err != nil {
		return s.err
	}
	s.deletedResourceID = resourceID
	return nil
}

func (s *resourceRepositoryStub) CountResourceOrders(_ context.Context, _ string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.resourceOrderCount, nil
}

func (s *resourceRepositoryStub) SearchArticleHosts(_ context.Context, _ string) ([]commerce.AdminArticleHostOptionDTO, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.articleHostOptions, nil
}

func (s *resourceRepositoryStub) FindResourceByArticleHost(_ context.Context, articleID string) (commerce.AdminResourceDetailDTO, error) {
	if s.err != nil {
		return commerce.AdminResourceDetailDTO{}, s.err
	}
	if s.adminResourceDetail.HostID != "" && s.adminResourceDetail.HostID != articleID {
		return commerce.AdminResourceDetailDTO{}, commerce.ErrResourceNotFound
	}
	return s.adminResourceDetail, nil
}

func (s *resourceRepositoryStub) FindResourceByHost(_ context.Context, hostType, hostID string) (commerce.ResourceRecordDTO, error) {
	if s.err != nil {
		return commerce.ResourceRecordDTO{}, s.err
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

func (s *resourceRepositoryStub) ArticleHostExists(_ context.Context, _ string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.articleHostExists, nil
}

func (s *resourceRepositoryStub) ResolveArticleIDByAbbrlink(_ context.Context, abbrlink string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.resourceByHost.HostID != "" {
		return s.resourceByHost.HostID, nil
	}
	return abbrlink, nil
}

func (s *resourceRepositoryStub) HasActiveGrant(_ context.Context, _ int64, _, _ string) (bool, error) {
	return s.hasGrant, nil
}

func (s *resourceRepositoryStub) HasGrantBySourceOrderNo(_ context.Context, _ int64, _ string) (bool, error) {
	return false, nil
}

func (s *resourceRepositoryStub) CreateGrant(_ context.Context, _ commerce.ResourceAccessGrantCreateDTO) error {
	return nil
}

type resourceOrderRepositoryStub struct {
	existing commerce.ResourceOrderRecordDTO
}

type memberZoneRepositoryStub struct {
	createErr error
}

func (s *memberZoneRepositoryStub) ListAdminMemberZones(_ context.Context, _ commerce.AdminMemberZoneListQueryDTO) (commerce.AdminMemberZoneListDTO, error) {
	return commerce.AdminMemberZoneListDTO{}, nil
}

func (s *memberZoneRepositoryStub) GetAdminMemberZoneDetail(_ context.Context, _ string) (commerce.AdminMemberZoneDetailDTO, error) {
	return commerce.AdminMemberZoneDetailDTO{}, s.createErr
}

func (s *memberZoneRepositoryStub) CreateAdminMemberZone(_ context.Context, input commerce.AdminMemberZoneDetailDTO) (commerce.AdminMemberZoneDetailDTO, error) {
	if s.createErr != nil {
		return commerce.AdminMemberZoneDetailDTO{}, s.createErr
	}
	return input, nil
}

func (s *memberZoneRepositoryStub) UpdateAdminMemberZone(_ context.Context, _ string, input commerce.AdminMemberZoneDetailDTO) (commerce.AdminMemberZoneDetailDTO, error) {
	if s.createErr != nil {
		return commerce.AdminMemberZoneDetailDTO{}, s.createErr
	}
	return input, nil
}

func (s *memberZoneRepositoryStub) DeleteAdminMemberZone(_ context.Context, _ string) error {
	return s.createErr
}

func (s *memberZoneRepositoryStub) FindAdminMemberZoneByArticle(_ context.Context, _ string) (commerce.AdminMemberZoneDetailDTO, error) {
	return commerce.AdminMemberZoneDetailDTO{}, s.createErr
}

func (s *memberZoneRepositoryStub) ListPublishedMemberZones(_ context.Context) ([]commerce.MemberZoneListItemDTO, error) {
	return nil, s.createErr
}

func (s *memberZoneRepositoryStub) GetPublishedMemberZoneMetaBySlug(_ context.Context, _ string) (commerce.MemberZoneMetaDTO, error) {
	return commerce.MemberZoneMetaDTO{}, s.createErr
}

func (s *memberZoneRepositoryStub) GetPublishedMemberZoneByArticle(_ context.Context, _ string) (commerce.MemberZoneMetaDTO, error) {
	return commerce.MemberZoneMetaDTO{}, s.createErr
}

func (s *memberZoneRepositoryStub) GetPublishedMemberZoneContentBySlug(_ context.Context, _ string) (commerce.MemberZoneContentDTO, error) {
	return commerce.MemberZoneContentDTO{}, s.createErr
}

func (s *resourceOrderRepositoryStub) Create(_ context.Context, input commerce.ResourceOrderCreateDTO) (commerce.ResourceOrderRecordDTO, error) {
	return commerce.ResourceOrderRecordDTO(input), nil
}

func (s *resourceOrderRepositoryStub) MarkPaid(_ context.Context, _, _ string, _ *time.Time) (bool, error) {
	return true, nil
}

func (s *resourceOrderRepositoryStub) UpdateExternalOrderNo(_ context.Context, _, _ string) error {
	return nil
}

func (s *resourceOrderRepositoryStub) FindLatestPendingByUserAndResource(_ context.Context, userID int64, resourceID string) (commerce.ResourceOrderRecordDTO, error) {
	if s.existing.BusinessOrderNo != "" && s.existing.UserID == userID && s.existing.ResourceID == resourceID && s.existing.Status == "pending" {
		return s.existing, nil
	}
	return commerce.ResourceOrderRecordDTO{}, commerce.ErrResourceOrderNotFound
}

func (s *resourceOrderRepositoryStub) FindByBusinessOrderNo(_ context.Context, businessOrderNo string) (commerce.ResourceOrderRecordDTO, error) {
	if s.existing.BusinessOrderNo == "" {
		s.existing.BusinessOrderNo = businessOrderNo
	}
	return s.existing, nil
}

func (s *resourceOrderRepositoryStub) FindByExternalOrderNo(_ context.Context, externalOrderNo string) (commerce.ResourceOrderRecordDTO, error) {
	if s.existing.ExternalOrderNo == "" {
		s.existing.ExternalOrderNo = externalOrderNo
	}
	return s.existing, nil
}

func (s *memberClientStub) MemberStatus(_ context.Context, req dp7575.MemberStatusRequest) (dp7575.MemberStatusResponse, error) {
	s.statusReq = req
	if s.statusErr != nil {
		return dp7575.MemberStatusResponse{}, s.statusErr
	}
	return s.statusResp, nil
}

func (s *memberClientStub) MemberProfile(context.Context, dp7575.MemberProfileRequest) (dp7575.MemberProfileResponse, error) {
	panic("unexpected call")
}

func (s *memberClientStub) MemberProductsCatalog(context.Context) (dp7575.MemberProductsCatalogResponse, error) {
	return s.catalogResp, nil
}

func (s *memberClientStub) CreateOrder(_ context.Context, req dp7575.OrderCreateRequest) (dp7575.OrderCreateResponse, error) {
	s.orderReq = req
	return s.orderResp, nil
}

func (s *memberClientStub) OrderPaymentDetail(_ context.Context, req dp7575.OrderPaymentDetailRequest) (dp7575.OrderPaymentDetailResponse, error) {
	s.paymentReq = req
	return s.paymentResp, nil
}

func (s *memberClientStub) CardRedeemPrecheck(context.Context, dp7575.CardRedeemPrecheckRequest) (dp7575.CardRedeemPrecheckResponse, error) {
	panic("unexpected call")
}

func (s *memberClientStub) CardRedeemCreate(_ context.Context, req dp7575.CardRedeemCreateRequest) (dp7575.CardRedeemCreateResponse, error) {
	s.redeemReq = req
	return s.redeemResp, nil
}

func (s *memberClientStub) ConfigComplete() bool { return true }

func (s *memberClientStub) HealthProbe(context.Context) (dp7575.HealthProbeResult, error) {
	panic("unexpected call")
}

func (s *memberClientStub) EnsureUserMapping(_ context.Context, req dp7575.UserMapEnsureRequest) (dp7575.UserMapEnsureResponse, error) {
	s.ensureReq = req
	if s.ensureErr != nil {
		return dp7575.UserMapEnsureResponse{}, s.ensureErr
	}
	return s.ensureResp, nil
}

func (s *memberClientStub) OrderList(_ context.Context, req dp7575.OrderListRequest) (dp7575.OrderListResponse, error) {
	return s.orderListResp, nil
}

func (s *memberClientStub) OrderDetail(_ context.Context, req dp7575.OrderDetailRequest) (dp7575.OrderDetailResponse, error) {
	return s.orderDetailResp, nil
}

func (s *memberClientStub) OrderStatus(_ context.Context, req dp7575.OrderStatusRequest) (dp7575.OrderStatusResponse, error) {
	return s.orderStatusResp, nil
}

func (s *memberClientStub) AdminOrderMappings(_ context.Context, _ dp7575.AdminOrderMappingListRequest) (dp7575.AdminOrderMappingListResponse, error) {
	if s.adminOrderMappingsErr != nil {
		return dp7575.AdminOrderMappingListResponse{}, s.adminOrderMappingsErr
	}
	return s.adminOrderMappingsResp, nil
}

func (s *memberClientStub) AdminCards(_ context.Context, _ dp7575.AdminCardListRequest) (dp7575.AdminCardListResponse, error) {
	if s.adminCardsErr != nil {
		return dp7575.AdminCardListResponse{}, s.adminCardsErr
	}
	return s.adminCardsResp, nil
}

func newMemberTestContext(t *testing.T, method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	require.NoError(t, idgen.InitSqidsEncoder())
	publicID, err := idgen.GeneratePublicID(1001, idgen.EntityTypeUser)
	require.NoError(t, err)
	c.Set(internalauth.ClaimsKey, &internalauth.CustomClaims{UserID: publicID})
	return c, recorder
}

func newOptionalMemberTestContext(t *testing.T, method, target, body string, loggedIn bool) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if loggedIn {
		require.NoError(t, idgen.InitSqidsEncoder())
		publicID, err := idgen.GeneratePublicID(1001, idgen.EntityTypeUser)
		require.NoError(t, err)
		c.Set(internalauth.ClaimsKey, &internalauth.CustomClaims{UserID: publicID})
	}
	return c, recorder
}

func TestHandler_CheckResourceAccess_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "资源ID、文章ID或文章短链至少提供一个")
}

func TestHandler_CheckResourceAccess_RejectsConflictingLocators(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"resource_id":"res_1","article_id":"art_1"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "只能提供一种资源定位参数")
}

func TestHandler_CheckResourceAccess_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{resourceByID: commerce.ResourceRecordDTO{ResourceID: "res_real_1", Title: "前端项目源码包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 29.9}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"resource_id":"res_real_1"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "resource_purchase")
	require.Contains(t, recorder.Body.String(), "res_real_1")
	require.Contains(t, recorder.Body.String(), "前端项目源码包")
}

func TestHandler_CheckResourceAccess_LoginRequiredForGuest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{resourceByID: commerce.ResourceRecordDTO{ResourceID: "res_real_1", Title: "前端项目源码包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 29.9}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"resource_id":"res_real_1"}`, false)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "login_required")
	require.Contains(t, recorder.Body.String(), "requires_login")
}

func TestHandler_CheckResourceAccess_ResolvesFromArticleID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{resourceByHost: commerce.ResourceRecordDTO{ResourceID: "res_article_1", HostType: "article", HostID: "art_public_1", Title: "文章配套资源", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 9.9}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"article_id":"art_public_1"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "res_article_1")
	require.Contains(t, recorder.Body.String(), "文章配套资源")
}

func TestHandler_CheckResourceAccess_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{err: commerce.ErrResourceNotFound}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"resource_id":"res_missing"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestHandler_CheckResourceAccess_ArticleWithoutResourceReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{err: commerce.ErrResourceNotFound}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"abbrlink":"Y4hH"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "resource_not_found")
}

func TestHandler_CheckResourceAccess_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "oerx", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{err: commerce.ErrResourceUnavailable}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newOptionalMemberTestContext(t, http.MethodPost, "/api/public/resource/access-check", `{"resource_id":"res_off"}`, true)
	handler.CheckResourceAccess(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestExtractOptionalUserID_PreservesExplicitExternalUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	require.NoError(t, idgen.InitSqidsEncoder())
	publicID, err := idgen.GeneratePublicID(1001, idgen.EntityTypeUser)
	require.NoError(t, err)
	c.Set(internalauth.ClaimsKey, &internalauth.CustomClaims{UserID: publicID, ExternalUserID: "dp-user-001"})

	actor := extractOptionalUserID(c)

	require.True(t, actor.LoggedIn)
	require.Equal(t, int64(1001), actor.UserID)
	require.Equal(t, "dp-user-001", actor.ExternalUserID)
}

func TestHandler_GetPurchaseCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{catalogResp: dp7575.MemberProductsCatalogResponse{Products: []dp7575.MemberProduct{{ProductID: "vip_1_0_pay", MemberLevel: 1, MemberLevelName: "普通会员", Title: "普通会员购买", Description: "12个月", Price: 99, OriginalPrice: 199, ActionType: "pay"}}}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/member/purchase/catalog", "")
	handler.GetPurchaseCatalog(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "获取会员购买目录成功")
	require.Contains(t, recorder.Body.String(), "vip_1_0_pay")
}

func TestHandler_CreatePurchaseOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderResp: dp7575.OrderCreateResponse{ZibOrderNum: "VIP20260418001", OrderStatus: "pending", PayType: "wechat", PayURL: "https://pay.example.com/wechat/VIP20260418001"}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/purchase/order", `{"product_id":"vip_1_0_pay","payment_method":"wechat"}`)
	handler.CreatePurchaseOrder(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "user_public_123", client.orderReq.ExternalUserID)
	require.Equal(t, "vip_1_0_pay", client.orderReq.ProductID)
	require.Contains(t, recorder.Body.String(), "创建会员订单成功")
	require.Contains(t, recorder.Body.String(), "VIP20260418001")
}

func TestHandler_CreateResourcePurchaseOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderResp: dp7575.OrderCreateResponse{ZibOrderNum: "RES20260419001", OrderStatus: "pending", PayType: "alipay", PayURL: "https://pay.example.com/alipay/RES20260419001", OrderPrice: 29.9}}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{resourceByID: commerce.ResourceRecordDTO{ResourceID: "res_real_1", HostType: "article", HostID: "art_public_1", Title: "前端项目源码包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 29.9}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order", `{"resource_id":"res_real_1"}`)
	handler.CreateResourcePurchaseOrder(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "创建资源订单成功")
	require.Contains(t, recorder.Body.String(), "res_real_1")
	require.Contains(t, recorder.Body.String(), "pay_url")
	require.Equal(t, "resource_purchase", client.orderReq.BusinessType)
}

func TestHandler_CreateResourcePurchaseOrder_UsesRequestedPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderResp: dp7575.OrderCreateResponse{ZibOrderNum: "RES20260419001", OrderStatus: "pending", PayType: "wechat", PayURL: "https://pay.example.com/wechat/RES20260419001", OrderPrice: 29.9}}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{resourceByID: commerce.ResourceRecordDTO{ResourceID: "res_real_1", HostType: "article", HostID: "art_public_1", Title: "前端项目源码包", ResourceType: "download_bundle", Status: "published", SaleEnabled: true, Price: 29.9}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order", `{"resource_id":"res_real_1","payment_method":"wechat"}`)
	handler.CreateResourcePurchaseOrder(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "wechat", client.orderReq.PaymentMethod)
}

func TestHandler_CreateResourcePurchaseOrder_RejectsConflictingLocators(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order", `{"resource_id":"res_1","abbrlink":"hello"}`)
	handler.CreateResourcePurchaseOrder(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "只能提供一种资源定位参数")
}

func TestHandler_CreateResourcePurchaseOrder_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{err: commerce.ErrResourceNotFound}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order", `{"resource_id":"res_missing"}`)
	handler.CreateResourcePurchaseOrder(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestHandler_CreateResourcePurchaseOrder_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	service := commerce.NewService(repo, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{err: commerce.ErrResourceUnavailable}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order", `{"resource_id":"res_off"}`)
	handler.CreateResourcePurchaseOrder(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestHandler_GetResourcePurchaseOrderStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderStatusResp: dp7575.OrderStatusResponse{ZibOrderNum: "RES20260419001", OrderStatus: "paid", OrderStatusLabel: "已支付", OrderPrice: 29.9, PayPrice: 29.9, PayType: "alipay", CreatedAt: "2026-04-19 12:00:00", PaidAt: "2026-04-19 12:01:00"}}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{existing: commerce.ResourceOrderRecordDTO{BusinessOrderNo: "YGZ_RES_002", UserID: 1001, ResourceID: "res_real_1", ExternalOrderNo: "RES20260419001", Status: "pending"}})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order/status", `{"business_order_no":"YGZ_RES_002"}`)
	handler.GetResourcePurchaseOrderStatus(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "获取资源订单状态成功")
	require.Contains(t, recorder.Body.String(), "status")
}

func TestHandler_GetResourcePurchaseOrderStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderStatusResp: dp7575.OrderStatusResponse{ZibOrderNum: "RES20260419001", OrderStatus: "paid"}}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{existing: commerce.ResourceOrderRecordDTO{BusinessOrderNo: "YGZ_RES_002", UserID: 2002, ResourceID: "res_real_1", ExternalOrderNo: "RES20260419001", Status: "pending"}})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order/status", `{"business_order_no":"YGZ_RES_002"}`)
	handler.GetResourcePurchaseOrderStatus(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestHandler_GetResourcePurchasePaymentDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{paymentResp: dp7575.OrderPaymentDetailResponse{
		OrderNum:   "ZGZ_EXT_002",
		Amount:     29.9,
		PayType:    "alipay",
		PayChannel: "alipay",
		PayDetail:  map[string]any{"url_qrcode": "data:image/png;base64,ALIPAYQR"},
	}}
	service := commerce.NewService(repo, client)
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{existing: commerce.ResourceOrderRecordDTO{BusinessOrderNo: "YGZ_RES_002", UserID: 1001, ResourceID: "res_real_1", ExternalOrderNo: "ZGZ_EXT_002", Status: "pending"}})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/resource/order/payment-detail", `{"business_order_no":"YGZ_RES_002"}`)
	handler.GetResourcePurchasePaymentDetail(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "user_public_123", client.paymentReq.ExternalUserID)
	require.Equal(t, "ZGZ_EXT_002", client.paymentReq.ZibOrderNum)
	require.Contains(t, recorder.Body.String(), "获取资源支付详情成功")
	require.Contains(t, recorder.Body.String(), "url_qrcode")
}

func TestHandler_ListAdminResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{adminResources: []commerce.AdminResourceListItemDTO{{
		ResourceID: "res_1",
		Title:      "源码包",
		Status:     "published",
		HostType:   "article",
		HostID:     "art_1",
		HostTitle:  "示例文章",
	}}}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/resources?page=1&pageSize=10", "")
	handler.ListAdminResources(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "源码包")
}

func TestHandler_BindAdminResourceToArticle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	repo := &resourceRepositoryStub{}
	service.SetResourceRepositories(repo, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/resources/res_2/bind-article", `{"article_id":"art_1"}`)
	c.Params = gin.Params{{Key: "id", Value: "res_2"}}
	handler.BindAdminResourceToArticle(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "res_2", repo.boundResourceID)
	require.Equal(t, "art_1", repo.boundArticleID)
	require.Contains(t, recorder.Body.String(), "绑定资源成功")
}

func TestHandler_DeleteAdminResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	repo := &resourceRepositoryStub{adminResourceDetail: commerce.AdminResourceDetailDTO{ResourceID: "res_3"}}
	service.SetResourceRepositories(repo, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodDelete, "/api/resources/res_3", "")
	c.Params = gin.Params{{Key: "id", Value: "res_3"}}
	handler.DeleteAdminResource(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "res_3", repo.deletedResourceID)
	require.Contains(t, recorder.Body.String(), "删除资源成功")
}

func TestHandler_ListAdminOrderMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{adminOrderMappingsResp: dp7575.AdminOrderMappingListResponse{
		Summary:    dp7575.AdminOrderMappingSummary{Total: 2, SiteIDZeroCount: 1, LatestCreatedAt: "2026-04-19 15:58:07"},
		List:       []dp7575.AdminOrderMappingItem{{ZibOrderNum: "2604191558069513850", ExternalUserID: "oerx", WpUserID: 7, ProductType: "custom_amount", CreatedAt: "2026-04-19 15:58:07", StoredSiteID: "yangguangzhan", ResolvedSiteID: "yangguangzhan", SnapshotSiteID: "yangguangzhan", ContextSource: "snapshot"}},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 20, Total: 2},
	}})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/admin/jiguangku/orders?page=1&page_size=20", "")
	handler.ListAdminOrderMappings(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "获取订单映射列表成功")
	require.Contains(t, recorder.Body.String(), "2604191558069513850")
}

func TestHandler_ListAdminCards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{adminCardsResp: dp7575.AdminCardListResponse{
		List:       []dp7575.AdminCardItem{{CardCode: "23282285775808818733", CardPassword: "Uu5WKelcw4Z86SfdonM9Kz1l09y6FBPrR7v", CardType: "vip_exchange", Status: "used", CreatedAt: "2026-04-18 22:27:00", UpdatedAt: "2026-04-18 22:33:43"}},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 20, Total: 1},
	}})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/admin/jiguangku/cards?status=used&page=1&page_size=20", "")
	handler.ListAdminCards(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "获取卡密列表成功")
	require.Contains(t, recorder.Body.String(), "23282285775808818733")
}

func TestHandler_ListAdminOrderMappings_WhenUpstreamMissing_ReturnsReadableMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{adminOrderMappingsErr: errors.New("dp7575 admin order mappings failed: status=404 body=404 Not Found")})
	handler := NewHandler(service)

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/admin/jiguangku/orders?page=1&page_size=20", "")
	handler.ListAdminOrderMappings(c)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "极光库管理员订单接口不可用")
	require.Contains(t, recorder.Body.String(), "admin/orders")
}

func TestHandler_CreateAdminResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	service.SetResourceRepositories(&resourceRepositoryStub{}, &resourceOrderRepositoryStub{})
	handler := NewHandler(service)

	body := `{"title":"资源 A","status":"published","sale_enabled":true,"price":0,"host_type":"article","host_id":"art_1","items":[{"title":"百度网盘","item_type":"link","url":"https://pan.example.com/x","status":"active"}]}`
	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/resources", body)
	handler.CreateAdminResource(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "资源 A")
}

func TestHandler_GetPurchasePaymentDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{paymentResp: dp7575.OrderPaymentDetailResponse{
		OrderNum:   "VIP20260418001",
		Amount:     199,
		PayType:    "wechat",
		PayChannel: "wechat",
		PayDetail:  map[string]any{"url_qrcode": "data:image/png;base64,WECHATQR"},
	}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/purchase/payment-detail", `{"order_no":"VIP20260418001"}`)
	handler.GetPurchasePaymentDetail(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "user_public_123", client.paymentReq.ExternalUserID)
	require.Equal(t, "VIP20260418001", client.paymentReq.ZibOrderNum)
	require.Contains(t, recorder.Body.String(), "获取会员支付详情成功")
	require.Contains(t, recorder.Body.String(), "url_qrcode")
}

func TestHandler_RedeemCard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{redeemResp: dp7575.CardRedeemCreateResponse{RedeemStatus: "success", TargetType: "member", TargetSummary: "会员兑换：等级 1", OrderNum: "26041820010001", EffectSummary: "兑换成功"}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/purchase/card/redeem", `{"card_code":"VIP-CARD-001","card_password":"SECURE-001"}`)
	handler.RedeemCard(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "user_public_123", client.redeemReq.ExternalUserID)
	require.Equal(t, "VIP-CARD-001", client.redeemReq.CardCode)
	require.Contains(t, recorder.Body.String(), "兑换会员卡密成功")
	require.Contains(t, recorder.Body.String(), "26041820010001")
}

func TestHandler_GetMemberOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderListResp: dp7575.OrderListResponse{
		List: []dp7575.OrderListItem{
			{OrderNum: "VIP20260418001", BusinessType: "member", ProductType: "vip", Status: "paid", Amount: 99, PayType: "wechat", CreateTime: "2026-04-18 09:32:00"},
		},
		Pagination: dp7575.OrderListPagination{Page: 1, PageSize: 10, Total: 1},
	}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodGet, "/api/member/orders", "")
	handler.GetMemberOrders(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "VIP20260418001")
	require.Contains(t, recorder.Body.String(), "member")
	require.Contains(t, recorder.Body.String(), "business_type")
	require.Contains(t, recorder.Body.String(), "status")
}

func TestHandler_GetMemberOrderDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderDetailResp: dp7575.OrderDetailResponse{
		OrderNum: "VIP20260418001", BusinessType: "member", ProductType: "vip", ProductID: "vip_1_0_pay",
		Status: "paid", Amount: 99, PayType: "wechat", PayTime: "2026-04-18 09:35:00", CreateTime: "2026-04-18 09:32:00",
		Snapshot: map[string]any{"product": map[string]any{"title": "年度会员"}},
	}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/orders/detail", `{"order_no":"VIP20260418001"}`)
	handler.GetMemberOrderDetail(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "VIP20260418001")
	require.Contains(t, recorder.Body.String(), "member")
	require.Contains(t, recorder.Body.String(), "business_type")
}

func TestHandler_GetMemberOrderStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepositoryStub{binding: commerce.MemberBindingDTO{ExternalUserID: "user_public_123", SiteID: "yangguangzhan", Status: "active"}}
	client := &memberClientStub{orderStatusResp: dp7575.OrderStatusResponse{
		BusinessType: "member", ZibOrderNum: "VIP20260418001", ProductType: "vip",
		OrderStatus: "paid", OrderStatusLabel: "已支付", OrderPrice: 99, PayPrice: 99,
		PayType: "wechat", CreatedAt: "2026-04-18 09:32:00", PaidAt: "2026-04-18 09:35:00",
	}}
	handler := NewHandler(commerce.NewService(repo, client))

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member/orders/status", `{"order_no":"VIP20260418001"}`)
	handler.GetMemberOrderStatus(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "VIP20260418001")
	require.Contains(t, recorder.Body.String(), "member")
	require.Contains(t, recorder.Body.String(), "已支付")
}

func TestHandler_CreateAdminMemberZone_BadRequestOnInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	svc.SetMemberZoneRepository(&memberZoneRepositoryStub{})
	handler := NewHandler(svc)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member-zone", `{"title":"","slug":"vip-zone","content_md":"hello","content_html":"<p>hello</p>","status":"published","access_level":"member"}`)
	handler.CreateAdminMemberZone(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "创建会员专区失败")
}

func TestHandler_CreateAdminMemberZone_ConflictOnDuplicateSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := commerce.NewService(&bindingRepositoryStub{}, &memberClientStub{})
	svc.SetMemberZoneRepository(&memberZoneRepositoryStub{createErr: commerce.ErrMemberZoneConflict})
	handler := NewHandler(svc)

	c, recorder := newMemberTestContext(t, http.MethodPost, "/api/member-zone", `{"title":"会员内容","slug":"vip-zone","content_md":"hello","content_html":"<p>hello</p>","status":"published","access_level":"member"}`)
	handler.CreateAdminMemberZone(c)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "创建会员专区失败")
}
