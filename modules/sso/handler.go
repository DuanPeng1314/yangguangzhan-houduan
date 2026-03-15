package sso

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/anzhiyu-c/anheyu-app/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	module *Module
}

func NewHandler(module *Module) *Handler {
	return &Handler{module: module}
}

func (h *Handler) Callback(c *gin.Context) {
	if h.module == nil {
		response.Fail(c, http.StatusInternalServerError, "SSO 模块未初始化")
		return
	}
	if !h.module.Enabled() {
		response.Fail(c, http.StatusForbidden, "SSO 功能未启用")
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		response.Fail(c, http.StatusBadRequest, "缺少 OAuth 授权码")
		return
	}

	redirectURI := buildCallbackURL(c)
	tokenResp, err := h.module.ExchangeCodeForToken(c.Request.Context(), code, redirectURI)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, fmt.Sprintf("获取访问令牌失败: %v", err))
		return
	}

	userInfo, err := h.module.GetUserInfo(c.Request.Context(), tokenResp.AccessToken)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, fmt.Sprintf("获取用户信息失败: %v", err))
		return
	}

	response.Success(c, gin.H{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"token_type":    tokenResp.TokenType,
		"expires_in":    tokenResp.ExpiresIn,
		"scope":         tokenResp.Scope,
		"user_info":     userInfo,
	}, "SSO 登录成功")
}

func buildCallbackURL(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.FullPath())
}
