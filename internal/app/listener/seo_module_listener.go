package listener

import (
	"log"

	"github.com/anzhiyu-c/anheyu-app/internal/pkg/event"
	"github.com/anzhiyu-c/anheyu-app/modules"
	"github.com/anzhiyu-c/anheyu-app/modules/seo"
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

	return &SeoModuleListener{
		registry:   registry,
		settingSvc: settingSvc,
	}
}

func (l *SeoModuleListener) RegisterHandlers(bus *event.EventBus) {
	bus.Subscribe(event.ArticlePublished, l.onArticlePublished)
	bus.Subscribe(event.ArticleUpdated, l.onArticleUpdated)

	log.Println("[SeoModuleListener] SEO 模块事件监听器已注册")
}

func (l *SeoModuleListener) onArticlePublished(payload interface{}) {
	if p, ok := payload.(*event.ArticlePayload); ok {
		l.registry.OnArticlePublished(p.ID, p.URL)
	}
}

func (l *SeoModuleListener) onArticleUpdated(payload interface{}) {
	if p, ok := payload.(*event.ArticlePayload); ok {
		l.registry.OnArticleUpdated(p.ID, p.URL)
	}
}
