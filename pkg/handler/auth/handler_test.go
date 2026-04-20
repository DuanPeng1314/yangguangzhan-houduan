package auth_handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalauth "github.com/anzhiyu-c/anheyu-app/internal/pkg/auth"
	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/model"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	serviceauth "github.com/anzhiyu-c/anheyu-app/pkg/service/auth"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/captcha"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type authServiceStub struct {
	user *model.User
	err  error
	activateErr error
	activatedID uint
	activatedSign string
	getUserByID uint
}

func (s *authServiceStub) Login(context.Context, string, string) (*model.User, error) {
	return s.user, s.err
}

func (s *authServiceStub) Register(context.Context, string, string, string) (bool, error) {
	panic("unexpected call")
}

func (s *authServiceStub) ActivateUser(_ context.Context, userID uint, sign string) error {
	s.activatedID = userID
	s.activatedSign = sign
	return s.activateErr
}

func (s *authServiceStub) RequestPasswordReset(context.Context, string) error {
	panic("unexpected call")
}

func (s *authServiceStub) PerformPasswordReset(context.Context, uint, string, string) error {
	panic("unexpected call")
}

func (s *authServiceStub) CheckEmailExists(context.Context, string) (bool, error) {
	panic("unexpected call")
}

func (s *authServiceStub) GetUserByID(_ context.Context, userID uint) (*model.User, error) {
	s.getUserByID = userID
	return s.user, s.err
}

var _ serviceauth.AuthService = (*authServiceStub)(nil)

type tokenServiceStub struct {
	generateExternalUserID string
	generateCalls int
	refreshAccessToken string
	refreshExpires int64
	refreshErr error
	refreshCalls int
}

func (s *tokenServiceStub) GenerateSessionTokens(_ context.Context, _ *model.User, externalUserID string) (string, string, int64, error) {
	s.generateExternalUserID = externalUserID
	s.generateCalls++
	return "access-token", "refresh-token", 1234567890, nil
}

func (s *tokenServiceStub) RefreshAccessToken(context.Context, string) (string, int64, error) {
	s.refreshCalls++
	return s.refreshAccessToken, s.refreshExpires, s.refreshErr
}

func (s *tokenServiceStub) GenerateSignedToken(string, time.Duration) (string, error) {
	panic("unexpected call")
}

func (s *tokenServiceStub) VerifySignedToken(string, string) error {
	panic("unexpected call")
}

func (s *tokenServiceStub) ParseAccessToken(context.Context, string) (*internalauth.CustomClaims, error) {
	panic("unexpected call")
}

type settingServiceStub struct{}

func (s *settingServiceStub) LoadAllSettings(context.Context) error { panic("unexpected call") }
func (s *settingServiceStub) Get(key string) string {
	if key == constant.KeyJWTSecret.String() {
		return "test-secret"
	}
	return "https://gravatar.example.com/"
}
func (s *settingServiceStub) GetBool(string) bool                   { panic("unexpected call") }
func (s *settingServiceStub) GetByKeys([]string) map[string]interface{} {
	panic("unexpected call")
}
func (s *settingServiceStub) GetSiteConfig() map[string]interface{} { panic("unexpected call") }
func (s *settingServiceStub) GetConfigVersion() int64               { panic("unexpected call") }
func (s *settingServiceStub) UpdateSettings(context.Context, map[string]string) error {
	panic("unexpected call")
}
func (s *settingServiceStub) RegisterPublicSettings([]string) { panic("unexpected call") }
func (s *settingServiceStub) IsPublicSetting(string) bool     { panic("unexpected call") }

var _ setting.SettingService = (*settingServiceStub)(nil)

type captchaServiceStub struct{}

func (s *captchaServiceStub) GetProvider() captcha.CaptchaProvider { return captcha.ProviderNone }
func (s *captchaServiceStub) GetConfig() captcha.CaptchaConfig {
	return captcha.CaptchaConfig{Provider: captcha.ProviderNone}
}
func (s *captchaServiceStub) GenerateImageCaptcha(context.Context) (*captcha.ImageCaptchaResponse, error) {
	panic("unexpected call")
}
func (s *captchaServiceStub) Verify(context.Context, captcha.CaptchaParams, string) error { return nil }
func (s *captchaServiceStub) IsEnabled() bool                                             { return false }

var _ captcha.CaptchaService = (*captchaServiceStub)(nil)

type memberAutoBinderStub struct {
	calls        int
	userID       int64
	publicUserID string
	externalUserID string
	err          error
}

func (s *memberAutoBinderStub) AutoBindAfterLogin(_ context.Context, userID int64, publicUserID string) (string, error) {
	s.calls++
	s.userID = userID
	s.publicUserID = publicUserID
	return s.externalUserID, s.err
}

func newLoginUser() *model.User {
	return &model.User{
		ID:          1,
		Username:    "yangguang",
		Nickname:    "阳光",
		Email:       "112961548@qq.com",
		Avatar:      "avatar/test",
		UserGroupID: 2,
		UserGroup:   model.UserGroup{ID: 2, Name: "默认组", Description: "默认组"},
		Status:      model.UserStatusActive,
	}
}

func TestAuthHandler_Login_TriggersMemberAutoBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"112961548@qq.com","password":"secret123"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	binder := &memberAutoBinderStub{}
	handler := NewAuthHandler(&authServiceStub{user: newLoginUser()}, &tokenServiceStub{}, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.Login(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, binder.calls)
	require.Equal(t, int64(newLoginUser().ID), binder.userID)
	pid, _ := idgen.GeneratePublicID(newLoginUser().ID, idgen.EntityTypeUser)
	require.Equal(t, pid, binder.publicUserID)
	require.Contains(t, recorder.Body.String(), "登录成功")
}

func TestAuthHandler_Login_PassesResolvedExternalUserIDToTokenService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"112961548@qq.com","password":"secret123"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	binder := &memberAutoBinderStub{externalUserID: "oerx"}
	tokenSvc := &tokenServiceStub{}
	handler := NewAuthHandler(&authServiceStub{user: newLoginUser()}, tokenSvc, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.Login(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "oerx", tokenSvc.generateExternalUserID)
}

func TestAuthHandler_Login_FailsWhenMemberAutoBindingFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"112961548@qq.com","password":"secret123"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	binder := &memberAutoBinderStub{err: errors.New("sync failed")}
	handler := NewAuthHandler(&authServiceStub{user: newLoginUser()}, &tokenServiceStub{}, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.Login(c)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, 1, binder.calls)
	require.Contains(t, recorder.Body.String(), "同步会员映射失败")
}

func TestAuthHandler_ActivateUser_PassesResolvedExternalUserIDToTokenService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	publicUserID, err := idgen.GeneratePublicID(newLoginUser().ID, idgen.EntityTypeUser)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/activate", strings.NewReader(`{"id":"`+publicUserID+`","sign":"activate-sign"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	authSvc := &authServiceStub{user: newLoginUser()}
	binder := &memberAutoBinderStub{externalUserID: "oerx"}
	tokenSvc := &tokenServiceStub{}
	handler := NewAuthHandler(authSvc, tokenSvc, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.ActivateUser(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, uint(newLoginUser().ID), authSvc.activatedID)
	require.Equal(t, "activate-sign", authSvc.activatedSign)
	require.Equal(t, uint(newLoginUser().ID), authSvc.getUserByID)
	require.Equal(t, 1, binder.calls)
	require.Equal(t, int64(newLoginUser().ID), binder.userID)
	require.Equal(t, publicUserID, binder.publicUserID)
	require.Equal(t, "oerx", tokenSvc.generateExternalUserID)
}

func TestAuthHandler_RefreshToken_ReissuesTokensWithResolvedExternalUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	refreshToken, err := internalauth.GenerateRefreshToken(newLoginUser().ID, "user_public_123", []byte("test-secret"))
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh-token", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer "+refreshToken)

	authSvc := &authServiceStub{user: newLoginUser()}
	binder := &memberAutoBinderStub{externalUserID: "oerx"}
	tokenSvc := &tokenServiceStub{}
	handler := NewAuthHandler(authSvc, tokenSvc, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.RefreshToken(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, uint(newLoginUser().ID), authSvc.getUserByID)
	require.Equal(t, 1, binder.calls)
	publicUserID, genErr := idgen.GeneratePublicID(newLoginUser().ID, idgen.EntityTypeUser)
	require.NoError(t, genErr)
	require.Equal(t, publicUserID, binder.publicUserID)
	require.Equal(t, "oerx", tokenSvc.generateExternalUserID)
	require.Equal(t, 1, tokenSvc.generateCalls)
}

func TestAuthHandler_RefreshToken_DoesNotFallbackToLegacyRefreshWhenIdentityResolutionFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, idgen.InitSqidsEncoder())

	refreshToken, err := internalauth.GenerateRefreshToken(newLoginUser().ID, "user_public_123", []byte("test-secret"))
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh-token", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer "+refreshToken)

	authSvc := &authServiceStub{user: newLoginUser()}
	binder := &memberAutoBinderStub{err: errors.New("sync failed")}
	tokenSvc := &tokenServiceStub{refreshAccessToken: "legacy-access", refreshExpires: 123}
	handler := NewAuthHandler(authSvc, tokenSvc, &settingServiceStub{}, &captchaServiceStub{}, binder)

	handler.RefreshToken(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "sync failed")
	require.Equal(t, 0, tokenSvc.refreshCalls)
	require.Equal(t, 0, tokenSvc.generateCalls)
}
