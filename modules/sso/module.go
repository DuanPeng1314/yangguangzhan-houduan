package sso

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/modules"
	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
)

const (
	ContextUserKey        = "sso_user"
	ContextAccessTokenKey = "sso_access_token"

	defaultAuthCenterURL = "https://account.dp7575.com"
	defaultSiteID        = "yangguangzhan"
)

type Module struct {
	settingSvc setting.SettingService
	httpClient *http.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

type UserInfo struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Tier     string `json:"tier"`
	IsActive bool   `json:"is_active"`
}

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

func NewModule() *Module {
	return &Module{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *Module) Name() string {
	return "sso"
}

func (m *Module) Version() string {
	return "1.0.0"
}

func (m *Module) Description() string {
	return "SSO 统一登录模块，支持 OAuth2 授权码模式"
}

func (m *Module) Init(ctx *modules.ModuleContext) error {
	m.settingSvc = ctx.SettingSvc
	return nil
}

func (m *Module) OnArticlePublished(articleID string, articleURL string) error {
	return nil
}

func (m *Module) OnArticleUpdated(articleID string, articleURL string) error {
	return nil
}

func (m *Module) Enabled() bool {
	if m.settingSvc == nil {
		return false
	}
	return m.settingSvc.GetBool(constant.KeySsoEnabled.String())
}

func (m *Module) AuthCenterURL() string {
	if m.settingSvc == nil {
		return defaultAuthCenterURL
	}
	if value := strings.TrimRight(m.settingSvc.Get(constant.KeySsoAuthCenterURL.String()), "/"); value != "" {
		return value
	}
	return defaultAuthCenterURL
}

func (m *Module) SiteID() string {
	if m.settingSvc == nil {
		return defaultSiteID
	}
	if value := strings.TrimSpace(m.settingSvc.Get(constant.KeySsoSiteID.String())); value != "" {
		return value
	}
	return defaultSiteID
}

func (m *Module) SiteSecret() string {
	if m.settingSvc == nil {
		return ""
	}
	return strings.TrimSpace(m.settingSvc.Get(constant.KeySsoSiteSecret.String()))
}

func (m *Module) ExchangeCodeForToken(ctx context.Context, code string, redirectURI string) (*TokenResponse, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("授权码不能为空")
	}
	if m.SiteSecret() == "" {
		return nil, fmt.Errorf("SSO 站点密钥未配置")
	}

	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     m.SiteID(),
		"client_secret": m.SiteSecret(),
		"redirect_uri":  redirectURI,
	}

	var tokenResp TokenResponse
	if err := m.doJSONRequest(ctx, http.MethodPost, m.AuthCenterURL()+"/oauth/token", payload, nil, &tokenResp); err != nil {
		return nil, err
	}

	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, fmt.Errorf("认证中心未返回 access_token")
	}

	return &tokenResp, nil
}

func (m *Module) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("访问令牌不能为空")
	}

	var userInfo UserInfo
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	if err := m.doJSONRequest(ctx, http.MethodGet, m.AuthCenterURL()+"/oauth/userinfo", nil, headers, &userInfo); err != nil {
		return nil, err
	}

	if strings.TrimSpace(userInfo.UserID) == "" {
		return nil, fmt.Errorf("认证中心未返回用户ID")
	}

	return &userInfo, nil
}

func (m *Module) VerifyToken(ctx context.Context, accessToken string) (*UserInfo, error) {
	return m.GetUserInfo(ctx, accessToken)
}

func (m *Module) doJSONRequest(ctx context.Context, method string, targetURL string, payload interface{}, headers map[string]string, out interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化请求参数失败: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求认证中心失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取认证中心响应失败: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return parseRemoteError(resp.StatusCode, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("解析认证中心响应失败: %w", err)
	}

	return nil
}

func parseRemoteError(statusCode int, respBody []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(respBody, &errResp); err == nil {
		message := strings.TrimSpace(errResp.ErrorDescription)
		if message == "" {
			message = strings.TrimSpace(errResp.Message)
		}
		if message == "" {
			message = strings.TrimSpace(errResp.Error)
		}
		if message != "" {
			return fmt.Errorf("认证中心返回错误(%d): %s", statusCode, message)
		}
	}

	message := strings.TrimSpace(string(respBody))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("认证中心返回错误(%d): %s", statusCode, message)
}
