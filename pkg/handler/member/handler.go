package member_handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/anzhiyu-c/anheyu-app/internal/pkg/auth"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	"github.com/anzhiyu-c/anheyu-app/pkg/response"
	"github.com/gin-gonic/gin"
)

func resourceErrorStatus(err error) int {
	switch {
	case errors.Is(err, commerce.ErrResourceLocatorRequired):
		return http.StatusBadRequest
	case errors.Is(err, commerce.ErrInvalidResourceLocator):
		return http.StatusBadRequest
	case errors.Is(err, commerce.ErrResourceNotFound):
		return http.StatusNotFound
	case errors.Is(err, commerce.ErrResourceOrderNotFound):
		return http.StatusNotFound
	case errors.Is(err, commerce.ErrResourcePurchaseNotRequired):
		return http.StatusForbidden
	case errors.Is(err, commerce.ErrResourceUnavailable):
		return http.StatusForbidden
	case errors.Is(err, commerce.ErrResourceBoundToArticle):
		return http.StatusConflict
	case errors.Is(err, commerce.ErrResourceHasOrders):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func memberZoneErrorStatus(err error) int {
	switch {
	case errors.Is(err, commerce.ErrMemberZoneInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, commerce.ErrMemberZoneNotFound):
		return http.StatusNotFound
	case errors.Is(err, commerce.ErrMemberZoneUnavailable):
		return http.StatusUnauthorized
	case errors.Is(err, commerce.ErrMemberZoneAccessDenied):
		return http.StatusForbidden
	case errors.Is(err, commerce.ErrMemberZoneConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func validateResourceLocatorRequest(req commerce.ResourceAccessCheckRequestDTO) error {
	if err := commerce.ValidateResourceLocatorForAPI(req); err != nil {
		return err
	}
	return nil
}

type Handler struct {
	service *commerce.Service
}

func NewHandler(service *commerce.Service) *Handler {
	return &Handler{service: service}
}

func extractUserID(c *gin.Context) (int64, bool) {
	claimsValue, exists := c.Get(auth.ClaimsKey)
	if !exists {
		response.Fail(c, http.StatusUnauthorized, "未登录或无法获取当前用户信息")
		return 0, false
	}

	claims, ok := claimsValue.(*auth.CustomClaims)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "用户信息格式不正确")
		return 0, false
	}

	userID, entityType, err := idgen.DecodePublicID(claims.UserID)
	if err != nil || entityType != idgen.EntityTypeUser {
		response.Fail(c, http.StatusUnauthorized, "用户ID无效")
		return 0, false
	}

	return int64(userID), true
}

func extractOptionalUserID(c *gin.Context) *commerce.ResourceAccessCheckActorDTO {
	claimsValue, exists := c.Get(auth.ClaimsKey)
	if !exists {
		return &commerce.ResourceAccessCheckActorDTO{LoggedIn: false}
	}

	claims, ok := claimsValue.(*auth.CustomClaims)
	if !ok {
		return &commerce.ResourceAccessCheckActorDTO{LoggedIn: false}
	}

	userID, entityType, err := idgen.DecodePublicID(claims.UserID)
	if err != nil || entityType != idgen.EntityTypeUser {
		return &commerce.ResourceAccessCheckActorDTO{LoggedIn: false}
	}

	return &commerce.ResourceAccessCheckActorDTO{
		UserID:         int64(userID),
		ExternalUserID: firstNonEmpty(claims.ExternalUserID, claims.UserID),
		LoggedIn:       true,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func (h *Handler) GetStatus(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	status, err := h.service.GetMemberStatus(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取会员状态失败")
		return
	}

	response.Success(c, status, "获取会员状态成功")
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	profile, err := h.service.GetMemberProfile(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取会员资料失败")
		return
	}

	response.Success(c, profile, "获取会员资料成功")
}

func (h *Handler) CheckResourceAccess(c *gin.Context) {
	var req commerce.ResourceAccessCheckRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if err := validateResourceLocatorRequest(req); err != nil {
		if errors.Is(err, commerce.ErrResourceLocatorRequired) {
			response.Fail(c, http.StatusBadRequest, "资源ID、文章ID或文章短链至少提供一个")
			return
		}
		if errors.Is(err, commerce.ErrInvalidResourceLocator) {
			response.Fail(c, http.StatusBadRequest, "资源ID、文章ID或文章短链只能提供一种资源定位参数")
			return
		}
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CheckResourceAccess(c.Request.Context(), extractOptionalUserID(c), req)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "资源访问判定失败")
		return
	}

	response.Success(c, result, "资源访问判定成功")
}

func (h *Handler) GetPurchaseCatalog(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	result, err := h.service.GetMemberPurchaseCatalog(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取会员购买目录失败")
		return
	}

	response.Success(c, result, "获取会员购买目录成功")
}

func (h *Handler) ListAdminResources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	result, err := h.service.ListAdminResources(c.Request.Context(), commerce.AdminResourceListQueryDTO{
		Page:     page,
		PageSize: pageSize,
		Query:    c.Query("query"),
		Status:   c.Query("status"),
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取资源列表失败")
		return
	}

	response.Success(c, result, "获取资源列表成功")
}

func (h *Handler) ListAdminMemberZones(c *gin.Context) {
	result, err := h.service.ListAdminMemberZones(c.Request.Context(), commerce.AdminMemberZoneListQueryDTO{
		Page:     parsePositiveInt(c.Query("page"), 1),
		PageSize: parsePositiveInt(c.Query("pageSize"), 10),
		Query:    c.Query("query"),
		Status:   c.Query("status"),
	})
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取会员专区列表失败")
		return
	}

	response.Success(c, result, "获取会员专区列表成功")
}

func (h *Handler) ListAdminOrderMappings(c *gin.Context) {
	result, err := h.service.ListAdminOrderMappings(c.Request.Context(), commerce.AdminOrderMappingListQueryDTO{
		SiteID:      c.Query("site_id"),
		ZibOrderNum: c.Query("zib_order_num"),
		Page:        parsePositiveInt(c.Query("page"), 1),
		PageSize:    parsePositiveInt(c.Query("page_size"), 20),
	})
	if err != nil {
		message := "获取订单映射列表失败"
		if strings.Contains(err.Error(), "dp7575 admin order mappings failed: status=404") {
			message = "极光库管理员订单接口不可用：远端 /admin/orders 尚未部署或不可达"
		}
		response.Fail(c, http.StatusInternalServerError, message)
		return
	}

	response.Success(c, result, "获取订单映射列表成功")
}

func (h *Handler) ListAdminCards(c *gin.Context) {
	result, err := h.service.ListAdminCards(c.Request.Context(), commerce.AdminCardListQueryDTO{
		CardType: c.Query("card_type"),
		Status:   c.Query("status"),
		Page:     parsePositiveInt(c.Query("page"), 1),
		PageSize: parsePositiveInt(c.Query("page_size"), 20),
	})
	if err != nil {
		message := "获取卡密列表失败"
		if strings.Contains(err.Error(), "dp7575 admin cards failed: status=404") {
			message = "极光库管理员卡密接口不可用：远端 /admin/cards 尚未部署或不可达"
		}
		response.Fail(c, http.StatusInternalServerError, message)
		return
	}

	response.Success(c, result, "获取卡密列表成功")
}

func (h *Handler) GetAdminResourceDetail(c *gin.Context) {
	result, err := h.service.GetAdminResourceDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "获取资源详情失败")
		return
	}

	response.Success(c, result, "获取资源详情成功")
}

func (h *Handler) GetAdminMemberZoneDetail(c *gin.Context) {
	result, err := h.service.GetAdminMemberZoneDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取会员专区详情失败")
		return
	}

	response.Success(c, result, "获取会员专区详情成功")
}

func (h *Handler) CreateAdminResource(c *gin.Context) {
	var req commerce.AdminResourceDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CreateAdminResource(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "创建资源失败")
		return
	}

	response.Success(c, result, "创建资源成功")
}

func (h *Handler) CreateAdminMemberZone(c *gin.Context) {
	var req commerce.AdminMemberZoneDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CreateAdminMemberZone(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "创建会员专区失败")
		return
	}

	response.Success(c, result, "创建会员专区成功")
}

func (h *Handler) UpdateAdminResource(c *gin.Context) {
	var req commerce.AdminResourceDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.UpdateAdminResource(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "更新资源失败")
		return
	}

	response.Success(c, result, "更新资源成功")
}

func (h *Handler) UpdateAdminMemberZone(c *gin.Context) {
	var req commerce.AdminMemberZoneDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.UpdateAdminMemberZone(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "更新会员专区失败")
		return
	}

	response.Success(c, result, "更新会员专区成功")
}

func (h *Handler) BindAdminResourceToArticle(c *gin.Context) {
	var req commerce.AdminResourceBindArticleDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.BindAdminResourceToArticle(c.Request.Context(), c.Param("id"), req.ArticleID)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "绑定资源失败")
		return
	}

	response.Success(c, result, "绑定资源成功")
}

func (h *Handler) DeleteAdminResource(c *gin.Context) {
	err := h.service.DeleteAdminResource(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "删除资源失败")
		return
	}

	response.Success(c, nil, "删除资源成功")
}

func (h *Handler) DeleteAdminMemberZone(c *gin.Context) {
	if err := h.service.DeleteAdminMemberZone(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "删除会员专区失败")
		return
	}

	response.Success(c, nil, "删除会员专区成功")
}

func (h *Handler) SearchAdminArticleHosts(c *gin.Context) {
	result, err := h.service.SearchAdminArticleHosts(c.Request.Context(), c.Query("query"))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "搜索文章失败")
		return
	}

	response.Success(c, result, "搜索文章成功")
}

func (h *Handler) GetAdminMemberZoneByArticle(c *gin.Context) {
	result, err := h.service.GetAdminMemberZoneByArticle(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取文章绑定会员专区失败")
		return
	}

	response.Success(c, result, "获取文章绑定会员专区成功")
}

func (h *Handler) GetAdminResourceByArticle(c *gin.Context) {
	result, err := h.service.GetAdminResourceByArticle(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "获取文章绑定资源失败")
		return
	}

	response.Success(c, result, "获取文章绑定资源成功")
}

func (h *Handler) ListPublishedMemberZones(c *gin.Context) {
	result, err := h.service.ListPublishedMemberZones(c.Request.Context())
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取会员专区列表失败")
		return
	}

	response.Success(c, result, "获取会员专区列表成功")
}

func (h *Handler) GetMemberZoneMeta(c *gin.Context) {
	result, err := h.service.GetPublishedMemberZoneMetaBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取会员专区信息失败")
		return
	}

	response.Success(c, result, "获取会员专区信息成功")
}

func (h *Handler) GetPublicMemberZoneByArticle(c *gin.Context) {
	result, err := h.service.GetPublishedMemberZoneByArticle(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取文章关联会员专区失败")
		return
	}

	response.Success(c, result, "获取文章关联会员专区成功")
}

func (h *Handler) CheckMemberZoneAccess(c *gin.Context) {
	var req commerce.MemberZoneAccessCheckRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Slug) == "" {
		response.Fail(c, http.StatusBadRequest, "slug 不能为空")
		return
	}

	result, err := h.service.CheckMemberZoneAccess(c.Request.Context(), extractOptionalUserID(c), req.Slug)
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "会员专区访问判定失败")
		return
	}

	response.Success(c, result, "会员专区访问判定成功")
}

func (h *Handler) GetMemberZoneContent(c *gin.Context) {
	var req commerce.MemberZoneAccessCheckRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Slug) == "" {
		response.Fail(c, http.StatusBadRequest, "slug 不能为空")
		return
	}

	result, err := h.service.GetMemberZoneContentForActor(c.Request.Context(), extractOptionalUserID(c), req.Slug)
	if err != nil {
		response.Fail(c, memberZoneErrorStatus(err), "获取会员专区正文失败")
		return
	}

	response.Success(c, result, "获取会员专区正文成功")
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req commerce.MemberOrderCreateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CreateMemberOrder(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "创建会员订单失败")
		return
	}

	response.Success(c, result, "创建会员订单成功")
}

func (h *Handler) CreateResourcePurchaseOrder(c *gin.Context) {
	actor := extractOptionalUserID(c)
	if actor == nil || !actor.LoggedIn {
		response.Fail(c, http.StatusUnauthorized, "未登录或无法获取当前用户信息")
		return
	}

	var req commerce.ResourceAccessCheckRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if err := validateResourceLocatorRequest(req); err != nil {
		if errors.Is(err, commerce.ErrResourceLocatorRequired) {
			response.Fail(c, http.StatusBadRequest, "资源ID、文章ID或文章短链至少提供一个")
			return
		}
		if errors.Is(err, commerce.ErrInvalidResourceLocator) {
			response.Fail(c, http.StatusBadRequest, "资源ID、文章ID或文章短链只能提供一种资源定位参数")
			return
		}
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CreateResourcePurchaseOrderForActor(c.Request.Context(), actor, req)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "创建资源订单失败")
		return
	}

	response.Success(c, result, "创建资源订单成功")
}

func (h *Handler) GetResourcePurchaseOrderStatus(c *gin.Context) {
	actor := extractOptionalUserID(c)
	if actor == nil || !actor.LoggedIn {
		response.Fail(c, http.StatusUnauthorized, "未登录或无法获取当前用户信息")
		return
	}

	var req commerce.ResourcePurchaseOrderStatusRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if req.BusinessOrderNo == "" {
		response.Fail(c, http.StatusBadRequest, "业务订单号不能为空")
		return
	}

	result, err := h.service.GetResourcePurchaseOrderStatusForActor(c.Request.Context(), actor, req.BusinessOrderNo)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "获取资源订单状态失败")
		return
	}

	response.Success(c, result, "获取资源订单状态成功")
}

func (h *Handler) GetResourcePurchasePaymentDetail(c *gin.Context) {
	actor := extractOptionalUserID(c)
	if actor == nil || !actor.LoggedIn {
		response.Fail(c, http.StatusUnauthorized, "未登录或无法获取当前用户信息")
		return
	}

	var req commerce.ResourceOrderPaymentDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if req.BusinessOrderNo == "" {
		response.Fail(c, http.StatusBadRequest, "业务订单号不能为空")
		return
	}

	result, err := h.service.GetResourcePaymentDetailForActor(c.Request.Context(), actor, req)
	if err != nil {
		response.Fail(c, resourceErrorStatus(err), "获取资源支付详情失败")
		return
	}

	response.Success(c, result, "获取资源支付详情成功")
}

func (h *Handler) RedeemCard(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req commerce.MemberCardRedeemDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.RedeemMemberCard(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "兑换会员卡密失败")
		return
	}

	response.Success(c, result, "兑换会员卡密成功")
}

func (h *Handler) GetPurchasePaymentDetail(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req commerce.MemberOrderPaymentDetailDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.GetMemberPaymentDetail(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取会员支付详情失败")
		return
	}

	response.Success(c, result, "获取会员支付详情成功")
}

func (h *Handler) GetHealth(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	result, err := h.service.GetHealthCheck(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "健康检查执行失败")
		return
	}

	response.Success(c, result, "健康检查完成")
}

func (h *Handler) GetMemberOrders(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	result, err := h.service.GetMemberOrders(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取我的订单失败")
		return
	}

	response.Success(c, result, "获取我的订单成功")
}

func (h *Handler) GetMemberOrderDetail(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req commerce.MemberOrderDetailRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.OrderNo == "" {
		response.Fail(c, http.StatusBadRequest, "订单号不能为空")
		return
	}

	result, err := h.service.GetMemberOrderDetail(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取订单详情失败")
		return
	}

	response.Success(c, result, "获取订单详情成功")
}

func (h *Handler) GetMemberOrderStatus(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req commerce.MemberOrderStatusRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.OrderNo == "" {
		response.Fail(c, http.StatusBadRequest, "订单号不能为空")
		return
	}

	result, err := h.service.GetMemberOrderStatus(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取订单状态失败")
		return
	}

	response.Success(c, result, "获取订单状态成功")
}
