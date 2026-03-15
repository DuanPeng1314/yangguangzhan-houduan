package payment

import (
	"context"
	"fmt"

	"github.com/anzhiyu-c/anheyu-app/modules"
)

// PaymentModule 负责文章付费内容块同步。
type PaymentModule struct {
	service Service
}

// NewPaymentModule 创建付费模块。
func NewPaymentModule() *PaymentModule {
	return &PaymentModule{}
}

// Name 返回模块名称。
func (m *PaymentModule) Name() string { return "payment" }

// Version 返回模块版本。
func (m *PaymentModule) Version() string { return "1.0.0" }

// Description 返回模块描述。
func (m *PaymentModule) Description() string {
	return "统一付费内容块模块，支持同步、鉴权与购买回调"
}

// Init 初始化模块。
func (m *PaymentModule) Init(ctx *modules.ModuleContext) error {
	if ctx == nil || ctx.DB == nil {
		return fmt.Errorf("付费模块初始化失败：数据库客户端为空")
	}
	m.service = NewService(ctx.DB)
	return nil
}

// OnArticlePublished 处理文章发布事件。
func (m *PaymentModule) OnArticlePublished(articleID string, articleURL string) error {
	return m.syncArticlePayments(articleID)
}

// OnArticleUpdated 处理文章更新事件。
func (m *PaymentModule) OnArticleUpdated(articleID string, articleURL string) error {
	return m.syncArticlePayments(articleID)
}

func (m *PaymentModule) syncArticlePayments(articleID string) error {
	if m.service == nil {
		return fmt.Errorf("付费模块尚未初始化")
	}
	return m.service.ParseAndSyncPaymentsByPublicID(context.Background(), articleID)
}
