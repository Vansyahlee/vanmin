package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrMidtransNotFound = errors.New("Midtrans belum memiliki transaksi untuk order ini")

type MidtransClient struct {
	ServerKey   string
	MerchantID  string
	Environment string
	SnapURL     string
	APIURL      string
	HTTP        *http.Client
}

type MidtransStatus struct {
	MerchantID        string `json:"merchant_id"`
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	Currency          string `json:"currency"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
	SettlementTime    string `json:"settlement_time"`
	TransactionTime   string `json:"transaction_time"`
	SignatureKey      string `json:"signature_key"`
	RefundAmount      string `json:"refund_amount"`
}

type SnapResult struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

func NewMidtransClient() *MidtransClient {
	if os.Getenv("MIDTRANS_ENABLED") != "true" || strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY")) == "" {
		return nil
	}
	client := &MidtransClient{
		ServerKey: strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY")), MerchantID: strings.TrimSpace(os.Getenv("MIDTRANS_MERCHANT_ID")), Environment: "sandbox",
		SnapURL: "https://app.sandbox.midtrans.com", APIURL: "https://api.sandbox.midtrans.com",
		HTTP: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
	}
	if os.Getenv("MIDTRANS_IS_PRODUCTION") == "true" {
		client.Environment = "production"
		client.SnapURL = "https://app.midtrans.com"
		client.APIURL = "https://api.midtrans.com"
	}
	return client
}

func (m *MidtransClient) request(ctx context.Context, method, endpoint string, body any, rawAuth bool, result any) error {
	var buffer bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buffer).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, &buffer)
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.ServerKey, "")
	if rawAuth {
		req.Header.Set("Authorization", m.ServerKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	response, err := m.HTTP.Do(req)
	if err != nil {
		return errors.New("Midtrans belum dapat dihubungi; coba cek status, jangan catat pembayaran yang sama secara manual")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	var status struct {
		StatusCode string `json:"status_code"`
	}
	_ = json.Unmarshal(data, &status)
	if status.StatusCode == "404" {
		return ErrMidtransNotFound
	}
	if result != nil {
		_ = json.Unmarshal(data, result)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Midtrans menolak permintaan (HTTP %d); periksa mode dan Server Key di backend", response.StatusCode)
	}
	if result != nil {
		return json.Unmarshal(data, result)
	}
	return nil
}

func (m *MidtransClient) CreateSnap(ctx context.Context, orderID string, rupiah int64, customer string, expires time.Time) (*SnapResult, error) {
	start := expires.Add(-30 * time.Minute).In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02 15:04:05 -0700")
	body := map[string]any{
		"transaction_details": map[string]any{"order_id": orderID, "gross_amount": rupiah},
		"customer_details":    map[string]string{"first_name": customer},
		"credit_card":         map[string]bool{"secure": true},
		"expiry":              map[string]any{"start_time": start, "unit": "minutes", "duration": 30},
		"page_expiry":         map[string]any{"unit": "minutes", "duration": 30},
	}
	var result SnapResult
	if err := m.request(ctx, "POST", m.SnapURL+"/snap/v1/transactions", body, false, &result); err != nil {
		return nil, err
	}
	target, err := url.Parse(result.RedirectURL)
	base, _ := url.Parse(m.SnapURL)
	if err != nil || result.Token == "" || target.Scheme != "https" || target.Host != base.Host || target.User != nil {
		return nil, errors.New("respons tautan Midtrans tidak valid")
	}
	return &result, nil
}

func (m *MidtransClient) Status(ctx context.Context, orderID string) (*MidtransStatus, error) {
	var result MidtransStatus
	if err := m.request(ctx, "GET", m.APIURL+"/v2/"+url.PathEscape(orderID)+"/status", nil, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (m *MidtransClient) CancelSnap(ctx context.Context, token string) error {
	var result struct {
		CanceledAt string   `json:"canceled_at"`
		Errors     []string `json:"error_messages"`
	}
	err := m.request(ctx, "POST", m.SnapURL+"/snap/v1/transactions/"+url.PathEscape(token)+"/cancel", map[string]any{}, true, &result)
	// A previously cancelled token is safe to retry after a network/DB failure.
	for _, message := range result.Errors {
		if message == "token already canceled" {
			return nil
		}
	}
	if err != nil {
		return err
	}
	if result.CanceledAt == "" {
		return errors.New("sesi Midtrans belum dapat dibatalkan; pembayaran mungkin sedang diproses")
	}
	return nil
}

func (m *MidtransClient) CancelOrder(ctx context.Context, orderID string) error {
	var result MidtransStatus
	if err := m.request(ctx, "POST", m.APIURL+"/v2/"+url.PathEscape(orderID)+"/cancel", map[string]any{}, false, &result); err != nil {
		return err
	}
	if result.TransactionStatus != "cancel" {
		return errors.New("transaksi Midtrans belum berhasil dibatalkan")
	}
	return nil
}

func (m *MidtransClient) VerifySignature(notification MidtransStatus) bool {
	sum := sha512.Sum512([]byte(notification.OrderID + notification.StatusCode + notification.GrossAmount + m.ServerKey))
	actual, err := hex.DecodeString(notification.SignatureKey)
	return err == nil && subtle.ConstantTimeCompare(sum[:], actual) == 1
}
