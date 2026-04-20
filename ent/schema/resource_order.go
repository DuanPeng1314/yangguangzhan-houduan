package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ResourceOrder holds the schema definition for the ResourceOrder entity.
type ResourceOrder struct {
	ent.Schema
}

func (ResourceOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("资源本地业务订单映射表"),
	}
}

func (ResourceOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Int64("user_id").Comment("阳光栈本地用户 ID"),
		field.Uint("resource_id").Comment("关联资源 ID"),
		field.Uint("resource_item_id").Optional().Nillable().Comment("关联资源交付物 ID，可空"),
		field.String("business_order_no").MaxLen(64).NotEmpty().Unique().Comment("阳光栈业务订单号"),
		field.String("external_order_no").MaxLen(64).Optional().Comment("外部支付单号，如极光库订单号"),
		field.Float("amount").Default(0).Comment("订单金额"),
		field.String("status").MaxLen(20).Default("pending").Comment("订单状态"),
		field.JSON("snapshot", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{dialect.MySQL: "json", dialect.Postgres: "jsonb", dialect.SQLite: "text"}).
			Comment("下单时的业务快照"),
		field.Time("paid_at").Optional().Nillable().Comment("支付完成时间"),
	}
}

func (ResourceOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", Resource.Type).Ref("orders").Unique().Required().Field("resource_id"),
		edge.From("resource_item", ResourceItem.Type).Ref("orders").Unique().Field("resource_item_id"),
	}
}

func (ResourceOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "resource_id", "status"),
		index.Fields("external_order_no"),
	}
}
