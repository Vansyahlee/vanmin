package handlers

import (
	"errors"
	"furi-backend/internal/database"
	"furi-backend/internal/expenselogic"
	"furi-backend/internal/paymentlogic"
	"furi-backend/middleware"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinanceLedgerHandler struct{ pool *pgxpool.Pool }

func NewFinanceLedgerHandler(pool *pgxpool.Pool) *FinanceLedgerHandler {
	return &FinanceLedgerHandler{pool: pool}
}
func ledgerError(w http.ResponseWriter, err error) {
	if errors.Is(err, expenselogic.ErrDetails) {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	status := 500
	message := "Data belum dapat diproses. Pastikan migrasi Pengeluaran dan Notifikasi sudah dijalankan."
	if errors.Is(err, database.ErrExpenseConflict) {
		status = 409
		message = "Kunci pengeluaran sudah dipakai untuk data berbeda. Periksa riwayat sebelum mencatat ulang."
	}
	if errors.Is(err, pgx.ErrNoRows) {
		status = 404
		message = "Data tidak ditemukan"
	}
	writeJSON(w, status, map[string]string{"error": message})
}
func ledgerFilter(w http.ResponseWriter, r *http.Request) (database.LedgerFilter, bool) {
	q := r.URL.Query()
	f := database.LedgerFilter{Kind: q.Get("expense_kind"), Query: strings.TrimSpace(q.Get("query")), DateFrom: q.Get("date_from"), DateTo: q.Get("date_to"), Status: q.Get("status"), Category: strings.TrimSpace(q.Get("category")), Method: q.Get("method"), Account: q.Get("account"), Direction: q.Get("direction"), Read: q.Get("read"), Limit: 50}
	if f.Status == "" {
		f.Status = "ALL"
	}
	if f.Read == "" {
		f.Read = "ALL"
	}
	valid := (f.Kind == "" || f.Kind == "GENERAL" || f.Kind == "SALARY" || f.Kind == "SUPPLIER") && len(f.Query) <= 200 && len(f.Category) <= 100 && (f.Status == "ALL" || f.Status == "POSTED" || f.Status == "VOID") && (f.Method == "" || f.Method == "CASH" || f.Method == "BANK_TRANSFER" || f.Method == "MANUAL" || f.Method == "MIDTRANS") && (f.Account == "" || f.Account == "CASH" || f.Account == "BANK" || f.Account == "MIDTRANS" || f.Account == "LEGACY") && (f.Direction == "" || f.Direction == "IN" || f.Direction == "OUT") && (f.Read == "ALL" || f.Read == "READ" || f.Read == "UNREAD")
	for _, d := range []string{f.DateFrom, f.DateTo} {
		if d != "" {
			_, err := time.Parse("2006-01-02", d)
			valid = valid && err == nil
		}
	}
	if f.DateFrom != "" && f.DateTo != "" && f.DateFrom > f.DateTo {
		valid = false
	}
	for _, key := range []string{"limit", "offset"} {
		if v := q.Get(key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || (key == "limit" && (n < 1 || n > 200)) || n > 10000000 {
				valid = false
			} else if key == "limit" {
				f.Limit = n
			} else {
				f.Offset = n
			}
		}
	}
	if !valid {
		writeJSON(w, 400, map[string]string{"error": "Filter, tanggal, atau halaman tidak valid"})
		return f, false
	}
	claims, _ := middleware.GetClaims(r.Context())
	f.Actor = claims.UserID
	return f, true
}
func (h *FinanceLedgerHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	f, ok := ledgerFilter(w, r)
	if !ok {
		return
	}
	payload, err := database.ListExpenses(r.Context(), h.pool, f)
	if err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, payload)
}
func (h *FinanceLedgerHandler) Cash(w http.ResponseWriter, r *http.Request) {
	f, ok := ledgerFilter(w, r)
	if !ok {
		return
	}
	payload, err := database.CashLedger(r.Context(), h.pool, f)
	if err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, payload)
}
func (h *FinanceLedgerHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	f, ok := ledgerFilter(w, r)
	if !ok {
		return
	}
	payload, err := database.ListFinanceNotifications(r.Context(), h.pool, f)
	if err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, payload)
}
func (h *FinanceLedgerHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Kind        string               `json:"expense_kind"`
		Details     expenselogic.Details `json:"details"`
		RequestKey  string               `json:"request_key"`
		Date        string               `json:"expense_date"`
		Category    string               `json:"category"`
		Payee       string               `json:"payee"`
		Description string               `json:"description"`
		Amount      string               `json:"amount"`
		Method      string               `json:"method"`
		Reference   string               `json:"reference_number"`
		ProofURL    string               `json:"proof_url"`
	}
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	if b.Kind == "" {
		b.Kind = "GENERAL"
	}
	if _, err := expenselogic.Normalize(b.Kind, b.Details, b.Amount); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	b.Category = expenselogic.Category(b.Kind, b.Category)
	key, e := uuid.Parse(b.RequestKey)
	amount, a := paymentlogic.Cents(b.Amount)
	d, t := time.Parse("2006-01-02", b.Date)
	b.Category = strings.TrimSpace(b.Category)
	b.Payee = strings.TrimSpace(b.Payee)
	b.Description = strings.TrimSpace(b.Description)
	b.Reference = strings.TrimSpace(b.Reference)
	b.ProofURL = strings.TrimSpace(b.ProofURL)
	location := time.FixedZone("WIB", 7*60*60)
	today := time.Now().In(location).Format("2006-01-02")
	valid := e == nil && key != uuid.Nil && a == nil && amount > 0 && amount <= 9999999999999999 && t == nil && d.Year() >= 2000 && b.Date <= today && len(b.Category) > 0 && len(b.Category) <= 100 && len(b.Payee) > 0 && len(b.Payee) <= 200 && len(b.Description) >= 3 && len(b.Description) <= 2000 && len(b.Reference) <= 150 && len(b.ProofURL) <= 2000 && (b.Method == "CASH" || b.Method == "BANK_TRANSFER")
	if b.Method == "BANK_TRANSFER" && b.Reference == "" {
		valid = false
	}
	if b.ProofURL != "" {
		u, err := url.Parse(b.ProofURL)
		valid = valid && err == nil && u != nil
		if u != nil {
			valid = valid && u.Scheme == "https" && u.Host != "" && u.User == nil
		}
	}
	if !valid {
		writeJSON(w, 400, map[string]string{"error": "Lengkapi tanggal yang sudah terjadi, kategori, penerima, uraian, dan nominal positif (tanpa pemisah ribuan). Transfer wajib referensi; tautan bukti harus HTTPS."})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	id, err := database.CreateExpense(r.Context(), h.pool, database.ExpenseInput{ID: uuid.New(), RequestKey: key, Date: b.Date, Category: b.Category, Payee: b.Payee, Description: b.Description, Amount: b.Amount, Method: b.Method, Reference: b.Reference, ProofURL: b.ProofURL, Actor: claims.UserID, Kind: b.Kind, Details: b.Details})
	if err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id.String(), "message": "Pengeluaran tersimpan"})
}
func (h *FinanceLedgerHandler) VoidExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := paymentID(w, r)
	if !ok {
		return
	}
	var b struct {
		Reason string `json:"reason"`
	}
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	b.Reason = strings.TrimSpace(b.Reason)
	if len(b.Reason) < 5 || len(b.Reason) > 1000 {
		writeJSON(w, 400, map[string]string{"error": "Alasan pembatalan wajib 5–1000 karakter"})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if err := database.VoidExpense(r.Context(), h.pool, id, claims.UserID, b.Reason); err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Pengeluaran dibatalkan; pembalik kas tercatat"})
}
func (h *FinanceLedgerHandler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, 400, map[string]string{"error": "ID notifikasi tidak valid"})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if err = database.ReadFinanceNotifications(r.Context(), h.pool, claims.UserID, id, false); err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (h *FinanceLedgerHandler) ReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ThroughID string `json:"through_id"`
	}
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	id, err := strconv.ParseInt(b.ThroughID, 10, 64)
	if err != nil || id < 0 {
		writeJSON(w, 400, map[string]string{"error": "Batas notifikasi tidak valid"})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if err = database.ReadFinanceNotifications(r.Context(), h.pool, claims.UserID, id, true); err != nil {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
