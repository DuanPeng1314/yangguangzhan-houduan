package commerce

import (
	"context"
	"time"
)

type resourceRepository interface {
	FindResourceByID(ctx context.Context, resourceID string) (ResourceRecordDTO, error)
	FindResourceByHost(ctx context.Context, hostType, hostID string) (ResourceRecordDTO, error)
	ListAdminResources(ctx context.Context, query AdminResourceListQueryDTO) (AdminResourceListDTO, error)
	GetAdminResourceDetail(ctx context.Context, resourceID string) (AdminResourceDetailDTO, error)
	CreateAdminResource(ctx context.Context, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error)
	UpdateAdminResource(ctx context.Context, resourceID string, input AdminResourceDetailDTO) (AdminResourceDetailDTO, error)
	BindAdminResourceToArticle(ctx context.Context, resourceID string, articleID string) (AdminResourceDetailDTO, error)
	DeleteAdminResource(ctx context.Context, resourceID string) error
	CountResourceOrders(ctx context.Context, resourceID string) (int, error)
	SearchArticleHosts(ctx context.Context, query string) ([]AdminArticleHostOptionDTO, error)
	FindResourceByArticleHost(ctx context.Context, articleID string) (AdminResourceDetailDTO, error)
	ResolveArticleIDByAbbrlink(ctx context.Context, abbrlink string) (string, error)
	HasActiveGrant(ctx context.Context, userID int64, resourceID string, resourceItemID string) (bool, error)
	HasGrantBySourceOrderNo(ctx context.Context, userID int64, sourceOrderNo string) (bool, error)
	CreateGrant(ctx context.Context, input ResourceAccessGrantCreateDTO) error
}

type resourceOrderRepository interface {
	Create(ctx context.Context, input ResourceOrderCreateDTO) (ResourceOrderRecordDTO, error)
	MarkPaid(ctx context.Context, businessOrderNo string, externalOrderNo string, paidAt *time.Time) (bool, error)
	UpdateExternalOrderNo(ctx context.Context, businessOrderNo string, externalOrderNo string) error
	FindLatestPendingByUserAndResource(ctx context.Context, userID int64, resourceID string) (ResourceOrderRecordDTO, error)
	FindByBusinessOrderNo(ctx context.Context, businessOrderNo string) (ResourceOrderRecordDTO, error)
	FindByExternalOrderNo(ctx context.Context, externalOrderNo string) (ResourceOrderRecordDTO, error)
}
