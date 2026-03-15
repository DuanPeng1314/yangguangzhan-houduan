package payment

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/ent/article"
	"github.com/anzhiyu-c/anheyu-app/ent/articlepayment"
	"github.com/anzhiyu-c/anheyu-app/ent/articlepurchase"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
)

var paidContentBlockRegexp = regexp.MustCompile(`(?is)<div\b[^>]*class\s*=\s*['"][^'"]*paid-content-editor-preview[^'"]*['"][^>]*>`)
var htmlAttributeRegexp = regexp.MustCompile(`([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*(['"])(.*?)\2`)

// Service 定义付费模块服务。
type Service interface {
	ParseAndSyncPayments(ctx context.Context, articleID uint, markdown string) error
	ParseAndSyncPaymentsByPublicID(ctx context.Context, articleID string) error
	CheckAccess(ctx context.Context, req CheckAccessRequest) (*CheckAccessResult, error)
	HandleCallback(ctx context.Context, req PaymentCallbackRequest) (*PaymentCallbackResult, error)
}

// MembershipStatus 表示会员状态。
type MembershipStatus struct {
	Tier     string `json:"tier"`
	IsActive bool   `json:"is_active"`
}

// CheckAccessRequest 表示访问权限检查请求。
type CheckAccessRequest struct {
	UserID     string            `json:"user_id"`
	ArticleID  uint              `json:"article_id"`
	BlockID    string            `json:"block_id"`
	Membership *MembershipStatus `json:"membership"`
}

// CheckAccessResult 表示访问权限检查结果。
type CheckAccessResult struct {
	HasAccess    bool                 `json:"has_access"`
	AccessType   string               `json:"access_type"`
	NeedPurchase bool                 `json:"need_purchase"`
	Reason       string               `json:"reason,omitempty"`
	Payment      *PaymentBlockSummary `json:"payment,omitempty"`
}

// PaymentBlockSummary 表示付费块摘要。
type PaymentBlockSummary struct {
	ArticleID             uint   `json:"article_id"`
	BlockID               string `json:"block_id"`
	Title                 string `json:"title"`
	Price                 int    `json:"price"`
	OriginalPrice         *int   `json:"original_price,omitempty"`
	Currency              string `json:"currency"`
	ContentLength         int    `json:"content_length"`
	ExcludeFromMembership bool   `json:"exclude_from_membership"`
}

// PaymentCallbackRequest 表示支付回调请求。
type PaymentCallbackRequest struct {
	UserID      string     `json:"user_id"`
	ArticleID   uint       `json:"article_id"`
	BlockID     string     `json:"block_id"`
	Price       int        `json:"price"`
	OrderNo     string     `json:"order_no"`
	Status      string     `json:"status"`
	PurchasedAt *time.Time `json:"purchased_at"`
}

// PaymentCallbackResult 表示支付回调结果。
type PaymentCallbackResult struct {
	Recorded    bool      `json:"recorded"`
	PurchasedAt time.Time `json:"purchased_at"`
}

type paymentBlock struct {
	BlockID               string
	Title                 string
	Price                 int
	OriginalPrice         *int
	Currency              string
	ContentLength         int
	ExcludeFromMembership bool
}

type paymentService struct {
	db *ent.Client
}

// NewService 创建付费服务。
func NewService(db *ent.Client) Service {
	return &paymentService{db: db}
}

// ParseAndSyncPayments 解析文章 Markdown 中的付费块并同步到数据库。
func (s *paymentService) ParseAndSyncPayments(ctx context.Context, articleID uint, markdown string) error {
	blocks, err := parsePaymentBlocks(markdown)
	if err != nil {
		return err
	}

	existing, err := s.db.ArticlePayment.Query().Where(articlepayment.ArticleIDEQ(articleID)).All(ctx)
	if err != nil {
		return fmt.Errorf("查询现有付费内容块失败: %w", err)
	}

	blockMap := make(map[string]paymentBlock, len(blocks))
	for _, block := range blocks {
		blockMap[block.BlockID] = block
	}

	for _, existingBlock := range existing {
		block, ok := blockMap[existingBlock.BlockID]
		if !ok {
			if err := s.db.ArticlePayment.DeleteOneID(existingBlock.ID).Exec(ctx); err != nil {
				return fmt.Errorf("删除失效付费内容块失败: %w", err)
			}
			continue
		}

		update := s.db.ArticlePayment.UpdateOneID(existingBlock.ID).
			SetTitle(block.Title).
			SetPrice(block.Price).
			SetCurrency(block.Currency).
			SetContentLength(block.ContentLength).
			SetExcludeFromMembership(block.ExcludeFromMembership)
		if block.OriginalPrice != nil {
			update.SetOriginalPrice(*block.OriginalPrice)
		} else {
			update.ClearOriginalPrice()
		}
		if err := update.Exec(ctx); err != nil {
			return fmt.Errorf("更新付费内容块失败: %w", err)
		}
		delete(blockMap, existingBlock.BlockID)
	}

	for _, block := range blockMap {
		create := s.db.ArticlePayment.Create().
			SetArticleID(articleID).
			SetBlockID(block.BlockID).
			SetTitle(block.Title).
			SetPrice(block.Price).
			SetCurrency(block.Currency).
			SetContentLength(block.ContentLength).
			SetExcludeFromMembership(block.ExcludeFromMembership)
		if block.OriginalPrice != nil {
			create.SetOriginalPrice(*block.OriginalPrice)
		}
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("创建付费内容块失败: %w", err)
		}
	}

	return nil
}

// ParseAndSyncPaymentsByPublicID 根据文章公开 ID 同步付费块。
func (s *paymentService) ParseAndSyncPaymentsByPublicID(ctx context.Context, articleID string) error {
	decodedArticleID, entityType, err := idgen.DecodePublicID(articleID)
	if err != nil || entityType != idgen.EntityTypeArticle {
		return fmt.Errorf("无效的文章ID: %s", articleID)
	}

	articleEntity, err := s.db.Article.Query().Where(article.IDEQ(decodedArticleID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("查询文章失败: %w", err)
	}

	return s.ParseAndSyncPayments(ctx, decodedArticleID, articleEntity.ContentMd)
}

// CheckAccess 检查用户对付费内容块的访问权限。
func (s *paymentService) CheckAccess(ctx context.Context, req CheckAccessRequest) (*CheckAccessResult, error) {
	paymentEntity, err := s.db.ArticlePayment.Query().
		Where(articlepayment.ArticleIDEQ(req.ArticleID), articlepayment.BlockIDEQ(req.BlockID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return &CheckAccessResult{HasAccess: true, AccessType: "free", Reason: "内容块未配置付费限制"}, nil
		}
		return nil, fmt.Errorf("查询付费内容块失败: %w", err)
	}

	if hasMembershipAccess(paymentEntity, req.Membership) {
		return &CheckAccessResult{HasAccess: true, AccessType: "membership", Reason: "会员权益可访问", Payment: toPaymentSummary(paymentEntity)}, nil
	}

	if req.UserID != "" {
		purchased, err := s.db.ArticlePurchase.Query().
			Where(articlepurchase.UserIDEQ(req.UserID), articlepurchase.ArticleIDEQ(req.ArticleID), articlepurchase.BlockIDEQ(req.BlockID)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询购买记录失败: %w", err)
		}
		if purchased {
			return &CheckAccessResult{HasAccess: true, AccessType: "purchase", Reason: "已完成购买", Payment: toPaymentSummary(paymentEntity)}, nil
		}
	}

	return &CheckAccessResult{HasAccess: false, AccessType: "none", NeedPurchase: true, Reason: "需要购买后访问", Payment: toPaymentSummary(paymentEntity)}, nil
}

// HandleCallback 处理支付回调并写入购买记录。
func (s *paymentService) HandleCallback(ctx context.Context, req PaymentCallbackRequest) (*PaymentCallbackResult, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return nil, fmt.Errorf("user_id 不能为空")
	}
	if req.ArticleID == 0 {
		return nil, fmt.Errorf("article_id 不能为空")
	}
	if strings.TrimSpace(req.BlockID) == "" {
		return nil, fmt.Errorf("block_id 不能为空")
	}
	if req.Price < 0 {
		return nil, fmt.Errorf("price 不能为负数")
	}
	if req.Status != "" {
		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status != "paid" && status != "success" {
			return nil, fmt.Errorf("不支持的支付状态: %s", req.Status)
		}
	}

	purchasedAt := time.Now()
	if req.PurchasedAt != nil {
		purchasedAt = req.PurchasedAt.UTC()
	}

	existing, err := s.db.ArticlePurchase.Query().
		Where(articlepurchase.UserIDEQ(req.UserID), articlepurchase.ArticleIDEQ(req.ArticleID), articlepurchase.BlockIDEQ(req.BlockID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("查询购买记录失败: %w", err)
	}

	if ent.IsNotFound(err) {
		create := s.db.ArticlePurchase.Create().
			SetUserID(req.UserID).
			SetArticleID(req.ArticleID).
			SetBlockID(req.BlockID).
			SetPrice(req.Price).
			SetPurchasedAt(purchasedAt)
		if strings.TrimSpace(req.OrderNo) != "" {
			create.SetOrderNo(strings.TrimSpace(req.OrderNo))
		}
		if _, err := create.Save(ctx); err != nil {
			return nil, fmt.Errorf("创建购买记录失败: %w", err)
		}
		return &PaymentCallbackResult{Recorded: true, PurchasedAt: purchasedAt}, nil
	}

	update := s.db.ArticlePurchase.UpdateOneID(existing.ID).SetPrice(req.Price).SetPurchasedAt(purchasedAt)
	if strings.TrimSpace(req.OrderNo) != "" {
		update.SetOrderNo(strings.TrimSpace(req.OrderNo))
	}
	if _, err := update.Save(ctx); err != nil {
		return nil, fmt.Errorf("更新购买记录失败: %w", err)
	}

	return &PaymentCallbackResult{Recorded: true, PurchasedAt: purchasedAt}, nil
}

func parsePaymentBlocks(markdown string) ([]paymentBlock, error) {
	matches := paidContentBlockRegexp.FindAllString(markdown, -1)
	blocks := make([]paymentBlock, 0, len(matches))
	for _, match := range matches {
		attrs := parseHTMLAttributes(match)
		blockID := firstNonEmpty(attrs["data-block-id"], attrs["data-section-id"])
		if blockID == "" {
			return nil, fmt.Errorf("检测到付费内容块但缺少 data-block-id")
		}

		price, err := parseIntAttr(attrs, "data-price", true)
		if err != nil {
			return nil, fmt.Errorf("解析付费内容块 %s 的价格失败: %w", blockID, err)
		}
		originalPrice, err := parseOptionalIntAttr(attrs, "data-original-price")
		if err != nil {
			return nil, fmt.Errorf("解析付费内容块 %s 的原价失败: %w", blockID, err)
		}
		contentLength, err := parseIntAttr(attrs, "data-content-length", false)
		if err != nil {
			return nil, fmt.Errorf("解析付费内容块 %s 的内容长度失败: %w", blockID, err)
		}

		blocks = append(blocks, paymentBlock{
			BlockID:               blockID,
			Title:                 firstNonEmpty(attrs["data-title"], "付费内容"),
			Price:                 price,
			OriginalPrice:         originalPrice,
			Currency:              firstNonEmpty(attrs["data-currency"], "¥"),
			ContentLength:         contentLength,
			ExcludeFromMembership: parseBoolAttr(attrs["data-exclude-from-membership"]),
		})
	}
	return blocks, nil
}

func parseHTMLAttributes(input string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range htmlAttributeRegexp.FindAllStringSubmatch(input, -1) {
		if len(match) >= 4 {
			attrs[strings.ToLower(match[1])] = strings.TrimSpace(match[3])
		}
	}
	return attrs
}

func parseIntAttr(attrs map[string]string, key string, required bool) (int, error) {
	value := strings.TrimSpace(attrs[key])
	if value == "" {
		if required {
			return 0, fmt.Errorf("%s 不能为空", key)
		}
		return 0, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s 不是有效整数", key)
	}
	if number < 0 {
		return 0, fmt.Errorf("%s 不能为负数", key)
	}
	return number, nil
}

func parseOptionalIntAttr(attrs map[string]string, key string) (*int, error) {
	value := strings.TrimSpace(attrs[key])
	if value == "" {
		return nil, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("%s 不是有效整数", key)
	}
	if number < 0 {
		return nil, fmt.Errorf("%s 不能为负数", key)
	}
	return &number, nil
}

func parseBoolAttr(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func hasMembershipAccess(paymentEntity *ent.ArticlePayment, membership *MembershipStatus) bool {
	if paymentEntity.ExcludeFromMembership || membership == nil {
		return false
	}
	return membership.IsActive && strings.TrimSpace(membership.Tier) != ""
}

func toPaymentSummary(paymentEntity *ent.ArticlePayment) *PaymentBlockSummary {
	return &PaymentBlockSummary{
		ArticleID:             paymentEntity.ArticleID,
		BlockID:               paymentEntity.BlockID,
		Title:                 paymentEntity.Title,
		Price:                 paymentEntity.Price,
		OriginalPrice:         paymentEntity.OriginalPrice,
		Currency:              paymentEntity.Currency,
		ContentLength:         paymentEntity.ContentLength,
		ExcludeFromMembership: paymentEntity.ExcludeFromMembership,
	}
}
