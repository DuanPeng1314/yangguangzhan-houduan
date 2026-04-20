package article

import (
	"context"
	"testing"
	"time"

	"github.com/anzhiyu-c/anheyu-app/pkg/domain/model"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/repository"
	"github.com/anzhiyu-c/anheyu-app/pkg/service/utility"
	"github.com/stretchr/testify/require"
)

type articleRepositoryStub struct {
	article *model.Article
}

func (s *articleRepositoryStub) FindByID(context.Context, uint) (*model.Article, error) {
	panic("unused")
}
func (s *articleRepositoryStub) Create(context.Context, *model.CreateArticleParams) (*model.Article, error) {
	panic("unused")
}
func (s *articleRepositoryStub) GetByID(context.Context, string) (*model.Article, error) {
	panic("unused")
}
func (s *articleRepositoryStub) Update(context.Context, string, *model.UpdateArticleRequest, *model.UpdateArticleComputedParams) (*model.Article, error) {
	panic("unused")
}
func (s *articleRepositoryStub) Delete(context.Context, string) error { panic("unused") }
func (s *articleRepositoryStub) List(context.Context, *model.ListArticlesOptions) ([]*model.Article, int, error) {
	panic("unused")
}
func (s *articleRepositoryStub) GetRandom(context.Context) (*model.Article, error) { panic("unused") }
func (s *articleRepositoryStub) ListHome(context.Context) ([]*model.Article, error) { panic("unused") }
func (s *articleRepositoryStub) ListPublic(context.Context, *model.ListPublicArticlesOptions) ([]*model.Article, int, error) {
	panic("unused")
}
func (s *articleRepositoryStub) GetSiteStats(context.Context) (*model.SiteStats, error) { panic("unused") }
func (s *articleRepositoryStub) IncrementViewCount(context.Context, string) error { panic("unused") }
func (s *articleRepositoryStub) UpdateViewCounts(context.Context, map[uint]int) error { panic("unused") }
func (s *articleRepositoryStub) GetBySlugOrID(context.Context, string) (*model.Article, error) {
	return s.article, nil
}
func (s *articleRepositoryStub) GetBySlugOrIDForPreview(context.Context, string) (*model.Article, error) {
	return s.article, nil
}
func (s *articleRepositoryStub) GetPrevArticle(context.Context, uint, time.Time) (*model.Article, error) {
	return nil, nil
}
func (s *articleRepositoryStub) GetNextArticle(context.Context, uint, time.Time) (*model.Article, error) {
	return nil, nil
}
func (s *articleRepositoryStub) FindRelatedArticles(context.Context, *model.Article, int) ([]*model.Article, error) {
	return nil, nil
}
func (s *articleRepositoryStub) GetArchiveSummary(context.Context) ([]*model.ArchiveItem, error) {
	panic("unused")
}
func (s *articleRepositoryStub) CountByCategoryWithMultipleCategories(context.Context, uint) (int, error) {
	panic("unused")
}
func (s *articleRepositoryStub) FindScheduledArticlesToPublish(context.Context, time.Time) ([]*model.Article, error) {
	panic("unused")
}
func (s *articleRepositoryStub) PublishScheduledArticle(context.Context, uint) error { panic("unused") }
func (s *articleRepositoryStub) ExistsByAbbrlink(context.Context, string, uint) (bool, error) {
	panic("unused")
}
func (s *articleRepositoryStub) ExistsByTitle(context.Context, string, uint) (bool, error) {
	panic("unused")
}

type cacheServiceStub struct{}

func (cacheServiceStub) Set(context.Context, string, interface{}, time.Duration) error { return nil }
func (cacheServiceStub) Get(context.Context, string) (string, error) { return "", nil }
func (cacheServiceStub) Delete(context.Context, ...string) error { return nil }
func (cacheServiceStub) Increment(context.Context, string) (int64, error) { return 0, nil }
func (cacheServiceStub) Expire(context.Context, string, time.Duration) error { return nil }
func (cacheServiceStub) Scan(context.Context, string) ([]string, error) { return nil, nil }
func (cacheServiceStub) GetAndDeleteMany(context.Context, []string) (map[string]int, error) { return nil, nil }
func (cacheServiceStub) RPush(context.Context, string, ...interface{}) error { return nil }
func (cacheServiceStub) LLen(context.Context, string) (int64, error) { return 0, nil }
func (cacheServiceStub) LIndex(context.Context, string, int64) (string, error) { return "", nil }
func (cacheServiceStub) LRange(context.Context, string, int64, int64) ([]string, error) { return nil, nil }
func (cacheServiceStub) Del(context.Context, ...string) error { return nil }
func (cacheServiceStub) SAdd(context.Context, string, ...interface{}) (int64, error) { return 0, nil }

func TestToAPIResponse_StripsPremiumBodyForPublicDetail(t *testing.T) {
	svc := &serviceImpl{}
	article := &model.Article{
		ID:          "art_public_1",
		CreatedAt:   time.Unix(1713500000, 0),
		UpdatedAt:   time.Unix(1713503600, 0),
		Title:       "测试文章",
		Status:      "PUBLISHED",
		ContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-1"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>会员正文内容</p></div><div class="premium-member-content-meta"><span class="content-length">约 4 字</span></div></div></div>`,
	}

	resp := svc.ToAPIResponse(article, false, true)
	resp.ContentHTML = stripPremiumMemberContentForPublic(resp.ContentHTML)

	require.NotContains(t, resp.ContentHTML, "会员正文内容")
	require.Contains(t, resp.ContentHTML, `class="premium-member-content-preview"></div>`)
	require.Contains(t, resp.ContentHTML, `class="content-length">约 4 字</span>`)
}

func TestToAPIResponse_KeepsPremiumBodyForPreview(t *testing.T) {
	svc := &serviceImpl{}
	article := &model.Article{
		ID:          "art_public_1",
		CreatedAt:   time.Unix(1713500000, 0),
		UpdatedAt:   time.Unix(1713503600, 0),
		Title:       "测试文章",
		Status:      "PUBLISHED",
		ContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-1"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>会员正文内容</p></div></div></div>`,
	}

	resp := svc.ToAPIResponse(article, false, true)

	require.Contains(t, resp.ContentHTML, "会员正文内容")
}

func TestGetPublicBySlugOrID_StripsPremiumBody(t *testing.T) {
	svc := &serviceImpl{
		repo:     &articleRepositoryStub{article: &model.Article{ID: "art_public_1", CreatedAt: time.Unix(1713500000, 0), UpdatedAt: time.Unix(1713503600, 0), Title: "测试文章", Status: "PUBLISHED", ContentHTML: `<div class="premium-member-content-editor-preview" data-content-id="premium-1"><div class="premium-member-content-body"><div class="premium-member-content-preview"><p>会员正文内容</p></div><div class="premium-member-content-meta"><span class="content-length">约 4 字</span></div></div></div>`}},
		cacheSvc: cacheServiceStub{},
	}

	resp, err := svc.GetPublicBySlugOrID(context.Background(), "art_public_1")
	require.NoError(t, err)
	require.NotContains(t, resp.ContentHTML, "会员正文内容")
	require.Contains(t, resp.ContentHTML, `class="premium-member-content-preview"></div>`)
}

var _ repository.ArticleRepository = (*articleRepositoryStub)(nil)
var _ utility.CacheService = cacheServiceStub{}
