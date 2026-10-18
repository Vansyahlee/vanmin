package services

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMidtransSignature(t *testing.T) {
	client := &MidtransClient{ServerKey: "test-server-key"}
	notification := MidtransStatus{OrderID: "FAS-test", StatusCode: "200", GrossAmount: "2000000.00"}
	sum := sha512.Sum512([]byte("FAS-test2002000000.00test-server-key"))
	notification.SignatureKey = hex.EncodeToString(sum[:])
	if !client.VerifySignature(notification) {
		t.Fatal("valid signature rejected")
	}
	notification.GrossAmount = "8000000.00"
	if client.VerifySignature(notification) {
		t.Fatal("tampered amount accepted")
	}
}

func TestMidtransStatusAndCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/FAS-notfound/status" {
			w.Write([]byte(`{"status_code":"404"}`))
			return
		}
		if r.URL.Path == "/snap/v1/transactions/cancelled/cancel" {
			if r.Header.Get("Authorization") != "test-key" {
				t.Error("Snap cancellation auth")
			}
			w.WriteHeader(400)
			w.Write([]byte(`{"error_messages":["token already canceled"]}`))
			return
		}
		user, password, ok := r.BasicAuth()
		if !ok || user != "test-key" || password != "" {
			t.Error("Basic auth invalid")
		}
		json.NewEncoder(w).Encode(MidtransStatus{OrderID: "FAS-paid", StatusCode: "200", Currency: "IDR", GrossAmount: "2000000.00", TransactionStatus: "settlement"})
	}))
	defer server.Close()
	client := &MidtransClient{ServerKey: "test-key", APIURL: server.URL, SnapURL: server.URL, HTTP: &http.Client{Timeout: time.Second}}
	status, err := client.Status(context.Background(), "FAS-paid")
	if err != nil || status.GrossAmount != "2000000.00" {
		t.Fatalf("%+v %v", status, err)
	}
	if _, err = client.Status(context.Background(), "FAS-notfound"); err != ErrMidtransNotFound {
		t.Fatal("404 must remain unconfirmed")
	}
	if err = client.CancelSnap(context.Background(), "cancelled"); err != nil {
		t.Fatal("cancel replay", err)
	}
}

func TestCreateSnapUsesInstallmentAmount(t *testing.T) {
	var host string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Transaction struct {
				Order  string `json:"order_id"`
				Amount int64  `json:"gross_amount"`
			} `json:"transaction_details"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Transaction.Order != "FAS-installment" || request.Transaction.Amount != 2000000 {
			t.Error("wrong installment amount")
		}
		user, _, ok := r.BasicAuth()
		if !ok || user != "test-key" {
			t.Error("missing auth")
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(SnapResult{Token: "token-test", RedirectURL: host + "/snap/v2/vtweb/token-test"})
	}))
	defer server.Close()
	host = server.URL
	client := &MidtransClient{ServerKey: "test-key", SnapURL: server.URL, HTTP: server.Client()}
	result, err := client.CreateSnap(context.Background(), "FAS-installment", 2000000, "Toko Uji", time.Now().Add(30*time.Minute))
	if err != nil || result.Token != "token-test" {
		t.Fatal(result, err)
	}
}

func TestMidtransSandboxConfiguration(t *testing.T) {
	t.Setenv("MIDTRANS_ENABLED", "true")
	t.Setenv("MIDTRANS_IS_PRODUCTION", "false")
	t.Setenv("MIDTRANS_SERVER_KEY", "test-sandbox-key")
	t.Setenv("MIDTRANS_MERCHANT_ID", "G-TEST")
	client := NewMidtransClient()
	if client == nil || client.Environment != "sandbox" || client.APIURL != "https://api.sandbox.midtrans.com" || client.MerchantID != "G-TEST" {
		t.Fatal("sandbox configuration not applied")
	}
	t.Setenv("MIDTRANS_ENABLED", "false")
	if NewMidtransClient() != nil {
		t.Fatal("disabled gateway should not initialize")
	}
}
