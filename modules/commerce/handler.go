package commerce

import (
	"net/http"
	"strings"

	"github.com/anzhiyu-c/anheyu-app/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createPurchaseRequest struct {
	ArticleID     string `json:"article_id"`
	ResourceID    string `json:"resource_id"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
}

type buildAuthorizeURLRequest struct {
	RedirectURI string `json:"redirect_uri" form:"redirect_uri"`
	State       string `json:"state" form:"state"`
}

type exchangeMemberTokenRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

type refreshMemberTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type redeemMemberKeyRequest struct {
	Key string `json:"key"`
}

func (h *Handler) CreateUserPurchase(c *gin.Context) {
	var req createPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	target, err := h.service.ResolvePurchaseTarget(c.Request.Context(), req.ArticleID, req.ResourceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.CreateUserPurchase(c.Request.Context(), c.GetHeader("Authorization"), target, normalizePaymentMethod(req.PaymentMethod))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "下单成功")
}

func (h *Handler) CreateGuestPurchase(c *gin.Context) {
	var req createPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	target, err := h.service.ResolvePurchaseTarget(c.Request.Context(), req.ArticleID, req.ResourceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.CreateGuestPurchase(c.Request.Context(), req.Email, target, normalizePaymentMethod(req.PaymentMethod))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "下单成功")
}

func (h *Handler) GetGuestOrderStatus(c *gin.Context) {
	result, err := h.service.GetGuestOrderStatus(c.Request.Context(), c.Param("orderNo"), c.Query("guest_token"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "获取成功")
}

func (h *Handler) GetMemberAuthorizeURL(c *gin.Context) {
	var req buildAuthorizeURLRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	result, err := h.service.BuildMemberAuthorizeURL(req.RedirectURI, req.State)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, result, "获取成功")
}

func (h *Handler) ExchangeMemberToken(c *gin.Context) {
	var req exchangeMemberTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	result, err := h.service.ExchangeMemberToken(c.Request.Context(), req.Code, req.RedirectURI)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "登录成功")
}

func (h *Handler) RefreshMemberToken(c *gin.Context) {
	var req refreshMemberTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	result, err := h.service.RefreshMemberToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "刷新成功")
}

func (h *Handler) GetMemberDashboard(c *gin.Context) {
	result, err := h.service.GetMemberDashboard(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "获取成功")
}

func (h *Handler) GetMemberTiers(c *gin.Context) {
	result, err := h.service.GetMemberTiers(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "获取成功")
}

func (h *Handler) RedeemMemberKey(c *gin.Context) {
	var req redeemMemberKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数无效")
		return
	}
	result, err := h.service.RedeemMemberKey(c.Request.Context(), c.GetHeader("Authorization"), req.Key)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, result, "兑换成功")
}

func normalizePaymentMethod(value string) string {
	method := strings.TrimSpace(value)
	if method == "" {
		return "alipay"
	}
	return method
}
