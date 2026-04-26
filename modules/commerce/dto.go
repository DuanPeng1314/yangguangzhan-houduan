package commerce

import "time"

type MemberBindingDTO struct {
	UserID         int64
	ExternalUserID string
	SiteID         string
	Status         string
	LastSyncedAt   *time.Time
}

type MemberStatusDTO struct {
	IsMember  bool   `json:"is_member"`
	Level     string `json:"level"`
	ExpiresAt string `json:"expires_at"`
	State     string `json:"state,omitempty"`
	Message   string `json:"message,omitempty"`
}

type MemberOrderSummaryDTO struct {
	OrderNo   string `json:"order_no"`
	Status    string `json:"status"`
	Amount    string `json:"amount"`
	CreatedAt string `json:"created_at"`
}

type MemberProfileDTO struct {
	IsMember     bool                    `json:"is_member"`
	Level        string                  `json:"level"`
	ExpiresAt    string                  `json:"expires_at"`
	State        string                  `json:"state,omitempty"`
	Message      string                  `json:"message,omitempty"`
	RecentOrders []MemberOrderSummaryDTO `json:"recent_orders"`
}

type HealthCheckItemDTO struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Result      string `json:"result"`
	Detail      string `json:"detail,omitempty"`
}

type HealthCheckDTO struct {
	Items []HealthCheckItemDTO `json:"items"`
}

type MemberPurchasePriceOptionDTO struct {
	ProductID     string  `json:"product_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	ActionType    string  `json:"action_type"`
	Tag           string  `json:"tag,omitempty"`
}

type MemberPurchaseMemberTypeDTO struct {
	Level        string                         `json:"level"`
	Name         string                         `json:"name"`
	PriceOptions []MemberPurchasePriceOptionDTO `json:"price_options"`
}

type MemberPurchaseCatalogDTO struct {
	MemberTypes    []MemberPurchaseMemberTypeDTO `json:"member_types"`
	PaymentMethods []string                      `json:"payment_methods"`
}

type MemberOrderCreateDTO struct {
	ProductID     string `json:"product_id"`
	PaymentMethod string `json:"payment_method"`
}

type MemberOrderDTO struct {
	OrderNo    string  `json:"order_no"`
	Status     string  `json:"status"`
	PayType    string  `json:"pay_type"`
	PayURL     string  `json:"pay_url"`
	OrderPrice float64 `json:"order_price"`
	CreatedAt  string  `json:"created_at"`
}

type MemberOrderPaymentDetailDTO struct {
	OrderNo string `json:"order_no"`
}

type MemberPaymentDetailDTO struct {
	OrderNo    string         `json:"order_no"`
	Amount     float64        `json:"amount"`
	PayType    string         `json:"pay_type"`
	PayChannel string         `json:"pay_channel"`
	PayTime    string         `json:"pay_time,omitempty"`
	PayDetail  map[string]any `json:"pay_detail"`
}

type MemberCardRedeemDTO struct {
	CardCode     string `json:"card_code"`
	CardPassword string `json:"card_password"`
}

type MemberCardRedeemResultDTO struct {
	Status        string `json:"status"`
	TargetType    string `json:"target_type"`
	TargetSummary string `json:"target_summary"`
	OrderNo       string `json:"order_no"`
	Message       string `json:"message"`
}

type MemberOrdersDTO struct {
	List       []MemberOrderListItemDTO  `json:"list"`
	Pagination MemberOrdersPaginationDTO `json:"pagination"`
}

type MemberOrderListItemDTO struct {
	OrderNo      string  `json:"order_no"`
	BusinessType string  `json:"business_type"`
	ProductType  string  `json:"product_type"`
	ProductTitle string  `json:"product_title,omitempty"`
	Status       string  `json:"status"`
	StatusLabel  string  `json:"status_label,omitempty"`
	Amount       float64 `json:"amount"`
	PayType      string  `json:"pay_type"`
	CreatedAt    string  `json:"created_at"`
}

type MemberOrdersPaginationDTO struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type MemberOrderDetailRequestDTO struct {
	OrderNo string `json:"order_no"`
}

type MemberOrderDetailDTO struct {
	OrderNo      string         `json:"order_no"`
	BusinessType string         `json:"business_type"`
	ProductType  string         `json:"product_type"`
	ProductID    string         `json:"product_id"`
	Status       string         `json:"status"`
	Amount       float64        `json:"amount"`
	PayType      string         `json:"pay_type"`
	PayTime      string         `json:"pay_time"`
	CreatedAt    string         `json:"created_at"`
	Snapshot     map[string]any `json:"snapshot"`
	ZibOrder     map[string]any `json:"zib_order"`
}

type MemberOrderStatusRequestDTO struct {
	OrderNo string `json:"order_no"`
}

type MemberOrderStatusDTO struct {
	OrderNo      string  `json:"order_no"`
	BusinessType string  `json:"business_type"`
	Status       string  `json:"status"`
	StatusLabel  string  `json:"status_label"`
	OrderPrice   float64 `json:"order_price"`
	PayPrice     float64 `json:"pay_price"`
	PayType      string  `json:"pay_type"`
	CreatedAt    string  `json:"created_at"`
	PaidAt       string  `json:"paid_at"`
}

type ResourceAccessCheckActorDTO struct {
	UserID         int64
	ExternalUserID string
	LoggedIn       bool
}

type PaidIdentityDTO struct {
	LoggedIn       bool
	UserID         int64
	ExternalUserID string
	SiteID         string
	BindingReady   bool
}

type ResourceAccessCheckRequestDTO struct {
	ResourceID    string `json:"resource_id"`
	ArticleID     string `json:"article_id,omitempty"`
	Abbrlink      string `json:"abbrlink,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

type ResourceRecordDTO struct {
	ResourceID    string
	HostType      string
	HostID        string
	Title         string
	Summary       string
	CoverURL      string
	ResourceType  string
	Status        string
	SaleEnabled   bool
	Price         float64
	OriginalPrice float64
	MemberFree    bool
	ResourceItems []ResourceAccessItemDTO
}

type ResourceAccessItemDTO struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url,omitempty"`
	ExtractionCode string `json:"extraction_code,omitempty"`
	Note           string `json:"note,omitempty"`
}

type AdminResourceListQueryDTO struct {
	Page     int
	PageSize int
	Query    string
	Status   string
}

type AdminOrderMappingListQueryDTO struct {
	SiteID      string `form:"site_id"`
	ZibOrderNum string `form:"zib_order_num"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type AdminOrderMappingSummaryDTO struct {
	Total           int    `json:"total"`
	SiteIDZeroCount int    `json:"site_id_zero_count"`
	LatestCreatedAt string `json:"latest_created_at"`
}

type AdminOrderMappingItemDTO struct {
	ZibOrderNum     string         `json:"zib_order_num"`
	ExternalUserID  string         `json:"external_user_id"`
	WpUserID        int            `json:"wp_user_id"`
	ProductType     string         `json:"product_type"`
	PostID          int            `json:"post_id"`
	ResourceID      int            `json:"resource_id"`
	CreatedAt       string         `json:"created_at"`
	StoredSiteID    string         `json:"stored_site_id"`
	ResolvedSiteID  string         `json:"resolved_site_id"`
	SnapshotSiteID  string         `json:"snapshot_site_id"`
	ContextSource   string         `json:"context_source"`
	IsDirty         bool           `json:"is_dirty"`
	RequestSnapshot map[string]any `json:"request_snapshot"`
}

type AdminOrderMappingPaginationDTO struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type AdminOrderMappingListDTO struct {
	Summary    AdminOrderMappingSummaryDTO    `json:"summary"`
	List       []AdminOrderMappingItemDTO     `json:"list"`
	Pagination AdminOrderMappingPaginationDTO `json:"pagination"`
}

type AdminMemberZoneListQueryDTO struct {
	Page     int
	PageSize int
	Query    string
	Status   string
}

type AdminMemberZoneListItemDTO struct {
	ContentID          string `json:"content_id"`
	Title              string `json:"title"`
	Slug               string `json:"slug"`
	Summary            string `json:"summary,omitempty"`
	Status             string `json:"status"`
	AccessLevel        string `json:"access_level"`
	SourceArticleID    string `json:"source_article_id,omitempty"`
	SourceArticleTitle string `json:"source_article_title,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	PublishedAt        string `json:"published_at,omitempty"`
}

type AdminMemberZoneListDTO struct {
	List     []AdminMemberZoneListItemDTO `json:"list"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type AdminMemberZoneDetailDTO struct {
	ContentID             string `json:"content_id"`
	Title                 string `json:"title"`
	Slug                  string `json:"slug"`
	Summary               string `json:"summary,omitempty"`
	CoverURL              string `json:"cover_url,omitempty"`
	ContentMD             string `json:"content_md"`
	ContentHTML           string `json:"content_html"`
	Status                string `json:"status"`
	AccessLevel           string `json:"access_level"`
	Sort                  int    `json:"sort"`
	SourceArticleID       string `json:"source_article_id,omitempty"`
	SourceArticleTitle    string `json:"source_article_title,omitempty"`
	SourceArticleAbbrlink string `json:"source_article_abbrlink,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
	PublishedAt           string `json:"published_at,omitempty"`
}

type MemberZoneListItemDTO struct {
	ContentID   string `json:"content_id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Summary     string `json:"summary,omitempty"`
	CoverURL    string `json:"cover_url,omitempty"`
	AccessLevel string `json:"access_level"`
	PublishedAt string `json:"published_at,omitempty"`
}

type MemberZoneMetaDTO struct {
	ContentID   string `json:"content_id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Summary     string `json:"summary,omitempty"`
	CoverURL    string `json:"cover_url,omitempty"`
	AccessLevel string `json:"access_level"`
	PublishedAt string `json:"published_at,omitempty"`
}

type MemberZoneContentDTO struct {
	ContentHTML string `json:"content_html"`
}

type MemberZoneAccessCheckRequestDTO struct {
	Slug string `json:"slug"`
}

type MemberZoneAccessCheckDTO struct {
	Allowed        bool              `json:"allowed"`
	Reason         string            `json:"reason"`
	RequiresLogin  bool              `json:"requires_login"`
	RequiresMember bool              `json:"requires_member"`
	RequiredLevel  string            `json:"required_level"`
	UserIsMember   bool              `json:"user_is_member"`
	UserLevel      string            `json:"user_level,omitempty"`
	MemberZoneMeta MemberZoneMetaDTO `json:"member_zone_meta"`
}

type AdminCardListQueryDTO struct {
	CardType string `json:"card_type,omitempty"`
	Status   string `json:"status,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

type AdminCardItemDTO struct {
	CardCode     string `json:"card_code"`
	CardPassword string `json:"card_password"`
	CardType     string `json:"card_type"`
	Status       string `json:"status"`
	Remark       string `json:"remark,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type AdminCardListDTO struct {
	List       []AdminCardItemDTO             `json:"list"`
	Pagination AdminOrderMappingPaginationDTO `json:"pagination"`
}

type AdminResourceListItemDTO struct {
	ResourceID    string  `json:"resource_id"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	SaleEnabled   bool    `json:"sale_enabled"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	MemberFree    bool    `json:"member_free"`
	HostType      string  `json:"host_type"`
	HostID        string  `json:"host_id"`
	HostTitle     string  `json:"host_title,omitempty"`
	OrderCount    int     `json:"order_count,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

type AdminResourceListDTO struct {
	List     []AdminResourceListItemDTO `json:"list"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

type AdminResourceItemDTO struct {
	ID             string `json:"id,omitempty"`
	Title          string `json:"title"`
	ItemType       string `json:"item_type"`
	URL            string `json:"url,omitempty"`
	ExtractionCode string `json:"extraction_code,omitempty"`
	Note           string `json:"note,omitempty"`
	Sort           int    `json:"sort"`
	Status         string `json:"status"`
}

type AdminResourceDetailDTO struct {
	ResourceID    string                 `json:"resource_id"`
	Title         string                 `json:"title"`
	Summary       string                 `json:"summary,omitempty"`
	CoverURL      string                 `json:"cover_url,omitempty"`
	Status        string                 `json:"status"`
	SaleEnabled   bool                   `json:"sale_enabled"`
	Price         float64                `json:"price"`
	OriginalPrice float64                `json:"original_price"`
	MemberFree    bool                   `json:"member_free"`
	HostType      string                 `json:"host_type,omitempty"`
	HostID        string                 `json:"host_id,omitempty"`
	HostTitle     string                 `json:"host_title,omitempty"`
	HostAbbrlink  string                 `json:"host_abbrlink,omitempty"`
	Items         []AdminResourceItemDTO `json:"items"`
}

type AdminArticleHostOptionDTO struct {
	ArticleID string `json:"article_id"`
	Title     string `json:"title"`
	Abbrlink  string `json:"abbrlink,omitempty"`
	Status    string `json:"status,omitempty"`
}

type AdminResourceBindArticleDTO struct {
	ArticleID string `json:"article_id"`
}

type ResourceOrderCreateDTO struct {
	UserID          int64
	ResourceID      string
	ResourceItemID  string
	BusinessOrderNo string
	ExternalOrderNo string
	Amount          float64
	Status          string
	Snapshot        map[string]any
	PaidAt          *time.Time
}

type ResourceOrderRecordDTO struct {
	UserID          int64
	ResourceID      string
	ResourceItemID  string
	BusinessOrderNo string
	ExternalOrderNo string
	Amount          float64
	Status          string
	Snapshot        map[string]any
	PaidAt          *time.Time
}

type ResourceAccessGrantCreateDTO struct {
	UserID         int64
	ResourceID     string
	ResourceItemID string
	GrantType      string
	SourceOrderNo  string
	Status         string
	GrantedAt      *time.Time
	ExpiredAt      *time.Time
}

type ResourceOrderPreviewDTO struct {
	Amount       float64 `json:"amount"`
	Subject      string  `json:"subject"`
	BusinessType string  `json:"business_type"`
	ResourceID   string  `json:"resource_id"`
}

type ResourceMetaDTO struct {
	ResourceID string `json:"resource_id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
}

type PaidAccessState string

const (
	PaidAccessStateAllowed          PaidAccessState = "allowed"
	PaidAccessStateLoginRequired    PaidAccessState = "login_required"
	PaidAccessStateMemberRequired   PaidAccessState = "member_required"
	PaidAccessStatePurchaseRequired PaidAccessState = "purchase_required"
	PaidAccessStateNotFound         PaidAccessState = "not_found"
	PaidAccessStateUnavailable      PaidAccessState = "unavailable"
)

type PaidAccessDecisionDTO struct {
	State   PaidAccessState
	Allowed bool
}

type ResourceAccessCheckDTO struct {
	AccessGranted    bool                    `json:"access_granted"`
	Reason           string                  `json:"reason"`
	RequiresLogin    bool                    `json:"requires_login"`
	RequiresPurchase bool                    `json:"requires_purchase"`
	MemberFree       bool                    `json:"member_free"`
	UserIsMember     bool                    `json:"user_is_member"`
	AlreadyPurchased bool                    `json:"already_purchased"`
	Price            float64                 `json:"price"`
	OriginalPrice    float64                 `json:"original_price"`
	BusinessType     string                  `json:"business_type"`
	BusinessPreview  ResourceOrderPreviewDTO `json:"business_order_preview"`
	Payable          bool                    `json:"payable"`
	ResourceMeta     ResourceMetaDTO         `json:"resource_meta"`
	ResourceItems    []ResourceAccessItemDTO `json:"resource_items,omitempty"`
}

type ResourcePurchaseOrderDTO struct {
	BusinessOrderNo string  `json:"business_order_no"`
	PayURL          string  `json:"pay_url"`
	Amount          float64 `json:"amount"`
	ResourceID      string  `json:"resource_id"`
}

type ResourcePurchaseOrderStatusRequestDTO struct {
	BusinessOrderNo string `json:"business_order_no"`
}

type ResourcePurchaseOrderStatusDTO struct {
	BusinessOrderNo string  `json:"business_order_no"`
	ExternalOrderNo string  `json:"external_order_no"`
	ResourceID      string  `json:"resource_id"`
	Status          string  `json:"status"`
	StatusLabel     string  `json:"status_label"`
	OrderPrice      float64 `json:"order_price"`
	PayPrice        float64 `json:"pay_price"`
	PayType         string  `json:"pay_type"`
	CreatedAt       string  `json:"created_at"`
	PaidAt          string  `json:"paid_at"`
}

type ResourceOrderPaymentDetailDTO struct {
	BusinessOrderNo string `json:"business_order_no"`
}

type ResourcePaymentDetailDTO struct {
	BusinessOrderNo string         `json:"business_order_no"`
	ResourceID      string         `json:"resource_id"`
	Amount          float64        `json:"amount"`
	PayType         string         `json:"pay_type"`
	PayChannel      string         `json:"pay_channel"`
	PayTime         string         `json:"pay_time,omitempty"`
	PayDetail       map[string]any `json:"pay_detail"`
}
