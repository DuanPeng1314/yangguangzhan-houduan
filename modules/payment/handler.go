package payment

import (
	"net/http"

	"github.com/anzhiyu-c/anheyu-app/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler 表示付费模块 HTTP 处理器。
type Handler struct {
	service Service
}

// NewHandler 创建付费处理器。
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CheckAccess 检查付费内容访问权限。
func (h *Handler) CheckAccess(c *gin.Context) {
	var req CheckAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.CheckAccess(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "检查访问权限失败: "+err.Error())
		return
	}

	response.Success(c, result, "检查访问权限成功")
}

// Callback 处理支付回调。
func (h *Handler) Callback(c *gin.Context) {
	var req PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.HandleCallback(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "处理支付回调失败: "+err.Error())
		return
	}

	response.Success(c, result, "支付回调处理成功")
}
