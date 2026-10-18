package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

type exchangeSnapshot struct {
	Source          string             `json:"source"`
	Base            string             `json:"base"`
	FetchedAt       time.Time          `json:"fetched_at"`
	Rates           map[string]float64 `json:"rates"`
	UpdateFrequency string             `json:"update_frequency"`
}
type FinanceExchangeHandler struct {
	mu         sync.Mutex
	cached     *exchangeSnapshot
	retryAfter time.Time
	client     *http.Client
}

func NewFinanceExchangeHandler() *FinanceExchangeHandler {
	return &FinanceExchangeHandler{client: &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (h *FinanceExchangeHandler) Rates(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("FREECURRENCY_API_KEY")
	if key == "" {
		writeJSON(w, 503, map[string]string{"error": "Kurs belum dihubungkan. Isi FREECURRENCY_API_KEY pada .env backend lalu restart server."})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cached != nil && time.Since(h.cached.FetchedAt) < 6*time.Hour {
		writeJSON(w, 200, h.cached)
		return
	}
	if time.Now().Before(h.retryAfter) {
		writeJSON(w, 503, map[string]string{"error": "Penyedia kurs belum tersedia. Coba lagi setelah jeda satu menit."})
		return
	}
	// Fixed destination and header auth: never expose API credentials to client/log URLs.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.freecurrencyapi.com/v1/latest?base_currency=USD", nil)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "Kurs belum tersedia"})
		return
	}
	req.Header.Set("apikey", key)
	h.retryAfter = time.Now().Add(time.Minute)
	response, err := h.client.Do(req)
	if err != nil {
		h.retryAfter = time.Now().Add(time.Minute)
		writeJSON(w, 502, map[string]string{"error": "Tidak dapat mengambil kurs dari penyedia. Periksa koneksi backend."})
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		h.retryAfter = time.Now().Add(time.Minute)
		writeJSON(w, 502, map[string]string{"error": "Penyedia kurs menolak permintaan. Periksa verifikasi email, API key, serta kuota akun."})
		return
	}
	var data struct {
		Data map[string]float64 `json:"data"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, response.Body, 1024*1024)).Decode(&data) != nil || data.Data["USD"] != 1 || data.Data["IDR"] <= 0 {
		writeJSON(w, 502, map[string]string{"error": "Format kurs penyedia tidak valid."})
		return
	}
	for k, v := range data.Data {
		if len(k) != 3 || v <= 0 || math.IsInf(v, 0) || math.IsNaN(v) {
			writeJSON(w, 502, map[string]string{"error": "Data kurs tidak valid."})
			return
		}
	}
	h.cached = &exchangeSnapshot{Source: "FreecurrencyAPI", Base: "USD", FetchedAt: time.Now().UTC(), Rates: data.Data, UpdateFrequency: "end_of_day"}
	writeJSON(w, 200, h.cached)
}
