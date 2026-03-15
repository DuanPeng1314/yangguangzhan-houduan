package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ArticlePayment 定义文章付费内容块表结构。
type ArticlePayment struct {
	ent.Schema
}

// Annotations 定义表注释。
func (ArticlePayment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("文章付费内容块表"),
	}
}

// Fields 定义字段。
func (ArticlePayment) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Uint("article_id").
			Comment("关联文章ID"),
		field.String("block_id").
			MaxLen(64).
			Comment("付费内容块唯一标识"),
		field.String("title").
			Default("付费内容").
			Comment("付费内容块标题"),
		field.Int("price").
			NonNegative().
			Comment("价格，单位为分"),
		field.Int("original_price").
			Optional().
			Nillable().
			NonNegative().
			Comment("原价，单位为分"),
		field.String("currency").
			Default("¥").
			Comment("货币符号"),
		field.Int("content_length").
			Default(0).
			NonNegative().
			Comment("内容字数"),
		field.Bool("exclude_from_membership").
			Default(false).
			Comment("会员是否也需要单独购买"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges 定义关系。
func (ArticlePayment) Edges() []ent.Edge {
	return nil
}

// Indexes 定义索引。
func (ArticlePayment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("article_id", "block_id").Unique(),
	}
}
