package dp7575

import "encoding/json"

type FlexibleString string

func (v *FlexibleString) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = FlexibleString(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*v = FlexibleString(number.String())
		return nil
	}

	return json.Unmarshal(data, &text)
}

type Config struct {
	BaseURL   string
	SiteID    string
	APISecret string
}

type MemberStatusRequest struct {
	ExternalUserID string `json:"external_user_id"`
	SiteID         string `json:"site_id,omitempty"`
}

type MemberProfileRequest struct {
	ExternalUserID string `json:"external_user_id"`
	SiteID         string `json:"site_id,omitempty"`
}

type UserMapQueryRequest struct {
	ExternalUserID string `json:"external_user_id"`
}

type UserMapQueryResponse struct {
	SiteID         string `json:"site_id"`
	ExternalUserID string `json:"external_user_id"`
	WPUserID       int64  `json:"wp_user_id"`
	IsMapped       bool   `json:"is_mapped"`
	Mapped         bool   `json:"mapped"`
}

type UserMapEnsureRequest struct {
	ExternalUserID string `json:"external_user_id"`
}

type UserMapEnsureResponse struct {
	SiteID         string `json:"site_id"`
	ExternalUserID string `json:"external_user_id"`
	WPUserID       int64  `json:"wp_user_id"`
	IsMapped       bool   `json:"is_mapped"`
	Mapped         bool   `json:"mapped"`
	Action         string `json:"action"`
}

type MemberStatusResponse struct {
	IsMember        bool            `json:"is_member"`
	MemberLevel     json.RawMessage `json:"member_level"`
	MemberLevelName string          `json:"member_level_name"`
	MemberExpireAt  string          `json:"member_expire_at"`
}

type MemberHistorySummary struct {
	LatestOrderNo     string `json:"latest_order_no"`
	LatestOrderStatus string `json:"latest_order_status"`
	LatestOrderAmount string `json:"latest_order_amount"`
	LatestOrderTime   string `json:"latest_order_time"`
}

type MemberProfileResponse struct {
	IsMember        bool                 `json:"is_member"`
	MemberLevel     json.RawMessage      `json:"member_level"`
	MemberLevelName string               `json:"member_level_name"`
	MemberExpireAt  string               `json:"member_expire_at"`
	HistorySummary  MemberHistorySummary `json:"history_summary"`
}

type MemberProductMeta struct {
	ZibProductID string `json:"zib_product_id"`
	Tag          string `json:"tag"`
}

type MemberProduct struct {
	ProductType     string            `json:"product_type"`
	ProductID       string            `json:"product_id"`
	ProductKey      string            `json:"product_key"`
	MemberLevel     int               `json:"member_level"`
	MemberLevelName string            `json:"member_level_name"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Price           float64           `json:"price"`
	OriginalPrice   float64           `json:"original_price"`
	CurrencyType    string            `json:"currency_type"`
	ActionType      string            `json:"action_type"`
	Availability    string            `json:"availability"`
	TimeValue       FlexibleString    `json:"time_value"`
	TimeUnit        string            `json:"time_unit"`
	TimeLabel       string            `json:"time_label"`
	Meta            MemberProductMeta `json:"meta"`
}

type MemberProductsCatalogResponse struct {
	SiteID         string          `json:"site_id"`
	ExternalUserID string          `json:"external_user_id"`
	WPUserID       int64           `json:"wp_user_id"`
	Products       []MemberProduct `json:"products"`
}

type OrderCreateRequest struct {
	ExternalUserID string `json:"external_user_id"`
	ProductType    string `json:"product_type"`
	ProductID      string `json:"product_id,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
	BusinessType   string         `json:"business_type,omitempty"`
	BusinessOrderNo string        `json:"business_order_no,omitempty"`
	Subject        string         `json:"subject,omitempty"`
	Amount         float64        `json:"amount,omitempty"`
	Attach         map[string]any `json:"attach,omitempty"`
}

type OrderCreateResponse struct {
	BusinessType   string  `json:"business_type"`
	ZibOrderNum    string  `json:"zib_order_num"`
	SiteID         string  `json:"site_id"`
	ExternalUserID string  `json:"external_user_id"`
	ProductType    string  `json:"product_type"`
	PostID         int64   `json:"post_id"`
	ResourceID     int64   `json:"resource_id"`
	OrderStatus    string  `json:"order_status"`
	OrderPrice     float64 `json:"order_price"`
	PayType        string  `json:"pay_type"`
	PayURL         string  `json:"pay_url"`
	CreatedAt      string  `json:"created_at"`
}

type OrderPaymentDetailRequest struct {
	ExternalUserID string `json:"external_user_id"`
	ZibOrderNum    string `json:"zib_order_num"`
}

type OrderPaymentDetailResponse struct {
	OrderNum   string         `json:"order_num"`
	Amount     float64        `json:"amount"`
	PayType    string         `json:"pay_type"`
	PayChannel string         `json:"pay_channel"`
	PayTime    string         `json:"pay_time"`
	PayDetail  map[string]any `json:"pay_detail"`
}

type CardRedeemPrecheckRequest struct {
	ExternalUserID string `json:"external_user_id"`
	CardCode       string `json:"card_code"`
	CardPassword   string `json:"card_password"`
}

type CardRedeemCapabilities struct {
	VIPExchangeEnabled    bool `json:"vip_exchange_enabled"`
	PointsExchangeEnabled bool `json:"points_exchange_enabled"`
	BalanceChargeEnabled  bool `json:"balance_charge_enabled"`
}

type CardRedeemPrecheckResponse struct {
	SiteID         string                 `json:"site_id"`
	ExternalUserID string                 `json:"external_user_id"`
	WPUserID       int64                  `json:"wp_user_id"`
	Redeemable     bool                   `json:"redeemable"`
	TargetType     string                 `json:"target_type"`
	TargetSummary  string                 `json:"target_summary"`
	RiskMessage    string                 `json:"risk_message"`
	CardType       string                 `json:"card_type"`
	Capabilities   CardRedeemCapabilities `json:"capabilities"`
}

type CardRedeemCreateRequest struct {
	ExternalUserID string `json:"external_user_id"`
	CardCode       string `json:"card_code"`
	CardPassword   string `json:"card_password"`
}

type CardRedeemCreateResponse struct {
	SiteID         string `json:"site_id"`
	ExternalUserID string `json:"external_user_id"`
	WPUserID       int64  `json:"wp_user_id"`
	RedeemStatus   string `json:"redeem_status"`
	TargetType     string `json:"target_type"`
	TargetSummary  string `json:"target_summary"`
	OrderNum       string `json:"order_num"`
	EffectSummary  string `json:"effect_summary"`
}

type AdminCardListRequest struct {
	CardType string `json:"card_type,omitempty"`
	Status   string `json:"status,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

type AdminCardItem struct {
	CardCode     string `json:"card_code"`
	CardPassword string `json:"card_password"`
	CardType     string `json:"card_type"`
	Status       string `json:"status"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type AdminCardListResponse struct {
	List       []AdminCardItem     `json:"list"`
	Pagination OrderListPagination `json:"pagination"`
}

type OrderListRequest struct {
	ExternalUserID string `json:"external_user_id"`
	BusinessType   string `json:"business_type,omitempty"`
	Page           int    `json:"page,omitempty"`
	PageSize       int    `json:"page_size,omitempty"`
}

type OrderListItem struct {
	OrderNum     string  `json:"order_num"`
	BusinessType string  `json:"business_type"`
	ProductType  string  `json:"product_type"`
	Status       string  `json:"status"`
	Amount       float64 `json:"amount"`
	PayType      string  `json:"pay_type"`
	CreateTime   string  `json:"create_time"`
}

type OrderListPagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type OrderListResponse struct {
	List       []OrderListItem     `json:"list"`
	Pagination OrderListPagination `json:"pagination"`
}

type AdminOrderMappingListRequest struct {
	SiteID      string `json:"site_id,omitempty"`
	ZibOrderNum string `json:"zib_order_num,omitempty"`
	Page        int    `json:"page,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
}

type AdminOrderMappingSummary struct {
	Total           int    `json:"total"`
	SiteIDZeroCount int    `json:"site_id_zero_count"`
	LatestCreatedAt string `json:"latest_created_at"`
}

type AdminOrderMappingItem struct {
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

type AdminOrderMappingListResponse struct {
	Summary    AdminOrderMappingSummary `json:"summary"`
	List       []AdminOrderMappingItem  `json:"list"`
	Pagination OrderListPagination      `json:"pagination"`
}

type OrderDetailRequest struct {
	ExternalUserID string `json:"external_user_id"`
	ZibOrderNum    string `json:"zib_order_num"`
}

type OrderDetailResponse struct {
	OrderNum     string         `json:"order_num"`
	BusinessType string         `json:"business_type"`
	ProductType  string         `json:"product_type"`
	ProductID    string         `json:"product_id"`
	Status       string         `json:"status"`
	Amount       float64        `json:"amount"`
	PayType      string         `json:"pay_type"`
	PayTime      string         `json:"pay_time"`
	CreateTime   string         `json:"create_time"`
	PayDetail    map[string]any `json:"pay_detail"`
	Snapshot     map[string]any `json:"snapshot"`
	ZibOrder     map[string]any `json:"zib_order"`
}

type OrderStatusRequest struct {
	ExternalUserID string `json:"external_user_id"`
	ZibOrderNum    string `json:"zib_order_num"`
}

type OrderStatusResponse struct {
	BusinessType     string  `json:"business_type"`
	ZibOrderNum      string  `json:"zib_order_num"`
	ProductType      string  `json:"product_type"`
	OrderStatus      string  `json:"order_status"`
	OrderStatusLabel string  `json:"order_status_label"`
	OrderPrice       float64 `json:"order_price"`
	PayPrice         float64 `json:"pay_price"`
	PayType          string  `json:"pay_type"`
	CreatedAt        string  `json:"created_at"`
	PaidAt           string  `json:"paid_at"`
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type HealthProbeResult struct {
	Connected      bool
	SignatureValid bool
	Detail         string
}
