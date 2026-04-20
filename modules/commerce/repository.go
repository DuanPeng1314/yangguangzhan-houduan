package commerce

import (
	"context"

	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/ent/article"
	"github.com/anzhiyu-c/anheyu-app/ent/memberbinding"
)

type BindingRepository struct {
	client *ent.Client
}

func NewBindingRepository(client *ent.Client) *BindingRepository {
	return &BindingRepository{client: client}
}

func (r *BindingRepository) Upsert(ctx context.Context, dto MemberBindingDTO) error {
	create := r.client.MemberBinding.Create().
		SetUserID(dto.UserID).
		SetExternalUserID(dto.ExternalUserID).
		SetSiteID(dto.SiteID).
		SetStatus(dto.Status)
	if dto.LastSyncedAt != nil {
		create.SetLastSyncedAt(*dto.LastSyncedAt)
	}

	return create.
		OnConflictColumns(memberbinding.FieldUserID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *BindingRepository) FindByUserID(ctx context.Context, userID int64) (MemberBindingDTO, error) {
	binding, err := r.client.MemberBinding.Query().
		Where(memberbinding.UserID(userID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return MemberBindingDTO{}, ErrMemberBindingNotFound
		}
		return MemberBindingDTO{}, err
	}

	return MemberBindingDTO{
		UserID:         binding.UserID,
		ExternalUserID: binding.ExternalUserID,
		SiteID:         binding.SiteID,
		Status:         binding.Status,
		LastSyncedAt:   binding.LastSyncedAt,
	}, nil
}

type ArticleContentRepository struct {
	client *ent.Client
}

func NewArticleContentRepository(client *ent.Client) *ArticleContentRepository {
	return &ArticleContentRepository{client: client}
}

func (r *ArticleContentRepository) FindContentHTMLByPremiumContentID(ctx context.Context, contentID string) (string, error) {
	entity, err := r.client.Article.Query().
		Where(
			article.DeletedAtIsNil(),
			article.ContentHTMLContains(`data-content-id="`+contentID+`"`),
		).
		First(ctx)
	if err != nil {
		return "", ErrPremiumBlockNotFound
	}

	return entity.ContentHTML, nil
}
