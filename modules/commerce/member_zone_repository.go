package commerce

import "context"

type memberZoneRepository interface {
	ListAdminMemberZones(ctx context.Context, query AdminMemberZoneListQueryDTO) (AdminMemberZoneListDTO, error)
	GetAdminMemberZoneDetail(ctx context.Context, contentID string) (AdminMemberZoneDetailDTO, error)
	CreateAdminMemberZone(ctx context.Context, input AdminMemberZoneDetailDTO) (AdminMemberZoneDetailDTO, error)
	UpdateAdminMemberZone(ctx context.Context, contentID string, input AdminMemberZoneDetailDTO) (AdminMemberZoneDetailDTO, error)
	DeleteAdminMemberZone(ctx context.Context, contentID string) error
	FindAdminMemberZoneByArticle(ctx context.Context, articleID string) (AdminMemberZoneDetailDTO, error)
	ListPublishedMemberZones(ctx context.Context) ([]MemberZoneListItemDTO, error)
	GetPublishedMemberZoneMetaBySlug(ctx context.Context, slug string) (MemberZoneMetaDTO, error)
	GetPublishedMemberZoneByArticle(ctx context.Context, articleID string) (MemberZoneMetaDTO, error)
	GetPublishedMemberZoneContentBySlug(ctx context.Context, slug string) (MemberZoneContentDTO, error)
}
