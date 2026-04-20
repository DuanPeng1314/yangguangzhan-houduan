package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MemberBinding stores the mapping between a local user and dp7575 identity.
type MemberBinding struct {
	ent.Schema
}

func (MemberBinding) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("会员身份映射表"),
	}
}

func (MemberBinding) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique().
			Comment("阳光栈本地用户 ID"),
		field.String("external_user_id").
			NotEmpty().
			Comment("极光库外部用户标识"),
		field.String("site_id").
			NotEmpty().
			Comment("极光库站点标识"),
		field.String("status").
			Default("active").
			Comment("映射状态"),
		field.Time("last_synced_at").
			Optional().
			Nillable().
			Comment("最近同步时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (MemberBinding) Edges() []ent.Edge {
	return nil
}

func (MemberBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").Unique(),
		index.Fields("site_id", "external_user_id").Unique(),
	}
}
