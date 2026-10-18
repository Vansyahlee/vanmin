package handlers

import (
	"encoding/json"
	"errors"
	"furi-backend/internal/database"
	"furi-backend/internal/paymentlogic"
	"furi-backend/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"strings"
	"time"
)

type FinanceReportingHandler struct{ pool *pgxpool.Pool }

func NewFinanceReportingHandler(pool *pgxpool.Pool) *FinanceReportingHandler {
	return &FinanceReportingHandler{pool: pool}
}
func reportError(w http.ResponseWriter, err error) {
	if errors.Is(err, database.ErrReportingRevision) {
		writeJSON(w, 409, map[string]string{"error": "Data telah berubah. Muat ulang sebelum menyimpan atau mengekspor."})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"error": "Faktur aktif tidak ditemukan."})
		return
	}
	writeJSON(w, 503, map[string]string{"error": "Laporan belum tersedia. Periksa migrasi finance_reporting dan koneksi database."})
}
func reportDates(r *http.Request) (string, string, bool) {
	from, to := r.URL.Query().Get("date_from"), r.URL.Query().Get("date_to")
	a, e1 := time.Parse("2006-01-02", from)
	b, e2 := time.Parse("2006-01-02", to)
	return from, to, e1 == nil && e2 == nil && !b.Before(a) && b.Sub(a) <= 3660*24*time.Hour
}
func (h *FinanceReportingHandler) Periods(w http.ResponseWriter, r *http.Request) {
	from, to, ok := reportDates(r)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "Isi rentang tanggal valid, maksimal 10 tahun."})
		return
	}
	data, err := database.FinancePeriodReport(r.Context(), h.pool, from, to)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, data)
}

type profitSettings struct {
	HPP          string `json:"hpp"`
	Salary       string `json:"salary_expense"`
	Opex         string `json:"operating_expense"`
	Tax          string `json:"income_tax"`
	Other        string `json:"other_income"`
	OwnerPercent string `json:"owner_percent"`
	Notes        string `json:"notes"`
}

func normalizeProfitSettings(s *profitSettings) bool {
	for _, p := range []*string{&s.HPP, &s.Salary, &s.Opex, &s.Tax, &s.Other} {
		v, err := paymentlogic.Cents(*p)
		if err != nil {
			return false
		}
		*p = paymentlogic.Decimal(v)
	}
	v, err := paymentlogic.Cents(s.OwnerPercent)
	if err != nil || v > 10000 {
		return false
	}
	s.OwnerPercent = paymentlogic.Decimal(v)
	s.Notes = strings.TrimSpace(s.Notes)
	return len(s.Notes) >= 5 && len(s.Notes) <= 2000
}
func (h *FinanceReportingHandler) SaveProfit(w http.ResponseWriter, r *http.Request) {
	period, err := time.Parse("2006-01", r.PathValue("period"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "Periode harus YYYY-MM."})
		return
	}
	var b struct {
		Settings profitSettings `json:"settings"`
		Revision int64          `json:"revision"`
	}
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	if b.Revision < 0 || !normalizeProfitSettings(&b.Settings) {
		writeJSON(w, 400, map[string]string{"error": "Isi seluruh nominal (0 jika nihil), persentase 0–100 dan catatan dasar perhitungan minimal 5 karakter."})
		return
	}
	raw, _ := json.Marshal(b.Settings)
	claims, _ := middleware.GetClaims(r.Context())
	err = database.SaveProfitPeriod(r.Context(), h.pool, period.Format("2006-01-02"), raw, b.Revision, claims.UserID)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"message": "Dasar laporan tersimpan dengan riwayat perubahan."})
}
func (h *FinanceReportingHandler) TaxSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "ID faktur tidak valid"})
		return
	}
	source, fp, err := database.FinanceTaxSource(r.Context(), h.pool, id)
	if err != nil {
		reportError(w, err)
		return
	}
	draft, err := database.GetTaxDraft(r.Context(), h.pool, id)
	if errors.Is(err, pgx.ErrNoRows) {
		draft = json.RawMessage(`{"document":null,"revision":0}`)
	} else if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"source": source, "fingerprint": fp, "draft": draft})
}
func (h *FinanceReportingHandler) SaveTax(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "ID faktur tidak valid"})
		return
	}
	var b struct {
		Document    json.RawMessage `json:"document"`
		Fingerprint string          `json:"fingerprint"`
		Revision    int64           `json:"revision"`
	}
	if !decodeReportingJSON(w, r, &b) {
		return
	}
	var doc map[string]any
	if len(b.Fingerprint) != 32 || b.Revision < 0 || len(b.Document) > 512000 || json.Unmarshal(b.Document, &doc) != nil || doc == nil || !validTaxDraftShape(doc) {
		writeJSON(w, 400, map[string]string{"error": "Draft pajak tidak valid atau terlalu besar."})
		return
	}
	// Stored as an internal draft only. Official validation/signing remains in Coretax.
	claims, _ := middleware.GetClaims(r.Context())
	err = database.SaveTaxDraft(r.Context(), h.pool, id, b.Fingerprint, b.Document, b.Revision, claims.UserID)
	if err != nil {
		reportError(w, err)
		return
	}
	h.TaxSource(w, r)
}
func (h *FinanceReportingHandler) History(w http.ResponseWriter, r *http.Request) {
	kind, ref := r.URL.Query().Get("kind"), r.URL.Query().Get("reference")
	if (kind != "PROFIT" && kind != "TAX") || len(ref) > 80 || ref == "" {
		writeJSON(w, 400, map[string]string{"error": "Referensi riwayat tidak valid"})
		return
	}
	data, err := database.ReportingHistory(r.Context(), h.pool, kind, ref)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}
func (h *FinanceReportingHandler) TaxRegister(w http.ResponseWriter, r *http.Request) {
	f, ok := ledgerFilter(w, r)
	if !ok {
		return
	}
	if f.Category != "" && f.Category != "PPN" && f.Category != "NON_PPN" {
		writeJSON(w, 400, map[string]string{"error": "Filter pajak tidak valid"})
		return
	}
	data, err := database.FinanceTaxRegister(r.Context(), h.pool, f)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, data)
}

func decodeReportingJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 600000)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		writeJSON(w, 400, map[string]string{"error": "Draft terlalu besar atau format JSON tidak valid."})
		return false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		writeJSON(w, 400, map[string]string{"error": "Hanya satu objek JSON diperbolehkan."})
		return false
	}
	return true
}
func validTaxDraftShape(doc map[string]any) bool {
	fields := []string{"seller_tin", "seller_nitku", "buyer_tin", "buyer_nitku", "buyer_name", "buyer_address", "buyer_email", "tax_date", "trx_code", "notes"}
	for _, key := range fields {
		v, ok := doc[key].(string)
		if !ok || len(v) > 2000 {
			return false
		}
	}
	if _, ok := doc["pkp_confirmed"].(bool); !ok {
		return false
	}
	items, ok := doc["items"].([]any)
	if !ok || len(items) == 0 || len(items) > 1000 {
		return false
	}
	for _, value := range items {
		item, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for _, key := range []string{"source_id", "name", "qty", "price", "discount", "base", "other_base", "vat_rate", "vat", "code", "unit"} {
			v, ok := item[key].(string)
			if !ok || len(v) > 500 {
				return false
			}
		}
	}
	return true
}
