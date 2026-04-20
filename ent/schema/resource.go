package schema

import (
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Resource holds the schema definition for the Resource entity.
type Resource struct {
	ent.Schema
}

func (Resource) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("独立资源销售单元表"),
	}
}

func (Resource) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

func (Resource) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("host_type").MaxLen(50).Comment("资源宿主类型，如 article 或 product"),
		field.String("host_id").MaxLen(128).Comment("资源宿主公共 ID"),
		field.String("title").MaxLen(255).NotEmpty().Comment("资源标题"),
		field.String("summary").MaxLen(1000).Optional().Comment("资源摘要"),
		field.String("cover_url").MaxLen(255).Optional().Comment("资源封面图 URL"),
		field.String("resource_type").MaxLen(50).Default("download_bundle").Comment("资源类型"),
		field.String("status").MaxLen(20).Default("draft").Comment("资源状态：draft/published/archived"),
		field.Bool("sale_enabled").Default(true).Comment("是否启用销售"),
		field.Float("price").Default(0).Comment("销售价格"),
		field.Float("original_price").Default(0).Comment("划线价格"),
		field.Bool("member_free").Default(false).Comment("会员是否免费"),
		field.Int("sort").Default(0).Comment("同宿主内排序，越小越靠前"),
	}
}

func (Resource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("items", ResourceItem.Type),
		edge.To("orders", ResourceOrder.Type),
		edge.To("access_grants", ResourceAccessGrant.Type),
	}
}

func (Resource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("host_type", "host_id", "status", "sort"),
	}
}
