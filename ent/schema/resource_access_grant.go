package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ResourceAccessGrant holds the schema definition for the ResourceAccessGrant entity.
type ResourceAccessGrant struct {
	ent.Schema
}

func (ResourceAccessGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("资源访问权益表"),
	}
}

func (ResourceAccessGrant) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Int64("user_id").Comment("阳光栈本地用户 ID"),
		field.Uint("resource_id").Comment("关联资源 ID"),
		field.Uint("resource_item_id").Optional().Nillable().Comment("关联资源交付物 ID，可空"),
		field.String("grant_type").MaxLen(20).Default("purchase").Comment("权益来源类型"),
		field.String("source_order_no").MaxLen(64).Optional().Comment("来源业务订单号"),
		field.String("status").MaxLen(20).Default("active").Comment("权益状态"),
		field.Time("granted_at").Default(time.Now).Comment("权益生效时间"),
		field.Time("expired_at").Optional().Nillable().Comment("权益过期时间，可空"),
	}
}

func (ResourceAccessGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", Resource.Type).Ref("access_grants").Unique().Required().Field("resource_id"),
		edge.From("resource_item", ResourceItem.Type).Ref("access_grants").Unique().Field("resource_item_id"),
	}
}

func (ResourceAccessGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "resource_id", "status"),
		index.Fields("source_order_no"),
	}
}
