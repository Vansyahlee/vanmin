package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type reportRoundTrip func(*http.Request) (*http.Response, error)

func (f reportRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestReportingProfitValidation(t *testing.T) {
	good := profitSettings{HPP: "3000000", Salary: "2000000", Opex: "500000.15", Tax: "250000", Other: "0", OwnerPercent: "25", Notes: "Dasar biaya periode telah diperiksa"}
	if !normalizeProfitSettings(&good) || good.Opex != "500000.15" || good.HPP != "3000000.00" {
		t.Fatal("valid exact amounts rejected")
	}
	for _, bad := range []profitSettings{{HPP: ""}, {HPP: "-1"}, {HPP: "0", Salary: "0", Opex: "0", Tax: "0", Other: "0", OwnerPercent: "100.01", Notes: "Valid note"}} {
		if normalizeProfitSettings(&bad) {
			t.Fatal("invalid inputs accepted")
		}
	}
}
func TestReportingExchangeCachesAndProtectsKey(t *testing.T) {
	t.Setenv("FREECURRENCY_API_KEY", "test-private-key")
	count := 0
	h := NewFinanceExchangeHandler()
	h.client.Transport = reportRoundTrip(func(r *http.Request) (*http.Response, error) {
		count++
		if strings.Contains(r.URL.String(), "test-private") || r.Header.Get("apikey") != "test-private-key" {
			t.Fatal("key placement invalid")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":{"USD":1,"IDR":16000,"EUR":0.9}}`)), Header: make(http.Header)}, nil
	})
	for range 2 {
		w := httptest.NewRecorder()
		h.Rates(w, httptest.NewRequest("GET", "/finance/exchange-rates", nil))
		if w.Code != 200 || strings.Contains(w.Body.String(), "test-private") {
			t.Fatal(w.Code, w.Body.String())
		}
		var body exchangeSnapshot
		if json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Rates["IDR"] != 16000 || body.UpdateFrequency != "end_of_day" {
			t.Fatal("rate payload invalid")
		}
	}
	if count != 1 {
		t.Fatal("cache did not prevent second provider call", count)
	}
}
func TestReportingExchangeFailureNoFalseRates(t *testing.T) {
	t.Setenv("FREECURRENCY_API_KEY", "")
	h := NewFinanceExchangeHandler()
	w := httptest.NewRecorder()
	h.Rates(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	t.Setenv("FREECURRENCY_API_KEY", "unit-key")
	h.client.Transport = reportRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("secret-provider-error")), Header: make(http.Header)}, nil
	})
	w = httptest.NewRecorder()
	h.Rates(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 502 || strings.Contains(w.Body.String(), "secret-provider-error") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestReportingInvalidDatesBeforeDatabase(t *testing.T) {
	h := NewFinanceReportingHandler(nil)
	for _, q := range []string{"date_from=2026-02-30&date_to=2026-03-01", "date_from=2026-10-20&date_to=2026-10-01", "date_from=2000-01-01&date_to=2026-01-01"} {
		w := httptest.NewRecorder()
		h.Periods(w, httptest.NewRequest("GET", "/?"+q, nil))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
}
func TestReportingDraftShapeRejectsUnexpectedTypes(t *testing.T) {
	if validTaxDraftShape(map[string]any{"items": false}) {
		t.Fatal("invalid draft accepted")
	}
}

func TestReportingLargeDraftDecoder(t *testing.T) {
	// Product-rich tax drafts exceed the payment endpoint's 16KB cap.
	body := `{"document":{"notes":"` + strings.Repeat("x", 20000) + `"}}`
	var target struct {
		Document json.RawMessage `json:"document"`
	}
	w := httptest.NewRecorder()
	if !decodeReportingJSON(w, httptest.NewRequest("PUT", "/", strings.NewReader(body)), &target) {
		t.Fatal("legitimate-sized reporting body rejected")
	}
	w = httptest.NewRecorder()
	if decodeReportingJSON(w, httptest.NewRequest("PUT", "/", strings.NewReader(`{} {}`)), &target) {
		t.Fatal("multiple JSON values accepted")
	}
}
