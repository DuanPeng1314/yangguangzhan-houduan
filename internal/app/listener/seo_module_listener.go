package listener

import (
	"log"
	"net/url"
	"strings"

	"github.com/anzhiyu-c/anheyu-app/internal/pkg/event"
	"github.com/anzhiyu-c/anheyu-app/modules"
	"github.com/anzhiyu-c/anheyu-app/modules/seo"
	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
)

type SeoModuleListener struct {
	registry   *modules.ModuleRegistry
	settingSvc setting.SettingService
}

func NewSeoModuleListener(settingSvc setting.SettingService) *SeoModuleListener {
	registry := modules.GetRegistry()
	seoModule := seo.NewSeoModule()
	registry.Register(seoModule)

	if err := seoModule.Init(&modules.ModuleContext{SettingSvc: settingSvc}); err != nil {
		log.Printf("[SeoModuleListener] SEO 模块初始化失败: %v", err)
	}

	return &SeoModuleListener{registry: registry, settingSvc: settingSvc}
}

func (l *SeoModuleListener) RegisterHandlers(bus *event.EventBus) {
	bus.Subscribe(event.ArticlePublished, l.onArticlePublished)
	bus.Subscribe(event.ArticleUpdated, l.onArticleUpdated)
	log.Println("[SeoModuleListener] SEO 模块事件监听器已注册")
}

func (l *SeoModuleListener) onArticlePublished(payload interface{}) {
	if p, ok := payload.(*event.ArticlePayload); ok {
		if articleURL, articleID := l.resolveArticleURL(p); articleURL != "" {
			l.registry.OnArticlePublished(articleID, articleURL)
		}
	}
}

func (l *SeoModuleListener) onArticleUpdated(payload interface{}) {
	if p, ok := payload.(*event.ArticlePayload); ok {
		if articleURL, articleID := l.resolveArticleURL(p); articleURL != "" {
			l.registry.OnArticleUpdated(articleID, articleURL)
		}
	}
}

func (l *SeoModuleListener) resolveArticleURL(payload *event.ArticlePayload) (string, string) {
	pathID := strings.TrimSpace(payload.Slug)
	if pathID == "" {
		pathID = strings.TrimSpace(payload.PublicID)
	}
	if pathID == "" {
		return "", ""
	}

	siteURL := strings.TrimRight(strings.TrimSpace(l.settingSvc.Get(constant.KeySiteURL.String())), "/")
	if siteURL == "" {
		log.Printf("[SeoModuleListener] SITE_URL 未配置，跳过 SEO 推送: slug=%s publicID=%s", payload.Slug, payload.PublicID)
		return "", pathID
	}

	escapedID := url.PathEscape(pathID)
	return siteURL + "/posts/" + escapedID, pathID
}
