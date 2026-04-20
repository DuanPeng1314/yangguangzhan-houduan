package schema

import (
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ResourceItem holds the schema definition for the ResourceItem entity.
type ResourceItem struct {
	ent.Schema
}

func (ResourceItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("资源交付物表"),
	}
}

func (ResourceItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

func (ResourceItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Uint("resource_id").Comment("关联资源 ID"),
		field.String("item_type").MaxLen(30).Default("file").Comment("交付物类型，如 file 或 link"),
		field.String("title").MaxLen(255).NotEmpty().Comment("交付物标题"),
		field.JSON("payload", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{dialect.MySQL: "json", dialect.Postgres: "jsonb", dialect.SQLite: "text"}).
			Comment("交付物差异化负载"),
		field.String("status").MaxLen(20).Default("active").Comment("交付物状态"),
		field.Int("sort").Default(0).Comment("资源内排序，越小越靠前"),
	}
}

func (ResourceItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", Resource.Type).Ref("items").Unique().Required().Field("resource_id"),
		edge.To("orders", ResourceOrder.Type),
		edge.To("access_grants", ResourceAccessGrant.Type),
	}
}

func (ResourceItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource_id", "status", "sort"),
	}
}
