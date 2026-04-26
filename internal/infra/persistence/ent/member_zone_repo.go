package ent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/ent/article"
	"github.com/anzhiyu-c/anheyu-app/ent/memberzonecontent"
	"github.com/anzhiyu-c/anheyu-app/ent/predicate"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
)

type MemberZoneRepo struct {
	db *ent.Client
}

func NewMemberZoneRepo(db *ent.Client) *MemberZoneRepo {
	return &MemberZoneRepo{db: db}
}

func (r *MemberZoneRepo) ListAdminMemberZones(ctx context.Context, query commerce.AdminMemberZoneListQueryDTO) (commerce.AdminMemberZoneListDTO, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	predicates := []predicate.MemberZoneContent{memberzonecontent.DeletedAtIsNil()}
	if keyword := strings.TrimSpace(query.Query); keyword != "" {
		predicates = append(predicates, memberzonecontent.Or(
			memberzonecontent.TitleContainsFold(keyword),
			memberzonecontent.SlugContainsFold(keyword),
		))
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		predicates = append(predicates, memberzonecontent.StatusEQ(status))
	}

	contentQuery := r.db.MemberZoneContent.Query().Where(predicates...)
	total, err := contentQuery.Count(ctx)
	if err != nil {
		return commerce.AdminMemberZoneListDTO{}, err
	}

	entities, err := contentQuery.
		Order(ent.Desc(memberzonecontent.FieldUpdatedAt), ent.Desc(memberzonecontent.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return commerce.AdminMemberZoneListDTO{}, err
	}

	items := make([]commerce.AdminMemberZoneListItemDTO, 0, len(entities))
	for _, entity := range entities {
		item, err := r.toAdminMemberZoneListItemDTO(ctx, entity)
		if err != nil {
			return commerce.AdminMemberZoneListDTO{}, err
		}
		items = append(items, item)
	}

	return commerce.AdminMemberZoneListDTO{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *MemberZoneRepo) GetAdminMemberZoneDetail(ctx context.Context, contentID string) (commerce.AdminMemberZoneDetailDTO, error) {
	dbID, err := decodeMemberZoneID(contentID)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, fmt.Errorf("%w: %v", commerce.ErrMemberZoneInvalidInput, err)
	}

	entity, err := r.db.MemberZoneContent.Query().
		Where(memberzonecontent.IDEQ(dbID), memberzonecontent.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminMemberZoneDetailDTO{}, commerce.ErrMemberZoneNotFound
		}
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	return r.toAdminMemberZoneDetailDTO(ctx, entity)
}

func (r *MemberZoneRepo) CreateAdminMemberZone(ctx context.Context, input commerce.AdminMemberZoneDetailDTO) (commerce.AdminMemberZoneDetailDTO, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}
	defer rollbackTx(tx)

	if err := clearArticleBindingForOtherMemberZone(ctx, tx, 0, input.SourceArticleID); err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	create := tx.MemberZoneContent.Create().
		SetTitle(input.Title).
		SetSlug(strings.TrimSpace(input.Slug)).
		SetContentMd(input.ContentMD).
		SetContentHTML(input.ContentHTML).
		SetStatus(input.Status).
		SetAccessLevel(input.AccessLevel).
		SetSort(input.Sort)
	applyMemberZoneOptionalFieldsForCreate(create, input)

	entity, err := create.Save(ctx)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, normalizeMemberZonePersistenceError(err)
	}
	if err := tx.Commit(); err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	return r.GetAdminMemberZoneDetail(ctx, mustMemberZonePublicID(entity.ID))
}

func (r *MemberZoneRepo) UpdateAdminMemberZone(ctx context.Context, contentID string, input commerce.AdminMemberZoneDetailDTO) (commerce.AdminMemberZoneDetailDTO, error) {
	dbID, err := decodeMemberZoneID(contentID)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, fmt.Errorf("%w: %v", commerce.ErrMemberZoneInvalidInput, err)
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}
	defer rollbackTx(tx)

	current, err := tx.MemberZoneContent.Query().
		Where(memberzonecontent.IDEQ(dbID), memberzonecontent.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminMemberZoneDetailDTO{}, commerce.ErrMemberZoneNotFound
		}
		return commerce.AdminMemberZoneDetailDTO{}, normalizeMemberZonePersistenceError(err)
	}

	if err := clearArticleBindingForOtherMemberZone(ctx, tx, dbID, input.SourceArticleID); err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	update := tx.MemberZoneContent.UpdateOneID(dbID).
		SetTitle(input.Title).
		SetSlug(strings.TrimSpace(input.Slug)).
		SetContentMd(input.ContentMD).
		SetContentHTML(input.ContentHTML).
		SetStatus(input.Status).
		SetAccessLevel(input.AccessLevel).
		SetSort(input.Sort)
	applyMemberZoneOptionalFieldsForUpdate(update, input, current.PublishedAt)

	if _, err := update.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminMemberZoneDetailDTO{}, commerce.ErrMemberZoneNotFound
		}
		return commerce.AdminMemberZoneDetailDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	return r.GetAdminMemberZoneDetail(ctx, contentID)
}

func (r *MemberZoneRepo) DeleteAdminMemberZone(ctx context.Context, contentID string) error {
	dbID, err := decodeMemberZoneID(contentID)
	if err != nil {
		return fmt.Errorf("%w: %v", commerce.ErrMemberZoneInvalidInput, err)
	}
	now := time.Now()
	if _, err := r.db.MemberZoneContent.UpdateOneID(dbID).
		Where(memberzonecontent.DeletedAtIsNil()).
		SetDeletedAt(now).
		SetSlug(fmt.Sprintf("deleted-%d-%d", dbID, now.Unix())).
		ClearSourceArticleID().
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return commerce.ErrMemberZoneNotFound
		}
		return normalizeMemberZonePersistenceError(err)
	}
	return nil
}

func (r *MemberZoneRepo) FindAdminMemberZoneByArticle(ctx context.Context, articleID string) (commerce.AdminMemberZoneDetailDTO, error) {
	entity, err := r.db.MemberZoneContent.Query().
		Where(
			memberzonecontent.SourceArticleIDEQ(articleID),
			memberzonecontent.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.AdminMemberZoneDetailDTO{}, commerce.ErrMemberZoneNotFound
		}
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	return r.toAdminMemberZoneDetailDTO(ctx, entity)
}

func (r *MemberZoneRepo) ListPublishedMemberZones(ctx context.Context) ([]commerce.MemberZoneListItemDTO, error) {
	entities, err := r.db.MemberZoneContent.Query().
		Where(
			memberzonecontent.StatusEQ("published"),
			memberzonecontent.DeletedAtIsNil(),
		).
		Order(ent.Asc(memberzonecontent.FieldSort), ent.Desc(memberzonecontent.FieldPublishedAt), ent.Desc(memberzonecontent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]commerce.MemberZoneListItemDTO, 0, len(entities))
	for _, entity := range entities {
		items = append(items, toMemberZoneListItemDTO(entity))
	}
	return items, nil
}

func (r *MemberZoneRepo) GetPublishedMemberZoneMetaBySlug(ctx context.Context, slug string) (commerce.MemberZoneMetaDTO, error) {
	entity, err := r.getPublishedMemberZoneBySlug(ctx, slug)
	if err != nil {
		return commerce.MemberZoneMetaDTO{}, err
	}
	return toMemberZoneMetaDTO(entity), nil
}

func (r *MemberZoneRepo) GetPublishedMemberZoneByArticle(ctx context.Context, articleID string) (commerce.MemberZoneMetaDTO, error) {
	entity, err := r.db.MemberZoneContent.Query().
		Where(
			memberzonecontent.SourceArticleIDEQ(articleID),
			memberzonecontent.StatusEQ("published"),
			memberzonecontent.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.MemberZoneMetaDTO{}, commerce.ErrMemberZoneNotFound
		}
		return commerce.MemberZoneMetaDTO{}, err
	}
	return toMemberZoneMetaDTO(entity), nil
}

func (r *MemberZoneRepo) GetPublishedMemberZoneContentBySlug(ctx context.Context, slug string) (commerce.MemberZoneContentDTO, error) {
	entity, err := r.getPublishedMemberZoneBySlug(ctx, slug)
	if err != nil {
		return commerce.MemberZoneContentDTO{}, err
	}
	return commerce.MemberZoneContentDTO{ContentHTML: entity.ContentHTML}, nil
}

func (r *MemberZoneRepo) getPublishedMemberZoneBySlug(ctx context.Context, slug string) (*ent.MemberZoneContent, error) {
	entity, err := r.db.MemberZoneContent.Query().
		Where(
			memberzonecontent.SlugEQ(strings.TrimSpace(slug)),
			memberzonecontent.StatusEQ("published"),
			memberzonecontent.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, commerce.ErrMemberZoneNotFound
		}
		return nil, err
	}
	return entity, nil
}

func normalizeMemberZonePersistenceError(err error) error {
	if err == nil {
		return nil
	}
	if sqlgraph.IsConstraintError(err) {
		return fmt.Errorf("%w: %v", commerce.ErrMemberZoneConflict, err)
	}
	return err
}

func (r *MemberZoneRepo) toAdminMemberZoneListItemDTO(ctx context.Context, entity *ent.MemberZoneContent) (commerce.AdminMemberZoneListItemDTO, error) {
	contentID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeMemberZone)
	if err != nil {
		return commerce.AdminMemberZoneListItemDTO{}, err
	}

	item := commerce.AdminMemberZoneListItemDTO{
		ContentID:       contentID,
		Title:           entity.Title,
		Slug:            entity.Slug,
		Summary:         entity.Summary,
		Status:          entity.Status,
		AccessLevel:     entity.AccessLevel,
		SourceArticleID: optionalString(entity.SourceArticleID),
		UpdatedAt:       entity.UpdatedAt.Format(time.RFC3339),
	}
	if entity.PublishedAt != nil {
		item.PublishedAt = entity.PublishedAt.Format(time.RFC3339)
	}
	if item.SourceArticleID != "" {
		articleMeta, err := r.getArticleHostMeta(ctx, item.SourceArticleID)
		if err != nil && !errorsIsResourceNotFound(err) {
			return commerce.AdminMemberZoneListItemDTO{}, err
		}
		item.SourceArticleTitle = articleMeta.Title
	}
	return item, nil
}

func (r *MemberZoneRepo) toAdminMemberZoneDetailDTO(ctx context.Context, entity *ent.MemberZoneContent) (commerce.AdminMemberZoneDetailDTO, error) {
	contentID, err := idgen.GeneratePublicID(entity.ID, idgen.EntityTypeMemberZone)
	if err != nil {
		return commerce.AdminMemberZoneDetailDTO{}, err
	}

	detail := commerce.AdminMemberZoneDetailDTO{
		ContentID:       contentID,
		Title:           entity.Title,
		Slug:            entity.Slug,
		Summary:         entity.Summary,
		CoverURL:        entity.CoverURL,
		ContentMD:       entity.ContentMd,
		ContentHTML:     entity.ContentHTML,
		Status:          entity.Status,
		AccessLevel:     entity.AccessLevel,
		Sort:            entity.Sort,
		SourceArticleID: optionalString(entity.SourceArticleID),
		UpdatedAt:       entity.UpdatedAt.Format(time.RFC3339),
	}
	if entity.PublishedAt != nil {
		detail.PublishedAt = entity.PublishedAt.Format(time.RFC3339)
	}
	if detail.SourceArticleID != "" {
		articleMeta, err := r.getArticleHostMeta(ctx, detail.SourceArticleID)
		if err != nil && !errorsIsResourceNotFound(err) {
			return commerce.AdminMemberZoneDetailDTO{}, err
		}
		detail.SourceArticleTitle = articleMeta.Title
		detail.SourceArticleAbbrlink = articleMeta.Abbrlink
	}
	return detail, nil
}

func (r *MemberZoneRepo) getArticleHostMeta(ctx context.Context, articlePublicID string) (articleHostMeta, error) {
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

func toMemberZoneListItemDTO(entity *ent.MemberZoneContent) commerce.MemberZoneListItemDTO {
	contentID := mustMemberZonePublicID(entity.ID)
	item := commerce.MemberZoneListItemDTO{
		ContentID:   contentID,
		Title:       entity.Title,
		Slug:        entity.Slug,
		Summary:     entity.Summary,
		CoverURL:    entity.CoverURL,
		AccessLevel: entity.AccessLevel,
	}
	if entity.PublishedAt != nil {
		item.PublishedAt = entity.PublishedAt.Format(time.RFC3339)
	}
	return item
}

func toMemberZoneMetaDTO(entity *ent.MemberZoneContent) commerce.MemberZoneMetaDTO {
	meta := commerce.MemberZoneMetaDTO{
		ContentID:   mustMemberZonePublicID(entity.ID),
		Title:       entity.Title,
		Slug:        entity.Slug,
		Summary:     entity.Summary,
		CoverURL:    entity.CoverURL,
		AccessLevel: entity.AccessLevel,
	}
	if entity.PublishedAt != nil {
		meta.PublishedAt = entity.PublishedAt.Format(time.RFC3339)
	}
	return meta
}

func applyMemberZoneOptionalFieldsForCreate(setter *ent.MemberZoneContentCreate, input commerce.AdminMemberZoneDetailDTO) {
	if summary := strings.TrimSpace(input.Summary); summary != "" {
		setter.SetSummary(summary)
	}
	if coverURL := strings.TrimSpace(input.CoverURL); coverURL != "" {
		setter.SetCoverURL(coverURL)
	}
	if articleID := strings.TrimSpace(input.SourceArticleID); articleID != "" {
		setter.SetSourceArticleID(articleID)
	}
	if input.Status == "published" {
		setter.SetPublishedAt(time.Now())
	}
}

func applyMemberZoneOptionalFieldsForUpdate(setter *ent.MemberZoneContentUpdateOne, input commerce.AdminMemberZoneDetailDTO, publishedAt *time.Time) {
	if summary := strings.TrimSpace(input.Summary); summary != "" {
		setter.SetSummary(summary)
	} else {
		setter.ClearSummary()
	}
	if coverURL := strings.TrimSpace(input.CoverURL); coverURL != "" {
		setter.SetCoverURL(coverURL)
	} else {
		setter.ClearCoverURL()
	}
	if articleID := strings.TrimSpace(input.SourceArticleID); articleID != "" {
		setter.SetSourceArticleID(articleID)
	} else {
		setter.ClearSourceArticleID()
	}
	if input.Status == "published" {
		if publishedAt == nil {
			setter.SetPublishedAt(time.Now())
		}
	} else {
		setter.ClearPublishedAt()
	}
}

func clearArticleBindingForOtherMemberZone(ctx context.Context, tx *ent.Tx, currentID uint, articleID string) error {
	articleID = strings.TrimSpace(articleID)
	if articleID == "" {
		return nil
	}

	query := tx.MemberZoneContent.Update().
		Where(
			memberzonecontent.SourceArticleIDEQ(articleID),
			memberzonecontent.DeletedAtIsNil(),
		).
		ClearSourceArticleID()
	if currentID > 0 {
		query = query.Where(memberzonecontent.IDNEQ(currentID))
	}
	_, err := query.Save(ctx)
	return err
}

func decodeMemberZoneID(publicID string) (uint, error) {
	dbID, entityType, err := idgen.DecodePublicID(publicID)
	if err != nil {
		return 0, fmt.Errorf("decode member zone id: %w", err)
	}
	if entityType != idgen.EntityTypeMemberZone {
		return 0, fmt.Errorf("invalid member zone entity type: %d", entityType)
	}
	return dbID, nil
}

func mustMemberZonePublicID(dbID uint) string {
	publicID, err := idgen.GeneratePublicID(dbID, idgen.EntityTypeMemberZone)
	if err != nil {
		panic(err)
	}
	return publicID
}
