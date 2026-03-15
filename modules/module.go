package modules

import (
	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/setting"
)

type ModuleContext struct {
	SettingSvc setting.SettingService
	DB         *ent.Client
}

type Module interface {
	Name() string
	Version() string
	Description() string
	Init(ctx *ModuleContext) error
	OnArticlePublished(articleID string, articleURL string) error
	OnArticleUpdated(articleID string, articleURL string) error
}

type ModuleInfo struct {
	Name        string
	Version     string
	Description string
	Module      Module
}
