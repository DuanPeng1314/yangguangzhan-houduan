package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ArticlePurchase 定义文章付费购买记录表结构。
type ArticlePurchase struct {
	ent.Schema
}

// Annotations 定义表注释。
func (ArticlePurchase) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("文章付费购买记录表"),
	}
}

// Fields 定义字段。
func (ArticlePurchase) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.String("user_id").
			MaxLen(64).
			Comment("支付系统用户ID"),
		field.Uint("article_id").
			Comment("文章ID"),
		field.String("block_id").
			MaxLen(64).
			Comment("付费内容块ID"),
		field.Int("price").
			NonNegative().
			Comment("购买价格，单位为分"),
		field.String("order_no").
			Optional().
			Nillable().
			Comment("订单号"),
		field.Time("purchased_at").
			Default(time.Now).
			Comment("购买时间"),
	}
}

// Edges 定义关系。
func (ArticlePurchase) Edges() []ent.Edge {
	return nil
}

// Indexes 定义索引。
func (ArticlePurchase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "article_id", "block_id").Unique(),
	}
}
