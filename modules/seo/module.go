package seo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/modules"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
	"golang.org/x/oauth2/google"
)

type SeoModule struct {
	settingSvc setting.SettingService
	httpClient *http.Client
}

func NewSeoModule() *SeoModule {
	return &SeoModule{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *SeoModule) Name() string { return "seo" }

func (m *SeoModule) Version() string { return "1.0.0" }

func (m *SeoModule) Description() string {
	return "SEO 自动推送模块，支持百度/Bing/Google 搜索引擎"
}

func (m *SeoModule) Init(ctx *modules.ModuleContext) error {
	m.settingSvc = ctx.SettingSvc
	log.Printf("[seo] SEO 推送模块初始化完成")
	return nil
}

func (m *SeoModule) OnArticlePublished(articleID string, articleURL string) error {
	if !m.isAutoSubmitEnabled() {
		return nil
	}

	log.Printf("[seo] 文章发布，准备推送: %s", articleURL)

	var errors []error

	if m.isBaiduEnabled() {
		if err := m.pushToBaidu(articleURL); err != nil {
			errors = append(errors, fmt.Errorf("百度推送失败: %w", err))
		}
	}

	if m.isBingEnabled() {
		if err := m.pushToBing(articleURL); err != nil {
			errors = append(errors, fmt.Errorf("Bing 推送失败: %w", err))
		}
	}

	if m.isGoogleEnabled() {
		if err := m.pushToGoogle(articleURL); err != nil {
			errors = append(errors, fmt.Errorf("Google 推送失败: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("推送过程中发生错误: %v", errors)
	}

	return nil
}

func (m *SeoModule) OnArticleUpdated(articleID string, articleURL string) error {
	return m.OnArticlePublished(articleID, articleURL)
}

func (m *SeoModule) isAutoSubmitEnabled() bool { return m.settingSvc.Get("seo.auto_submit") == "true" }

func (m *SeoModule) getRetryConfig() (times int, interval time.Duration) {
	times = 3
	interval = time.Second

	if v := m.settingSvc.Get("seo.retry_times"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			times = n
		}
	}

	if v := m.settingSvc.Get("seo.retry_interval"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			interval = time.Duration(n) * time.Millisecond
		}
	}

	return
}

func (m *SeoModule) isBaiduEnabled() bool  { return m.settingSvc.Get("seo.baidu.enable") == "true" }
func (m *SeoModule) isBingEnabled() bool   { return m.settingSvc.Get("seo.bing.enable") == "true" }
func (m *SeoModule) isGoogleEnabled() bool { return m.settingSvc.Get("seo.google.enable") == "true" }

func (m *SeoModule) pushWithRetry(fn func() error) error {
	times, interval := m.getRetryConfig()

	var lastErr error
	for i := 0; i < times; i++ {
		if err := fn(); err != nil {
			lastErr = err
			log.Printf("[seo] 推送失败 (第 %d/%d 次): %v", i+1, times, err)
			if i < times-1 {
				time.Sleep(interval)
			}
			continue
		}
		return nil
	}

	return lastErr
}

func (m *SeoModule) pushToBaidu(url string) error {
	site := m.settingSvc.Get("seo.baidu.site")
	token := m.settingSvc.Get("seo.baidu.token")

	if site == "" || token == "" {
		return fmt.Errorf("百度推送配置不完整")
	}

	apiURL := fmt.Sprintf("http://data.zz.baidu.com/urls?site=%s&token=%s", site, token)

	return m.pushWithRetry(func() error {
		body := strings.NewReader(url)
		req, err := http.NewRequest("POST", apiURL, body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("百度推送返回状态码 %d: %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Success int    `json:"success"`
			Remain  int    `json:"remain"`
			Error   int    `json:"error,omitempty"`
			Message string `json:"message,omitempty"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("解析百度响应失败: %w", err)
		}
		if result.Error != 0 {
			return fmt.Errorf("百度推送错误: %s", result.Message)
		}

		log.Printf("[seo] 百度推送成功: 成功 %d 条, 剩余配额 %d", result.Success, result.Remain)
		return nil
	})
}

func (m *SeoModule) pushToBing(url string) error {
	apiKey := m.settingSvc.Get("seo.bing.api_key")
	if apiKey == "" {
		return fmt.Errorf("Bing API Key 未配置")
	}

	return m.pushWithRetry(func() error {
		apiURL := fmt.Sprintf("https://www.bing.com/indexnow?url=%s&key=%s", url, apiKey)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[seo] Bing IndexNow 推送成功: %s", url)
			return nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bing 推送返回状态码 %d: %s", resp.StatusCode, string(respBody))
	})
}

func (m *SeoModule) pushToGoogle(url string) error {
	credentialJSON := m.settingSvc.Get("seo.google.credential")
	if credentialJSON == "" {
		return fmt.Errorf("Google 凭证未配置")
	}

	return m.pushWithRetry(func() error {
		accessToken, err := m.getGoogleAccessToken(credentialJSON)
		if err != nil {
			return fmt.Errorf("获取 Google Access Token 失败: %w", err)
		}

		payload := map[string]string{"url": url, "type": "URL_UPDATED"}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", "https://indexing.googleapis.com/v3/urlNotifications:publish", bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[seo] Google Indexing API 推送成功: %s", url)
			return nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Google 推送返回状态码 %d: %s", resp.StatusCode, string(respBody))
	})
}

func (m *SeoModule) getGoogleAccessToken(credentialJSON string) (string, error) {
	config, err := google.JWTConfigFromJSON([]byte(credentialJSON), "https://www.googleapis.com/auth/indexing")
	if err != nil {
		return "", fmt.Errorf("解析凭证 JSON 失败: %w", err)
	}

	token, err := config.TokenSource(context.Background()).Token()
	if err != nil {
		return "", fmt.Errorf("获取 Token 失败: %w", err)
	}

	return token.AccessToken, nil
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
