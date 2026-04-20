package dp7575

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_BuildHeaders(t *testing.T) {
	client := NewClient(Config{
		BaseURL:   "https://api.example.com",
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	headers, body := client.BuildSignedHeaders(map[string]any{"external_user_id": "dp-user-001"})
	require.Equal(t, "application/json", headers.Get("Content-Type"))
	require.Equal(t, "yangguangzhan", headers.Get("YgzSiteId"))
	require.NotEmpty(t, headers.Get("YgzTimestamp"))
	require.NotEmpty(t, headers.Get("YgzNonce"))
	require.NotEmpty(t, headers.Get("YgzSignature"))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "dp-user-001", payload["external_user_id"])
}

func TestClient_MemberStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/member/status", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "dp-user-001", payload["external_user_id"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"is_member":true,"member_level":1,"member_level_name":"年度会员","member_expire_at":"2027-04-18T00:00:00Z"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	status, err := client.MemberStatus(context.Background(), MemberStatusRequest{ExternalUserID: "dp-user-001"})
	require.NoError(t, err)
	require.True(t, status.IsMember)
	require.Equal(t, "年度会员", status.MemberLevelName)
	require.Equal(t, "2027-04-18T00:00:00Z", status.MemberExpireAt)
}

func TestClient_MemberProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/member/profile", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "dp-user-001", payload["external_user_id"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"is_member":true,"member_level_name":"年度会员","member_expire_at":"2027-04-18T00:00:00Z","history_summary":{"latest_order_no":"VIP20260418001","latest_order_status":"paid","latest_order_amount":"99.00","latest_order_time":"2026-04-18 09:32:00"}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	profile, err := client.MemberProfile(context.Background(), MemberProfileRequest{ExternalUserID: "dp-user-001"})
	require.NoError(t, err)
	require.True(t, profile.IsMember)
	require.Equal(t, "年度会员", profile.MemberLevelName)
	require.Equal(t, "VIP20260418001", profile.HistorySummary.LatestOrderNo)
}

func TestClient_QueryUserMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/user/map/query", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"user_public_123","wp_user_id":7788,"is_mapped":true,"mapped":true}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.QueryUserMapping(context.Background(), UserMapQueryRequest{ExternalUserID: "user_public_123"})
	require.NoError(t, err)
	require.True(t, result.IsMapped)
	require.Equal(t, "yangguangzhan", result.SiteID)
	require.Equal(t, "user_public_123", result.ExternalUserID)
	require.Equal(t, int64(7788), result.WPUserID)
}

func TestClient_EnsureUserMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/user/map/ensure", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"user_public_123","wp_user_id":7788,"is_mapped":true,"mapped":true,"action":"created"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.EnsureUserMapping(context.Background(), UserMapEnsureRequest{ExternalUserID: "user_public_123"})
	require.NoError(t, err)
	require.Equal(t, "yangguangzhan", result.SiteID)
	require.Equal(t, "user_public_123", result.ExternalUserID)
	require.Equal(t, int64(7788), result.WPUserID)
	require.True(t, result.IsMapped)
	require.True(t, result.Mapped)
	require.Equal(t, "created", result.Action)
}

func TestClient_QueryUserMapping_ReturnsUnmappedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"user_public_123","wp_user_id":0,"is_mapped":false,"mapped":false}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.QueryUserMapping(context.Background(), UserMapQueryRequest{ExternalUserID: "user_public_123"})
	require.NoError(t, err)
	require.False(t, result.IsMapped)
	require.False(t, result.Mapped)
	require.Zero(t, result.WPUserID)
}

func TestClient_MemberProductsCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/member/products/catalog", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Empty(t, payload)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"","wp_user_id":0,"products":[{"product_type":"member","product_id":"vip_1_0_pay","product_key":"vip_1_0_pay","member_level":1,"member_level_name":"年度会员","title":"年度会员购买","description":"12个月","price":99,"original_price":199,"currency_type":"cash","action_type":"pay","availability":"available","time_value":"12","time_unit":"month","time_label":"12个月","meta":{"zib_product_id":"vip_1_0_pay","tag":"限时优惠"}}]}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.MemberProductsCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Products, 1)
	require.Equal(t, "vip_1_0_pay", result.Products[0].ProductID)
	require.Equal(t, 1, result.Products[0].MemberLevel)
	require.Equal(t, "pay", result.Products[0].ActionType)
	require.Equal(t, "限时优惠", result.Products[0].Meta.Tag)
}

func TestClient_MemberProductsCatalog_AcceptsNumericTimeValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"","wp_user_id":0,"products":[{"product_type":"member","product_id":"vip_2_1_upgrade","product_key":"vip_2_1_upgrade","member_level":2,"member_level_name":"钻石会员","title":"钻石会员升级","description":"1天","price":10,"original_price":10,"currency_type":"cash","action_type":"upgrade","availability":"available","time_value":1,"time_unit":"day","time_label":"1天","meta":{"zib_product_id":"vip_2_1_upgrade","tag":"站长推荐"}}]}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.MemberProductsCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Products, 1)
	require.Equal(t, FlexibleString("1"), result.Products[0].TimeValue)
}

func TestClient_CreateOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/order/create", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])
		require.Equal(t, "vip", payload["product_type"])
		require.Equal(t, "vip_1_0_pay", payload["product_id"])
		require.Equal(t, "wechat", payload["payment_method"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"business_type":"member","zib_order_num":"VIP20260418001","site_id":"yangguangzhan","external_user_id":"user_public_123","product_type":"member","post_id":0,"resource_id":0,"order_status":"pending","order_price":99,"pay_type":"wechat","pay_url":"https://pay.example.com/wechat/VIP20260418001","created_at":"2026-04-18 20:01:00"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.CreateOrder(context.Background(), OrderCreateRequest{
		ExternalUserID: "user_public_123",
		ProductType:    "vip",
		ProductID:      "vip_1_0_pay",
		PaymentMethod:  "wechat",
	})
	require.NoError(t, err)
	require.Equal(t, "VIP20260418001", result.ZibOrderNum)
	require.Equal(t, "pending", result.OrderStatus)
	require.Equal(t, "wechat", result.PayType)
	require.Equal(t, "https://pay.example.com/wechat/VIP20260418001", result.PayURL)
}

func TestClient_OrderPaymentDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/order/payment-detail", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])
		require.Equal(t, "VIP20260418001", payload["zib_order_num"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"order_num":"VIP20260418001","amount":99,"pay_type":"wechat","pay_channel":"wechat","pay_time":"","pay_detail":{"url_qrcode":"data:image/png;base64,WECHATQR","code_url":"weixin://wxpay/bizpayurl?pr=test-code"}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.OrderPaymentDetail(context.Background(), OrderPaymentDetailRequest{
		ExternalUserID: "user_public_123",
		ZibOrderNum:    "VIP20260418001",
	})
	require.NoError(t, err)
	require.Equal(t, "VIP20260418001", result.OrderNum)
	require.Equal(t, "wechat", result.PayType)
	require.Equal(t, "data:image/png;base64,WECHATQR", result.PayDetail["url_qrcode"])
	require.Equal(t, "weixin://wxpay/bizpayurl?pr=test-code", result.PayDetail["code_url"])
}

func TestClient_CardRedeemPrecheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/card/redeem/precheck", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])
		require.Equal(t, "VIP-CARD-001", payload["card_code"])
		require.Equal(t, "SECURE-001", payload["card_password"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"user_public_123","wp_user_id":7788,"redeemable":true,"target_type":"member","target_summary":"会员兑换：等级 1","risk_message":"","card_type":"vip_exchange","capabilities":{"vip_exchange_enabled":true,"points_exchange_enabled":false,"balance_charge_enabled":false}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.CardRedeemPrecheck(context.Background(), CardRedeemPrecheckRequest{
		ExternalUserID: "user_public_123",
		CardCode:       "VIP-CARD-001",
		CardPassword:   "SECURE-001",
	})
	require.NoError(t, err)
	require.True(t, result.Redeemable)
	require.Equal(t, "member", result.TargetType)
	require.Equal(t, "vip_exchange", result.CardType)
	require.True(t, result.Capabilities.VIPExchangeEnabled)
}

func TestClient_CardRedeemCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/card/redeem/create", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "user_public_123", payload["external_user_id"])
		require.Equal(t, "VIP-CARD-001", payload["card_code"])
		require.Equal(t, "SECURE-001", payload["card_password"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"site_id":"yangguangzhan","external_user_id":"user_public_123","wp_user_id":7788,"redeem_status":"success","target_type":"member","target_summary":"会员兑换：等级 1","order_num":"26041820010001","effect_summary":"兑换成功"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.CardRedeemCreate(context.Background(), CardRedeemCreateRequest{
		ExternalUserID: "user_public_123",
		CardCode:       "VIP-CARD-001",
		CardPassword:   "SECURE-001",
	})
	require.NoError(t, err)
	require.Equal(t, "success", result.RedeemStatus)
	require.Equal(t, "member", result.TargetType)
	require.Equal(t, "26041820010001", result.OrderNum)
	require.Equal(t, "兑换成功", result.EffectSummary)
}

func TestClient_OrderList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/order/list", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "dp-user-001", payload["external_user_id"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"list":[{"order_num":"VIP20260418001","business_type":"member","product_type":"vip","status":"paid","amount":99.00,"pay_type":"wechat","create_time":"2026-04-18 09:32:00"}],"pagination":{"page":1,"page_size":10,"total":1}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.OrderList(context.Background(), OrderListRequest{ExternalUserID: "dp-user-001"})
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	require.Equal(t, "member", result.List[0].BusinessType)
	require.Equal(t, "VIP20260418001", result.List[0].OrderNum)
	require.Equal(t, "paid", result.List[0].Status)
	require.Equal(t, 99.00, result.List[0].Amount)
}

func TestClient_OrderDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/order/detail", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "dp-user-001", payload["external_user_id"])
		require.Equal(t, "VIP20260418001", payload["zib_order_num"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"order_num":"VIP20260418001","business_type":"member","product_type":"vip","product_id":"vip_1_0_pay","status":"paid","amount":99.00,"pay_type":"wechat","pay_time":"2026-04-18 09:35:00","create_time":"2026-04-18 09:32:00","pay_detail":{},"snapshot":{"product":{"title":"年度会员"}},"zib_order":{}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.OrderDetail(context.Background(), OrderDetailRequest{ExternalUserID: "dp-user-001", ZibOrderNum: "VIP20260418001"})
	require.NoError(t, err)
	require.Equal(t, "VIP20260418001", result.OrderNum)
	require.Equal(t, "member", result.BusinessType)
	require.Equal(t, "paid", result.Status)
}

func TestClient_OrderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/order/status", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "dp-user-001", payload["external_user_id"])
		require.Equal(t, "VIP20260418001", payload["zib_order_num"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"business_type":"member","zib_order_num":"VIP20260418001","product_type":"vip","order_status":"paid","order_status_label":"已支付","order_price":99.00,"pay_price":99.00,"pay_type":"wechat","created_at":"2026-04-18 09:32:00","paid_at":"2026-04-18 09:35:00"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.OrderStatus(context.Background(), OrderStatusRequest{ExternalUserID: "dp-user-001", ZibOrderNum: "VIP20260418001"})
	require.NoError(t, err)
	require.Equal(t, "member", result.BusinessType)
	require.Equal(t, "VIP20260418001", result.ZibOrderNum)
	require.Equal(t, "paid", result.OrderStatus)
}

func TestClient_AdminOrderMappings_WhenUpstreamReturns404_IncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/admin/orders", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 Not Found"))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	_, err := client.AdminOrderMappings(context.Background(), AdminOrderMappingListRequest{Page: 1, PageSize: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status=404")
	require.Contains(t, err.Error(), "404 Not Found")
}

func TestClient_AdminCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/admin/cards", r.URL.Path)
		require.Equal(t, "yangguangzhan", r.Header.Get("YgzSiteId"))
		require.NotEmpty(t, r.Header.Get("YgzSignature"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "used", payload["status"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"list":[{"card_code":"23282285775808818733","card_password":"Uu5WKelcw4Z86SfdonM9Kz1l09y6FBPrR7v","card_type":"vip_exchange","status":"used","remark":"cardpass_20260418222640","created_at":"2026-04-18 22:27:00","updated_at":"2026-04-18 22:33:43"}],"pagination":{"page":1,"page_size":20,"total":1}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		SiteID:    "yangguangzhan",
		APISecret: "secret-123",
	})

	result, err := client.AdminCards(context.Background(), AdminCardListRequest{Status: "used", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	require.Equal(t, "23282285775808818733", result.List[0].CardCode)
	require.Equal(t, "used", result.List[0].Status)
}
