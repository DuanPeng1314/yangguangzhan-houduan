package ent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/ent/article"
	"github.com/anzhiyu-c/anheyu-app/ent/predicate"
	"github.com/anzhiyu-c/anheyu-app/ent/resource"
	"github.com/anzhiyu-c/anheyu-app/ent/resourceaccessgrant"
	"github.com/anzhiyu-c/anheyu-app/ent/resourceitem"
	"github.com/anzhiyu-c/anheyu-app/ent/resourceorder"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
)

type ResourceRepo struct {
	db *ent.Client
}

func NewResourceRepo(db *ent.Client) *ResourceRepo {
	return &ResourceRepo{db: db}
}

func (r *ResourceRepo) FindResourceByID(ctx context.Context, resourceID string) (commerce.ResourceRecordDTO, error) {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return commerce.ResourceRecordDTO{}, err
	}

	entity, err := r.db.Resource.Query().
		Where(resource.IDEQ(dbID), resource.DeletedAtIsNil()).
		WithItems(func(query *ent.ResourceItemQuery) {
			query.Where(resourceitem.DeletedAtIsNil()).Order(ent.Asc(resourceitem.FieldSort), ent.Asc(resourceitem.FieldID))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.ResourceRecordDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.ResourceRecordDTO{}, err
	}

	return toResourceRecordDTO(entity)
}

func (r *ResourceRepo) FindResourceByHost(ctx context.Context, hostType, hostID string) (commerce.ResourceRecordDTO, error) {
	entity, err := r.db.Resource.Query().
		Where(
			resource.HostTypeEQ(hostType),
			resource.HostIDEQ(hostID),
			resource.StatusEQ("published"),
			resource.DeletedAtIsNil(),
		).
		WithItems(func(query *ent.ResourceItemQuery) {
			query.Where(resourceitem.DeletedAtIsNil()).Order(ent.Asc(resourceitem.FieldSort), ent.Asc(resourceitem.FieldID))
		}).
		Order(ent.Asc(resource.FieldSort), ent.Asc(resource.FieldID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.ResourceRecordDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.ResourceRecordDTO{}, err
	}

	return toResourceRecordDTO(entity)
}

func (r *ResourceRepo) ArticleHostExists(ctx context.Context, articleID string) (bool, error) {
	_, err := r.getArticleHostMeta(ctx, articleID)
	if err != nil {
		if errorsIsResourceNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ResourceRepo) ListAdminResources(ctx context.Context, query commerce.AdminResourceListQueryDTO) (commerce.AdminResourceListDTO, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	predicates := []predicate.Resource{resource.DeletedAtIsNil()}
	if strings.TrimSpace(query.Query) != "" {
		predicates = append(predicates, resource.TitleContainsFold(strings.TrimSpace(query.Query)))
	}
	if strings.TrimSpace(query.Status) != "" {
		predicates = append(predicates, resource.StatusEQ(strings.TrimSpace(query.Status)))
	}

	resourceQuery := r.db.Resource.Query().Where(predicates...)
	total, err := resourceQuery.Count(ctx)
	if err != nil {
		return commerce.AdminResourceListDTO{}, err
	}

	entities, err := r.db.Resource.Query().
		Where(predicates...).
		Order(ent.Desc(resource.FieldUpdatedAt), ent.Desc(resource.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return commerce.AdminResourceListDTO{}, err
	}

	items := make([]commerce.AdminResourceListItemDTO, 0, len(entities))
	for _, entity := range entities {
		item, err := r.toAdminResourceListItemDTO(ctx, entity)
		if err != nil {
			return commerce.AdminResourceListDTO{}, err
		}
		items = append(items, item)
	}

	return commerce.AdminResourceListDTO{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *ResourceRepo) GetAdminResourceDetail(ctx context.Context, resourceID string) (commerce.AdminResourceDetailDTO, error) {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	entity, err := r.db.Resource.Query().
		Where(resource.IDEQ(dbID), resource.DeletedAtIsNil()).
		WithItems(func(query *ent.ResourceItemQuery) {
			query.Where(resourceitem.DeletedAtIsNil()).Order(ent.Asc(resourceitem.FieldSort), ent.Asc(resourceitem.FieldID))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminResourceDetailDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.AdminResourceDetailDTO{}, err
	}

	return r.toAdminResourceDetailDTO(ctx, entity)
}

func (r *ResourceRepo) CreateAdminResource(ctx context.Context, input commerce.AdminResourceDetailDTO) (commerce.AdminResourceDetailDTO, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	defer rollbackTx(tx)

	create := tx.Resource.Create().
		SetHostType(input.HostType).
		SetHostID(input.HostID).
		SetTitle(input.Title).
		SetStatus(input.Status).
		SetSaleEnabled(input.SaleEnabled).
		SetPrice(input.Price).
		SetOriginalPrice(input.OriginalPrice).
		SetMemberFree(input.MemberFree)
	if input.Summary != "" {
		create.SetSummary(input.Summary)
	}
	if input.CoverURL != "" {
		create.SetCoverURL(input.CoverURL)
	}

	entity, err := create.Save(ctx)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	if err := syncAdminResourceItems(ctx, tx, entity.ID, input.Items); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	return r.GetAdminResourceDetail(ctx, mustResourcePublicID(entity.ID))
}

func (r *ResourceRepo) UpdateAdminResource(ctx context.Context, resourceID string, input commerce.AdminResourceDetailDTO) (commerce.AdminResourceDetailDTO, error) {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	defer rollbackTx(tx)

	update := tx.Resource.UpdateOneID(dbID).
		SetHostType(input.HostType).
		SetHostID(input.HostID).
		SetTitle(input.Title).
		SetStatus(input.Status).
		SetSaleEnabled(input.SaleEnabled).
		SetPrice(input.Price).
		SetOriginalPrice(input.OriginalPrice).
		SetMemberFree(input.MemberFree)

	if input.Summary != "" {
		update.SetSummary(input.Summary)
	} else {
		update.ClearSummary()
	}
	if input.CoverURL != "" {
		update.SetCoverURL(input.CoverURL)
	} else {
		update.ClearCoverURL()
	}

	if _, err := update.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminResourceDetailDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.AdminResourceDetailDTO{}, err
	}
	if err := syncAdminResourceItems(ctx, tx, dbID, input.Items); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	return r.GetAdminResourceDetail(ctx, resourceID)
}

func (r *ResourceRepo) BindAdminResourceToArticle(ctx context.Context, resourceID string, articleID string) (commerce.AdminResourceDetailDTO, error) {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	defer rollbackTx(tx)

	target, err := tx.Resource.Query().Where(resource.IDEQ(dbID), resource.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminResourceDetailDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.AdminResourceDetailDTO{}, err
	}

	current, err := tx.Resource.Query().
		Where(resource.HostTypeEQ("article"), resource.HostIDEQ(articleID), resource.DeletedAtIsNil()).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return commerce.AdminResourceDetailDTO{}, err
	}
	if err == nil && current.ID != target.ID {
		if _, err := tx.Resource.UpdateOneID(current.ID).SetHostType("").SetHostID("").Save(ctx); err != nil {
			return commerce.AdminResourceDetailDTO{}, err
		}
	}

	if _, err := tx.Resource.UpdateOneID(target.ID).SetHostType("article").SetHostID(articleID).Save(ctx); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	if err := tx.Commit(); err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}

	return r.GetAdminResourceDetail(ctx, resourceID)
}

func (r *ResourceRepo) DeleteAdminResource(ctx context.Context, resourceID string) error {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return err
	}
	now := time.Now()
	if _, err := r.db.Resource.UpdateOneID(dbID).Where(resource.DeletedAtIsNil()).SetDeletedAt(now).Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return commerce.ErrResourceNotFound
		}
		return err
	}
	return nil
}

func (r *ResourceRepo) CountResourceOrders(ctx context.Context, resourceID string) (int, error) {
	dbID, err := decodeResourceID(resourceID)
	if err != nil {
		return 0, err
	}
	return r.db.ResourceOrder.Query().Where(resourceorder.ResourceIDEQ(dbID)).Count(ctx)
}

func (r *ResourceRepo) SearchArticleHosts(ctx context.Context, query string) ([]commerce.AdminArticleHostOptionDTO, error) {
	articleQuery := r.db.Article.Query().Where(article.DeletedAtIsNil())
	if strings.TrimSpace(query) != "" {
		keyword := strings.TrimSpace(query)
		articleQuery = articleQuery.Where(article.Or(article.TitleContainsFold(keyword), article.AbbrlinkContainsFold(keyword)))
	}
	entities, err := articleQuery.Order(ent.Desc(article.FieldUpdatedAt), ent.Desc(article.FieldID)).Limit(20).All(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]commerce.AdminArticleHostOptionDTO, 0, len(entities))
	for _, entity := range entities {
		publicID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeArticle)
		if err != nil {
			return nil, err
		}
		options = append(options, commerce.AdminArticleHostOptionDTO{
			ArticleID: publicID,
			Title:     entity.Title,
			Abbrlink:  optionalString(entity.Abbrlink),
			Status:    string(entity.Status),
		})
	}
	return options, nil
}

func (r *ResourceRepo) FindResourceByArticleHost(ctx context.Context, articleID string) (commerce.AdminResourceDetailDTO, error) {
	entity, err := r.db.Resource.Query().
		Where(resource.HostTypeEQ("article"), resource.HostIDEQ(articleID), resource.DeletedAtIsNil()).
		Order(ent.Asc(resource.FieldSort), ent.Asc(resource.FieldID)).
		WithItems(func(query *ent.ResourceItemQuery) {
			query.Where(resourceitem.DeletedAtIsNil()).Order(ent.Asc(resourceitem.FieldSort), ent.Asc(resourceitem.FieldID))
		}).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminResourceDetailDTO{}, commerce.ErrResourceNotFound
		}
		return commerce.AdminResourceDetailDTO{}, err
	}
	return r.toAdminResourceDetailDTO(ctx, entity)
}

func (r *ResourceRepo) ResolveArticleIDByAbbrlink(ctx context.Context, abbrlink string) (string, error) {
	entity, err := r.db.Article.Query().Where(article.AbbrlinkEQ(abbrlink), article.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", commerce.ErrResourceNotFound
		}
		return "", err
	}

	publicID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeArticle)
	if err != nil {
		return "", err
	}
	return publicID, nil
}

func (r *ResourceRepo) HasActiveGrant(ctx context.Context, userID int64, resourceID string, resourceItemID string) (bool, error) {
	resourceDBID, err := decodeResourceID(resourceID)
	if err != nil {
		return false, err
	}

	predicates := []predicate.ResourceAccessGrant{
		resourceaccessgrant.UserIDEQ(userID),
		resourceaccessgrant.StatusEQ("active"),
		resourceaccessgrant.ResourceIDEQ(resourceDBID),
		resourceaccessgrant.Or(resourceaccessgrant.ExpiredAtIsNil(), resourceaccessgrant.ExpiredAtGT(time.Now())),
	}

	if resourceItemID != "" {
		resourceItemDBID, err := decodeResourceItemID(resourceItemID)
		if err != nil {
			return false, err
		}
		predicates = append(predicates, resourceaccessgrant.Or(
			resourceaccessgrant.ResourceItemIDIsNil(),
			resourceaccessgrant.ResourceItemIDEQ(resourceItemDBID),
		))
	} else {
		predicates = append(predicates, resourceaccessgrant.ResourceItemIDIsNil())
	}

	exists, err := r.db.ResourceAccessGrant.Query().Where(predicates...).Exist(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *ResourceRepo) HasGrantBySourceOrderNo(ctx context.Context, userID int64, sourceOrderNo string) (bool, error) {
	if sourceOrderNo == "" {
		return false, nil
	}

	return r.db.ResourceAccessGrant.Query().Where(
		resourceaccessgrant.UserIDEQ(userID),
		resourceaccessgrant.SourceOrderNoEQ(sourceOrderNo),
		resourceaccessgrant.StatusEQ("active"),
		resourceaccessgrant.Or(resourceaccessgrant.ExpiredAtIsNil(), resourceaccessgrant.ExpiredAtGT(time.Now())),
	).Exist(ctx)
}

func (r *ResourceRepo) CreateGrant(ctx context.Context, input commerce.ResourceAccessGrantCreateDTO) error {
	resourceDBID, err := decodeResourceID(input.ResourceID)
	if err != nil {
		return err
	}

	create := r.db.ResourceAccessGrant.Create().
		SetUserID(input.UserID).
		SetResourceID(resourceDBID).
		SetGrantType(defaultString(input.GrantType, "purchase")).
		SetSourceOrderNo(input.SourceOrderNo).
		SetStatus(defaultString(input.Status, "active"))

	if input.GrantedAt != nil {
		create.SetGrantedAt(*input.GrantedAt)
	}
	if input.ExpiredAt != nil {
		create.SetExpiredAt(*input.ExpiredAt)
	}
	if input.ResourceItemID != "" {
		itemDBID, err := decodeResourceItemID(input.ResourceItemID)
		if err != nil {
			return err
		}
		create.SetResourceItemID(itemDBID)
	}

	return create.Exec(ctx)
}

func toResourceRecordDTO(entity *ent.Resource) (commerce.ResourceRecordDTO, error) {
	publicID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResource)
	if err != nil {
		return commerce.ResourceRecordDTO{}, err
	}

	record := commerce.ResourceRecordDTO{
		ResourceID:    publicID,
		HostType:      entity.HostType,
		HostID:        entity.HostID,
		Title:         entity.Title,
		Summary:       entity.Summary,
		CoverURL:      entity.CoverURL,
		ResourceType:  entity.ResourceType,
		Status:        entity.Status,
		SaleEnabled:   entity.SaleEnabled,
		Price:         entity.Price,
		OriginalPrice: entity.OriginalPrice,
		MemberFree:    entity.MemberFree,
		ResourceItems: make([]commerce.ResourceAccessItemDTO, 0, len(entity.Edges.Items)),
	}
	for _, item := range entity.Edges.Items {
		itemDTO, err := toResourceAccessItemDTO(item)
		if err != nil {
			return commerce.ResourceRecordDTO{}, err
		}
		record.ResourceItems = append(record.ResourceItems, itemDTO)
	}
	return record, nil
}

func toResourceAccessItemDTO(entity *ent.ResourceItem) (commerce.ResourceAccessItemDTO, error) {
	itemID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResourceItem)
	if err != nil {
		return commerce.ResourceAccessItemDTO{}, err
	}
	payload := entity.Payload
	return commerce.ResourceAccessItemDTO{
		ID:             itemID,
		Title:          entity.Title,
		Description:    stringFromPayload(payload, "description"),
		URL:            stringFromPayload(payload, "url"),
		ExtractionCode: stringFromPayload(payload, "extraction_code"),
		Note:           stringFromPayload(payload, "note"),
	}, nil
}

func decodeResourceID(publicID string) (uint, error) {
	dbID, entityType, err := idgen.DecodePublicID(publicID)
	if err != nil {
		return 0, fmt.Errorf("decode resource id: %w", err)
	}
	if entityType != idgen.EntityTypeResource {
		return 0, fmt.Errorf("invalid resource entity type: %d", entityType)
	}
	return dbID, nil
}

func decodeResourceItemID(publicID string) (uint, error) {
	dbID, entityType, err := idgen.DecodePublicID(publicID)
	if err != nil {
		return 0, fmt.Errorf("decode resource item id: %w", err)
	}
	if entityType != idgen.EntityTypeResourceItem {
		return 0, fmt.Errorf("invalid resource item entity type: %d", entityType)
	}
	return dbID, nil
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (r *ResourceRepo) toAdminResourceListItemDTO(ctx context.Context, entity *ent.Resource) (commerce.AdminResourceListItemDTO, error) {
	resourceID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResource)
	if err != nil {
		return commerce.AdminResourceListItemDTO{}, err
	}
	item := commerce.AdminResourceListItemDTO{
		ResourceID:    resourceID,
		Title:         entity.Title,
		Status:        entity.Status,
		SaleEnabled:   entity.SaleEnabled,
		Price:         entity.Price,
		OriginalPrice: entity.OriginalPrice,
		MemberFree:    entity.MemberFree,
		HostType:      entity.HostType,
		HostID:        entity.HostID,
		UpdatedAt:     entity.UpdatedAt.Format(time.RFC3339),
	}
	orderCount, err := r.db.ResourceOrder.Query().Where(resourceorder.ResourceIDEQ(entity.ID)).Count(ctx)
	if err != nil {
		return commerce.AdminResourceListItemDTO{}, err
	}
	item.OrderCount = orderCount
	if entity.HostType == "article" && entity.HostID != "" {
		articleMeta, err := r.getArticleHostMeta(ctx, entity.HostID)
		if err != nil && !errorsIsResourceNotFound(err) {
			return commerce.AdminResourceListItemDTO{}, err
		}
		item.HostTitle = articleMeta.Title
	}
	return item, nil
}

func (r *ResourceRepo) toAdminResourceDetailDTO(ctx context.Context, entity *ent.Resource) (commerce.AdminResourceDetailDTO, error) {
	resourceID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResource)
	if err != nil {
		return commerce.AdminResourceDetailDTO{}, err
	}
	detail := commerce.AdminResourceDetailDTO{
		ResourceID:    resourceID,
		Title:         entity.Title,
		Summary:       entity.Summary,
		CoverURL:      entity.CoverURL,
		Status:        entity.Status,
		SaleEnabled:   entity.SaleEnabled,
		Price:         entity.Price,
		OriginalPrice: entity.OriginalPrice,
		MemberFree:    entity.MemberFree,
		HostType:      entity.HostType,
		HostID:        entity.HostID,
		Items:         make([]commerce.AdminResourceItemDTO, 0, len(entity.Edges.Items)),
	}
	if entity.HostType == "article" && entity.HostID != "" {
		articleMeta, err := r.getArticleHostMeta(ctx, entity.HostID)
		if err != nil && !errorsIsResourceNotFound(err) {
			return commerce.AdminResourceDetailDTO{}, err
		}
		detail.HostTitle = articleMeta.Title
		detail.HostAbbrlink = articleMeta.Abbrlink
	}
	for _, item := range entity.Edges.Items {
		itemDTO, err := toAdminResourceItemDTO(item)
		if err != nil {
			return commerce.AdminResourceDetailDTO{}, err
		}
		detail.Items = append(detail.Items, itemDTO)
	}
	return detail, nil
}

type articleHostMeta struct {
	Title    string
	Abbrlink string
}

func (r *ResourceRepo) getArticleHostMeta(ctx context.Context, articlePublicID string) (articleHostMeta, error) {
	articleDBID, err := decodeArticleID(articlePublicID)
	if err != nil {
		return articleHostMeta{}, err
	}
	entity, err := r.db.Article.Query().Where(article.IDEQ(articleDBID), article.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return articleHostMeta{}, commerce.ErrResourceNotFound
		}
		return articleHostMeta{}, err
	}
	return articleHostMeta{Title: entity.Title, Abbrlink: optionalString(entity.Abbrlink)}, nil
}

func toAdminResourceItemDTO(entity *ent.ResourceItem) (commerce.AdminResourceItemDTO, error) {
	itemID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeResourceItem)
	if err != nil {
		return commerce.AdminResourceItemDTO{}, err
	}
	payload := entity.Payload
	return commerce.AdminResourceItemDTO{
		ID:             itemID,
		Title:          entity.Title,
		ItemType:       entity.ItemType,
		URL:            stringFromPayload(payload, "url"),
		ExtractionCode: stringFromPayload(payload, "extraction_code"),
		Note:           stringFromPayload(payload, "note"),
		Sort:           entity.Sort,
		Status:         entity.Status,
	}, nil
}

func syncAdminResourceItems(ctx context.Context, tx *ent.Tx, resourceDBID uint, items []commerce.AdminResourceItemDTO) error {
	if _, err := tx.ResourceItem.Update().Where(resourceitem.ResourceIDEQ(resourceDBID), resourceitem.DeletedAtIsNil()).SetDeletedAt(time.Now()).Save(ctx); err != nil {
		return err
	}
	for _, item := range items {
		create := tx.ResourceItem.Create().
			SetResourceID(resourceDBID).
			SetItemType(defaultString(item.ItemType, "link")).
			SetTitle(item.Title).
			SetStatus(defaultString(item.Status, "active")).
			SetSort(item.Sort).
			SetPayload(map[string]interface{}{
				"url":             item.URL,
				"extraction_code": item.ExtractionCode,
				"note":            item.Note,
			})
		if _, err := create.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func decodeArticleID(publicID string) (uint, error) {
	dbID, entityType, err := idgen.DecodePublicID(publicID)
	if err != nil {
		return 0, fmt.Errorf("decode article id: %w", err)
	}
	if entityType != idgen.EntityTypeArticle {
		return 0, fmt.Errorf("invalid article entity type: %d", entityType)
	}
	return dbID, nil
}

func mustResourcePublicID(dbID uint) string {
	publicID, err := idgen.GeneratePublicID(dbID, idgen.EntityTypeResource)
	if err != nil {
		panic(err)
	}
	return publicID
}

func stringFromPayload(payload map[string]interface{}, key string) string {
	if payload == nil || payload[key] == nil {
		return ""
	}
	return fmt.Sprint(payload[key])
}

func rollbackTx(tx *ent.Tx) {
	_ = tx.Rollback()
}

func errorsIsResourceNotFound(err error) bool {
	return err == commerce.ErrResourceNotFound
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
