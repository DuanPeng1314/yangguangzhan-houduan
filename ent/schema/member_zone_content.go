package schema

import (
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MemberZoneContent holds the schema definition for the MemberZoneContent entity.
type MemberZoneContent struct {
	ent.Schema
}

func (MemberZoneContent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("会员专区内容表"),
	}
}

func (MemberZoneContent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

func (MemberZoneContent) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("title").MaxLen(255).NotEmpty().Comment("内容标题"),
		field.String("slug").MaxLen(128).NotEmpty().Unique().Comment("内容短链"),
		field.String("summary").MaxLen(1000).Optional().Comment("内容摘要"),
		field.String("cover_url").MaxLen(255).Optional().Comment("封面图 URL"),
		field.Text("content_md").Comment("Markdown 原文"),
		field.Text("content_html").Comment("渲染后的 HTML"),
		field.String("status").MaxLen(20).Default("draft").Comment("状态：draft/published/archived"),
		field.String("access_level").MaxLen(20).Default("member").Comment("访问等级：member/premium"),
		field.Int("sort").Default(0).Comment("排序值，越小越靠前"),
		field.String("source_article_id").MaxLen(128).Optional().Nillable().Comment("关联文章公共 ID"),
		field.Time("published_at").Optional().Nillable().Comment("发布时间"),
	}
}

func (MemberZoneContent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "sort", "published_at"),
		index.Fields("source_article_id").Unique(),
	}
}
