package ent

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entschema "entgo.io/ent/dialect/sql/schema"
	"github.com/anzhiyu-c/anheyu-app/ent/enttest"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestResourceRepo_FindResourceByID_SkipsSoftDeleted(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-softdelete?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	entity, err := client.Resource.Create().
		SetHostType("article").
		SetHostID("art_public_1").
		SetTitle("已软删除资源").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(29.9).
		SetDeletedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	publicID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResource)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	_, err = repo.FindResourceByID(ctx, publicID)
	require.ErrorIs(t, err, commerce.ErrResourceNotFound)
}

func TestResourceRepo_HasActiveGrant_IgnoresExpiredGrant(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-expired-grant?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	resourceEntity, err := client.Resource.Create().
		SetHostType("article").
		SetHostID("art_public_1").
		SetTitle("时效资源").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(29.9).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ResourceAccessGrant.Create().
		SetUserID(1001).
		SetResourceID(resourceEntity.ID).
		SetGrantType("purchase").
		SetSourceOrderNo("YGZ_RES_001").
		SetStatus("active").
		SetGrantedAt(time.Now().Add(-2 * time.Hour)).
		SetExpiredAt(time.Now().Add(-time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	resourcePublicID, err := idgen.GeneratePublicID(resourceEntity.ID, idgen.EntityTypeResource)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	hasGrant, err := repo.HasActiveGrant(ctx, 1001, resourcePublicID, "")
	require.NoError(t, err)
	require.False(t, hasGrant)
}

func TestResourceRepo_ListAdminResources_IncludesArticleBinding(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-admin-list?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	articleEntity, err := client.Article.Create().
		SetTitle("示例文章").
		SetStatus("PUBLISHED").
		SetAbbrlink("hello-world").
		Save(ctx)
	require.NoError(t, err)

	articlePublicID, err := idgen.GeneratePublicID(articleEntity.ID, idgen.EntityTypeArticle)
	require.NoError(t, err)

	_, err = client.Resource.Create().
		SetHostType("article").
		SetHostID(articlePublicID).
		SetTitle("前端源码包").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(29.9).
		Save(ctx)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	result, err := repo.ListAdminResources(ctx, commerce.AdminResourceListQueryDTO{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	require.Equal(t, "示例文章", result.List[0].HostTitle)
	require.Equal(t, articlePublicID, result.List[0].HostID)
}

func TestResourceRepo_UpdateAdminResource_ReplacesItems(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-admin-update?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	articleEntity, err := client.Article.Create().
		SetTitle("示例文章").
		SetStatus("PUBLISHED").
		SetAbbrlink("hello-world").
		Save(ctx)
	require.NoError(t, err)

	articlePublicID, err := idgen.GeneratePublicID(articleEntity.ID, idgen.EntityTypeArticle)
	require.NoError(t, err)

	resourceEntity, err := client.Resource.Create().
		SetHostType("article").
		SetHostID(articlePublicID).
		SetTitle("旧资源").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(29.9).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ResourceItem.Create().
		SetResourceID(resourceEntity.ID).
		SetItemType("link").
		SetTitle("旧网盘").
		SetPayload(map[string]any{"url": "https://pan.example.com/old"}).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	resourcePublicID, err := idgen.GeneratePublicID(resourceEntity.ID, idgen.EntityTypeResource)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	updated, err := repo.UpdateAdminResource(ctx, resourcePublicID, commerce.AdminResourceDetailDTO{
		Title:         "新资源",
		Status:        "published",
		SaleEnabled:   true,
		Price:         19.9,
		OriginalPrice: 29.9,
		MemberFree:    false,
		HostType:      "article",
		HostID:        articlePublicID,
		Items: []commerce.AdminResourceItemDTO{{
			Title:    "夸克网盘",
			ItemType: "link",
			URL:      "https://pan.example.com/new",
			Status:   "active",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "新资源", updated.Title)
	require.Len(t, updated.Items, 1)
	require.Equal(t, "夸克网盘", updated.Items[0].Title)
	require.Equal(t, "https://pan.example.com/new", updated.Items[0].URL)
}

func TestResourceRepo_SearchArticleHosts_SupportsAbbrlink(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-article-search?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	_, err := client.Article.Create().
		SetTitle("一篇测试文章").
		SetStatus("PUBLISHED").
		SetAbbrlink("resource-bind-demo").
		Save(ctx)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	result, err := repo.SearchArticleHosts(ctx, "resource-bind-demo")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "resource-bind-demo", result[0].Abbrlink)
}

func TestResourceRepo_DeleteAdminResource_SoftDeletesResource(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-admin-delete?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	resourceEntity, err := client.Resource.Create().
		SetHostType("").
		SetHostID("").
		SetTitle("待删除资源").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(9.9).
		Save(ctx)
	require.NoError(t, err)

	resourcePublicID, err := idgen.GeneratePublicID(resourceEntity.ID, idgen.EntityTypeResource)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	require.NoError(t, repo.DeleteAdminResource(ctx, resourcePublicID))

	entity, err := client.Resource.Get(ctx, resourceEntity.ID)
	require.NoError(t, err)
	require.NotNil(t, entity.DeletedAt)

	_, err = repo.GetAdminResourceDetail(ctx, resourcePublicID)
	require.ErrorIs(t, err, commerce.ErrResourceNotFound)
}

func TestResourceRepo_CountResourceOrders_ReturnsAllOrders(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	ctx := context.Background()
	client := enttest.Open(t, dialect.SQLite, "file:resource-repo-order-count?mode=memory&_fk=1", enttest.WithMigrateOptions(entschema.WithGlobalUniqueID(false)))
	defer client.Close()

	resourceEntity, err := client.Resource.Create().
		SetHostType("").
		SetHostID("").
		SetTitle("有订单资源").
		SetStatus("published").
		SetSaleEnabled(true).
		SetPrice(9.9).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ResourceOrder.Create().
		SetUserID(1001).
		SetResourceID(resourceEntity.ID).
		SetBusinessOrderNo("YGZ_RES_001").
		SetExternalOrderNo("EXT_001").
		SetAmount(9.9).
		SetStatus("pending").
		SetSnapshot(map[string]any{"title": "有订单资源"}).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ResourceOrder.Create().
		SetUserID(1002).
		SetResourceID(resourceEntity.ID).
		SetBusinessOrderNo("YGZ_RES_002").
		SetExternalOrderNo("EXT_002").
		SetAmount(9.9).
		SetStatus("paid").
		SetSnapshot(map[string]any{"title": "有订单资源"}).
		Save(ctx)
	require.NoError(t, err)

	resourcePublicID, err := idgen.GeneratePublicID(resourceEntity.ID, idgen.EntityTypeResource)
	require.NoError(t, err)

	repo := NewResourceRepo(client)
	count, err := repo.CountResourceOrders(ctx, resourcePublicID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
