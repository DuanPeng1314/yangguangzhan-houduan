package modules

import (
	"log"
	"sync"
)

var registry = &ModuleRegistry{modules: make(map[string]Module)}

type ModuleRegistry struct {
	modules map[string]Module
	mu      sync.RWMutex
}

func GetRegistry() *ModuleRegistry {
	return registry
}

func (r *ModuleRegistry) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[m.Name()]; exists {
		log.Printf("[modules] 模块 %s 已注册，跳过", m.Name())
		return
	}

	r.modules[m.Name()] = m
	log.Printf("[modules] 模块 %s (v%s) 注册成功", m.Name(), m.Version())
}

func (r *ModuleRegistry) OnArticlePublished(articleID string, articleURL string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.modules {
		go func(mod Module) {
			if err := mod.OnArticlePublished(articleID, articleURL); err != nil {
				log.Printf("[modules] 模块 %s 处理文章发布事件失败: %v", mod.Name(), err)
			}
		}(m)
	}
}

func (r *ModuleRegistry) OnArticleUpdated(articleID string, articleURL string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.modules {
		go func(mod Module) {
			if err := mod.OnArticleUpdated(articleID, articleURL); err != nil {
				log.Printf("[modules] 模块 %s 处理文章更新事件失败: %v", mod.Name(), err)
			}
		}(m)
	}
}
