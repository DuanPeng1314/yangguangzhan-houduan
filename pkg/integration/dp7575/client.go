package dp7575

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	cfg        Config
	httpClient *http.Client
}

var ErrNotConfigured = errors.New("dp7575 client not configured")

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) BuildSignedHeaders(payload map[string]any) (http.Header, []byte) {
	body, _ := json.Marshal(payload)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := uuid.NewString()
	signature := md5Hex(c.cfg.SiteID + timestamp + nonce + string(body) + c.cfg.APISecret)

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("YgzSiteId", c.cfg.SiteID)
	headers.Set("YgzTimestamp", timestamp)
	headers.Set("YgzNonce", nonce)
	headers.Set("YgzSignature", signature)

	return headers, body
}

func (c *Client) doSignedPost(ctx context.Context, path string, payload map[string]any, out any, errorPrefix string) error {
	if strings.TrimSpace(c.cfg.BaseURL) == "" {
		return ErrNotConfigured
	}

	headers, body := c.BuildSignedHeaders(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header = headers

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		bodyText := strings.TrimSpace(string(respBody))
		if bodyText == "" {
			return fmt.Errorf("%s: status=%d", errorPrefix, resp.StatusCode)
		}
		return fmt.Errorf("%s: status=%d body=%s", errorPrefix, resp.StatusCode, bodyText)
	}

	var envelope responseEnvelope[json.RawMessage]
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("%s: code=%d message=%s", errorPrefix, envelope.Code, envelope.Message)
	}
	if out == nil {
		return nil
	}

	return json.Unmarshal(envelope.Data, out)
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (c *Client) MemberStatus(ctx context.Context, req MemberStatusRequest) (MemberStatusResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
	}

	var result MemberStatusResponse
	if err := c.doSignedPost(ctx, "/member/status", payload, &result, "dp7575 member status request failed"); err != nil {
		return MemberStatusResponse{}, err
	}

	return result, nil
}

func (c *Client) MemberProfile(ctx context.Context, req MemberProfileRequest) (MemberProfileResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
	}

	var result MemberProfileResponse
	if err := c.doSignedPost(ctx, "/member/profile", payload, &result, "dp7575 member profile request failed"); err != nil {
		return MemberProfileResponse{}, err
	}

	return result, nil
}

func (c *Client) QueryUserMapping(ctx context.Context, req UserMapQueryRequest) (UserMapQueryResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
	}

	var result UserMapQueryResponse
	if err := c.doSignedPost(ctx, "/user/map/query", payload, &result, "dp7575 user map query failed"); err != nil {
		return UserMapQueryResponse{}, err
	}

	return result, nil
}

func (c *Client) EnsureUserMapping(ctx context.Context, req UserMapEnsureRequest) (UserMapEnsureResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
	}

	var result UserMapEnsureResponse
	if err := c.doSignedPost(ctx, "/user/map/ensure", payload, &result, "dp7575 ensure user mapping failed"); err != nil {
		return UserMapEnsureResponse{}, err
	}

	return result, nil
}

func (c *Client) MemberProductsCatalog(ctx context.Context) (MemberProductsCatalogResponse, error) {
	var result MemberProductsCatalogResponse
	if err := c.doSignedPost(ctx, "/member/products/catalog", map[string]any{}, &result, "dp7575 member products catalog failed"); err != nil {
		return MemberProductsCatalogResponse{}, err
	}

	return result, nil
}

func (c *Client) CreateOrder(ctx context.Context, req OrderCreateRequest) (OrderCreateResponse, error) {
	if req.Amount > 0 {
		payload := map[string]any{
			"external_user_id":  req.ExternalUserID,
			"amount":            req.Amount,
			"payment_method":    req.PaymentMethod,
			"subject":           req.Subject,
			"business_order_no": req.BusinessOrderNo,
			"business_type":     req.BusinessType,
		}
		if len(req.Attach) > 0 {
			payload["attach"] = req.Attach
		}

		var result OrderCreateResponse
		if err := c.doSignedPost(ctx, "/order/custom-amount/create", payload, &result, "dp7575 create custom amount order failed"); err != nil {
			return OrderCreateResponse{}, err
		}

		return result, nil
	}

	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"product_type":     req.ProductType,
	}
	if req.ProductID != "" {
		payload["product_id"] = req.ProductID
	}
	if req.PaymentMethod != "" {
		payload["payment_method"] = req.PaymentMethod
	}

	var result OrderCreateResponse
	if err := c.doSignedPost(ctx, "/order/create", payload, &result, "dp7575 create order failed"); err != nil {
		return OrderCreateResponse{}, err
	}

	return result, nil
}

func (c *Client) OrderPaymentDetail(ctx context.Context, req OrderPaymentDetailRequest) (OrderPaymentDetailResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"zib_order_num":    req.ZibOrderNum,
	}

	var result OrderPaymentDetailResponse
	if err := c.doSignedPost(ctx, "/order/payment-detail", payload, &result, "dp7575 order payment detail failed"); err != nil {
		return OrderPaymentDetailResponse{}, err
	}

	return result, nil
}

func (c *Client) OrderList(ctx context.Context, req OrderListRequest) (OrderListResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
	}
	if req.BusinessType != "" {
		payload["business_type"] = req.BusinessType
	}
	if req.Page > 0 {
		payload["page"] = req.Page
	}
	if req.PageSize > 0 {
		payload["page_size"] = req.PageSize
	}

	var result OrderListResponse
	if err := c.doSignedPost(ctx, "/order/list", payload, &result, "dp7575 order list failed"); err != nil {
		return OrderListResponse{}, err
	}

	return result, nil
}

func (c *Client) AdminOrderMappings(ctx context.Context, req AdminOrderMappingListRequest) (AdminOrderMappingListResponse, error) {
	payload := map[string]any{}
	if req.SiteID != "" {
		payload["site_id"] = req.SiteID
	}
	if req.ZibOrderNum != "" {
		payload["zib_order_num"] = req.ZibOrderNum
	}
	if req.Page > 0 {
		payload["page"] = req.Page
	}
	if req.PageSize > 0 {
		payload["page_size"] = req.PageSize
	}

	var result AdminOrderMappingListResponse
	if err := c.doSignedPost(ctx, "/admin/orders", payload, &result, "dp7575 admin order mappings failed"); err != nil {
		return AdminOrderMappingListResponse{}, err
	}

	return result, nil
}

func (c *Client) AdminCards(ctx context.Context, req AdminCardListRequest) (AdminCardListResponse, error) {
	payload := map[string]any{
		"page":      req.Page,
		"page_size": req.PageSize,
	}
	if req.CardType != "" {
		payload["card_type"] = req.CardType
	}
	if req.Status != "" {
		payload["status"] = req.Status
	}

	var result AdminCardListResponse
	if err := c.doSignedPost(ctx, "/admin/cards", payload, &result, "dp7575 admin cards failed"); err != nil {
		return AdminCardListResponse{}, err
	}

	return result, nil
}

func (c *Client) OrderDetail(ctx context.Context, req OrderDetailRequest) (OrderDetailResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"zib_order_num":    req.ZibOrderNum,
	}

	var result OrderDetailResponse
	if err := c.doSignedPost(ctx, "/order/detail", payload, &result, "dp7575 order detail failed"); err != nil {
		return OrderDetailResponse{}, err
	}

	return result, nil
}

func (c *Client) OrderStatus(ctx context.Context, req OrderStatusRequest) (OrderStatusResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"zib_order_num":    req.ZibOrderNum,
	}

	var result OrderStatusResponse
	if err := c.doSignedPost(ctx, "/order/status", payload, &result, "dp7575 order status failed"); err != nil {
		return OrderStatusResponse{}, err
	}

	return result, nil
}

func (c *Client) CardRedeemPrecheck(ctx context.Context, req CardRedeemPrecheckRequest) (CardRedeemPrecheckResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"card_code":        req.CardCode,
		"card_password":    req.CardPassword,
	}

	var result CardRedeemPrecheckResponse
	if err := c.doSignedPost(ctx, "/card/redeem/precheck", payload, &result, "dp7575 card redeem precheck failed"); err != nil {
		return CardRedeemPrecheckResponse{}, err
	}

	return result, nil
}

func (c *Client) CardRedeemCreate(ctx context.Context, req CardRedeemCreateRequest) (CardRedeemCreateResponse, error) {
	payload := map[string]any{
		"external_user_id": req.ExternalUserID,
		"card_code":        req.CardCode,
		"card_password":    req.CardPassword,
	}

	var result CardRedeemCreateResponse
	if err := c.doSignedPost(ctx, "/card/redeem/create", payload, &result, "dp7575 card redeem create failed"); err != nil {
		return CardRedeemCreateResponse{}, err
	}

	return result, nil
}

func (c *Client) ConfigComplete() bool {
	return c.cfg.BaseURL != "" && c.cfg.SiteID != "" && c.cfg.APISecret != ""
}

func (c *Client) HealthProbe(ctx context.Context) (HealthProbeResult, error) {
	if !c.ConfigComplete() {
		return HealthProbeResult{Connected: false, SignatureValid: false, Detail: "还没有填写完整接入配置"}, nil
	}

	headers, body := c.BuildSignedHeaders(map[string]any{
		"external_user_id": "health_check_probe",
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.BaseURL, "/")+"/user/map/query", strings.NewReader(string(body)))
	if err != nil {
		return HealthProbeResult{}, err
	}
	httpReq.Header = headers

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return HealthProbeResult{Connected: false, SignatureValid: false, Detail: "当前无法连接极光库服务"}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return HealthProbeResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return HealthProbeResult{Connected: false, SignatureValid: false, Detail: fmt.Sprintf("远端返回状态码 %d", resp.StatusCode)}, nil
	}

	var envelope responseEnvelope[map[string]any]
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return HealthProbeResult{}, err
	}
	if envelope.Code == 1001 {
		return HealthProbeResult{Connected: true, SignatureValid: false, Detail: "鉴权信息校验失败，请检查站点标识和密钥"}, nil
	}

	return HealthProbeResult{Connected: true, SignatureValid: true, Detail: envelope.Message}, nil
}
