package sso

import (
	"net/http"
	"strings"

	"github.com/anzhiyu-c/anheyu-app/pkg/response"
	"github.com/gin-gonic/gin"
)

type Middleware struct {
	module *Module
}

func NewMiddleware(module *Module) *Middleware {
	return &Middleware{module: module}
}

func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.module == nil {
			response.Fail(c, http.StatusInternalServerError, "SSO 模块未初始化")
			c.Abort()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Fail(c, http.StatusUnauthorized, "请求未携带 Bearer Token")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Fail(c, http.StatusUnauthorized, "Authorization 头格式不正确")
			c.Abort()
			return
		}

		accessToken := strings.TrimSpace(parts[1])
		userInfo, err := m.module.VerifyToken(c.Request.Context(), accessToken)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Token 验证失败")
			c.Abort()
			return
		}

		c.Set(ContextUserKey, userInfo)
		c.Set(ContextAccessTokenKey, accessToken)
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (*UserInfo, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}

	userInfo, ok := value.(*UserInfo)
	return userInfo, ok
}
