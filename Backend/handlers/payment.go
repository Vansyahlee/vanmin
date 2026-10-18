package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"furi-backend/internal/database"
	"furi-backend/internal/paymentlogic"
	"furi-backend/middleware"
	"furi-backend/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentHandler struct {
	pool     *pgxpool.Pool
	midtrans *services.MidtransClient
}

func NewPaymentHandler(pool *pgxpool.Pool) *PaymentHandler {
	return &PaymentHandler{pool: pool, midtrans: services.NewMidtransClient()}
}

func paymentError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "Pembayaran belum dapat diproses. Pastikan migrasi pembayaran sudah dijalankan."
	var postgres *pgconn.PgError
	switch {
	case errors.Is(err, database.ErrPaymentConflict):
		status = 409
		message = "Saldo tidak cukup, ada tautan Midtrans yang masih aktif, atau isi permintaan pembayaran berubah. Perbarui saldo dan periksa riwayat sebelum mengulang."
	case errors.Is(err, database.ErrPaymentInvoice), errors.Is(err, pgx.ErrNoRows):
		status = 404
		message = "Faktur tempo atau pembayaran tidak ditemukan."
	case errors.Is(err, paymentlogic.ErrAmount):
		status = 400
		message = paymentlogic.ErrAmount.Error()
	case errors.As(err, &postgres) && postgres.Code == "23505":
		status = 409
		message = "Permintaan atau referensi pembayaran sudah tercatat. Periksa riwayat pembayaran."
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func paymentID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "ID tidak valid"})
		return uuid.Nil, false
	}
	return id, true
}

func decodePaymentJSON(w http.ResponseWriter, r *http.Request, target any, strict bool) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	decoder := json.NewDecoder(r.Body)
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Isi permintaan JSON tidak valid"})
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeJSON(w, 400, map[string]string{"error": "Permintaan harus berisi satu objek JSON"})
		return false
	}
	return true
}

func (h *PaymentHandler) Config(w http.ResponseWriter, r *http.Request) {
	environment := "sandbox"
	if h.midtrans != nil {
		environment = h.midtrans.Environment
	}
	writeJSON(w, 200, map[string]any{"enabled": h.midtrans != nil, "environment": environment})
}

func (h *PaymentHandler) InvoiceDetails(w http.ResponseWriter, r *http.Request) {
	id, ok := paymentID(w, r)
	if !ok {
		return
	}
	payload, err := database.PaymentInvoiceDetails(r.Context(), h.pool, id)
	if err != nil {
		paymentError(w, err)
		return
	}
	var parsed struct {
		Invoice json.RawMessage `json:"invoice"`
	}
	if json.Unmarshal(payload, &parsed) != nil || string(parsed.Invoice) == "null" {
		paymentError(w, database.ErrPaymentInvoice)
		return
	}
	writeJSON(w, 200, payload)
}

func (h *PaymentHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := database.PaymentListFilter{Query: strings.TrimSpace(q.Get("query")), DateFrom: q.Get("date_from"), DateTo: q.Get("date_to"), Status: strings.ToUpper(q.Get("status")), Limit: 50}
	for _, value := range []string{f.DateFrom, f.DateTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				writeJSON(w, 400, map[string]string{"error": "Format tanggal harus YYYY-MM-DD"})
				return
			}
		}
	}
	if f.DateFrom != "" && f.DateTo != "" && f.DateFrom > f.DateTo {
		writeJSON(w, 400, map[string]string{"error": "Tanggal awal melebihi tanggal akhir"})
		return
	}
	for _, key := range []string{"limit", "offset"} {
		if value := q.Get(key); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || (key == "limit" && (n < 1 || n > 200)) || (key == "offset" && (n < 0 || n > 10000000)) {
				writeJSON(w, 400, map[string]string{"error": "Limit atau offset tidak valid"})
				return
			}
			if key == "limit" {
				f.Limit = n
			} else {
				f.Offset = n
			}
		}
	}
	if len(f.Query) > 200 || (f.Status != "" && f.Status != "ALL" && f.Status != "CONFIRMED" && f.Status != "VOID") {
		writeJSON(w, 400, map[string]string{"error": "Filter pembayaran tidak valid"})
		return
	}
	payload, err := database.ListInvoicePayments(r.Context(), h.pool, f)
	if err != nil {
		paymentError(w, err)
		return
	}
	writeJSON(w, 200, payload)
}

func (h *PaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InvoiceID  string `json:"invoice_id"`
		RequestKey string `json:"request_key"`
		Amount     string `json:"amount"`
		Source     string `json:"source"`
		Method     string `json:"method"`
		Reference  string `json:"reference_number"`
		Notes      string `json:"notes"`
		PaidAt     string `json:"paid_at"`
	}
	if !decodePaymentJSON(w, r, &body, true) {
		return
	}
	invoiceID, err := uuid.Parse(body.InvoiceID)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "ID faktur tidak valid"})
		return
	}
	requestKey, err := uuid.Parse(body.RequestKey)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "Kunci permintaan tidak valid"})
		return
	}
	amount, err := paymentlogic.Cents(body.Amount)
	if err != nil || amount <= 0 {
		paymentError(w, paymentlogic.ErrAmount)
		return
	}
	body.Source = strings.ToUpper(strings.TrimSpace(body.Source))
	body.Method = strings.ToUpper(strings.TrimSpace(body.Method))
	body.Reference = strings.TrimSpace(body.Reference)
	body.Notes = strings.TrimSpace(body.Notes)
	if len(body.Reference) > 150 || len(body.Notes) > 2000 {
		writeJSON(w, 400, map[string]string{"error": "Referensi maksimal 150 dan catatan maksimal 2000 karakter"})
		return
	}
	environment := "sandbox"
	if body.Source == "MIDTRANS" {
		if h.midtrans == nil {
			writeJSON(w, 503, map[string]string{"error": "Midtrans belum aktif. Isi MIDTRANS_ENABLED dan MIDTRANS_SERVER_KEY di backend."})
			return
		}
		if amount%100 != 0 || amount < 100000 || amount > 99999999999900 {
			writeJSON(w, 400, map[string]string{"error": "Midtrans membutuhkan nominal rupiah bulat, minimal Rp1.000"})
			return
		}
		body.Method = "MIDTRANS"
		body.Reference = ""
		environment = h.midtrans.Environment
	} else if body.Source != "MANUAL" || (body.Method != "CASH" && body.Method != "BANK_TRANSFER") {
		writeJSON(w, 400, map[string]string{"error": "Pilih pembayaran tunai, transfer bank, atau Midtrans"})
		return
	}
	if body.Source == "MANUAL" && body.Method == "BANK_TRANSFER" && body.Reference == "" {
		writeJSON(w, 400, map[string]string{"error": "Nomor referensi transfer wajib diisi"})
		return
	}
	paidAt := time.Now().UTC()
	if body.Source == "MANUAL" {
		paidAt, err = time.Parse(time.RFC3339, body.PaidAt)
		if err != nil || paidAt.After(time.Now().Add(5*time.Minute)) {
			writeJSON(w, 400, map[string]string{"error": "Tanggal pembayaran tidak valid atau masih di masa depan"})
			return
		}
	}
	claims, _ := middleware.GetClaims(r.Context())
	item, err := database.CreatePaymentIntent(r.Context(), h.pool, database.PaymentInput{
		InvoiceID: invoiceID, RequestKey: requestKey, Amount: body.Amount, Source: body.Source, Method: body.Method,
		Reference: body.Reference, Notes: body.Notes, PaidAt: paidAt, Actor: claims.UserID, Environment: environment,
	})
	if err != nil {
		paymentError(w, err)
		return
	}
	if item.Source == "MIDTRANS" && item.State == "CREATING" && item.Token == "" {
		item, err = h.ensureSnap(r, item.ID)
		if err != nil {
			writeJSON(w, 502, map[string]any{"error": err.Error(), "intent": map[string]string{"state": "CREATING"}, "message": "Reservasi dipertahankan. Buka riwayat Midtrans lalu gunakan Pulihkan Tautan / Cek Status."})
			return
		}
	}
	writeJSON(w, 200, map[string]any{"data": item})
}

func (h *PaymentHandler) gatewayIntent(w http.ResponseWriter, r *http.Request) (*database.PaymentIntent, bool) {
	id, ok := paymentID(w, r)
	if !ok {
		return nil, false
	}
	item, err := database.GetPaymentIntent(r.Context(), h.pool, id)
	if err != nil {
		paymentError(w, err)
		return nil, false
	}
	if h.midtrans == nil || item.Source != "MIDTRANS" || item.Environment != h.midtrans.Environment {
		writeJSON(w, 409, map[string]string{"error": "Aktifkan Midtrans dengan environment yang sama seperti transaksi ini"})
		return nil, false
	}
	return item, true
}

func (h *PaymentHandler) verifiedStatus(r *http.Request, item *database.PaymentIntent) (*services.MidtransStatus, error) {
	status, err := h.midtrans.Status(r.Context(), item.OrderID)
	if err != nil {
		return nil, err
	}
	if status.OrderID != item.OrderID || status.Currency != "IDR" || (h.midtrans.MerchantID != "" && status.MerchantID != h.midtrans.MerchantID) {
		return nil, database.ErrPaymentConflict
	}
	paidAt := time.Now().UTC()
	for _, value := range []string{status.SettlementTime, status.TransactionTime} {
		if parsed, e := time.ParseInLocation("2006-01-02 15:04:05", value, time.FixedZone("WIB", 7*3600)); e == nil {
			paidAt = parsed
			break
		}
	}
	err = database.ApplyGatewayPayment(r.Context(), h.pool, database.GatewayPaymentUpdate{
		OrderID: status.OrderID, Amount: status.GrossAmount, Status: status.TransactionStatus, Method: status.PaymentType,
		TransactionID: status.TransactionID, Paid: paymentlogic.IsPaid(status.TransactionStatus, status.FraudStatus, status.StatusCode),
		Refund: status.RefundAmount, PaidAt: paidAt,
	})
	return status, err
}

func (h *PaymentHandler) Sync(w http.ResponseWriter, r *http.Request) {
	item, ok := h.gatewayIntent(w, r)
	if !ok {
		return
	}
	_, err := h.verifiedStatus(r, item)
	if errors.Is(err, services.ErrMidtransNotFound) {
		writeJSON(w, 200, map[string]string{"message": "Belum ada transaksi pada metode pembayaran. Tautan Snap tetap aktif; saldo belum dikurangi."})
		return
	}
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "Status Midtrans belum dapat diverifikasi. Periksa koneksi dan environment lalu ulangi."})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Status telah diverifikasi ke Midtrans"})
}

func (h *PaymentHandler) Recover(w http.ResponseWriter, r *http.Request) {
	item, ok := h.gatewayIntent(w, r)
	if !ok {
		return
	}
	if item.State != "CREATING" || item.Token != "" {
		writeJSON(w, 200, map[string]any{"data": item})
		return
	}
	status, err := h.verifiedStatus(r, item)
	if err == nil && status != nil {
		writeJSON(w, 200, map[string]string{"message": "Transaksi sudah ada. Status diperbarui; cek riwayat."})
		return
	}
	if !errors.Is(err, services.ErrMidtransNotFound) {
		writeJSON(w, 502, map[string]string{"error": "Status belum dapat dipastikan; reservasi tetap dipertahankan"})
		return
	}
	if _, err = h.ensureSnap(r, item.ID); err != nil {
		writeJSON(w, 502, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Tautan dipulihkan dengan order ID yang sama. Perbarui riwayat."})
}

func (h *PaymentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	item, ok := h.gatewayIntent(w, r)
	if !ok {
		return
	}
	if item.State == "RELEASED" {
		writeJSON(w, 200, map[string]string{"message": "Sesi sudah dibatalkan"})
		return
	}
	if item.State != "CREATING" && item.State != "PENDING" {
		writeJSON(w, 409, map[string]string{"error": "Pembayaran sudah terkonfirmasi. Pembatalan ini hanya untuk tautan yang belum dibayar."})
		return
	}
	if item.Token == "" {
		writeJSON(w, 409, map[string]string{"error": "Pulihkan tautan dahulu, kemudian batalkan sesi"})
		return
	}
	status, err := h.verifiedStatus(r, item)
	if err != nil && !errors.Is(err, services.ErrMidtransNotFound) {
		writeJSON(w, 502, map[string]string{"error": "Status belum dapat diverifikasi; sesi belum dibatalkan"})
		return
	}
	if status != nil && paymentlogic.IsPaid(status.TransactionStatus, status.FraudStatus, status.StatusCode) {
		writeJSON(w, 409, map[string]string{"error": "Pembayaran ternyata sudah berhasil. Saldo telah diperbarui; tautan tidak dibatalkan."})
		return
	}
	if status != nil && status.TransactionStatus == "pending" {
		if err = h.midtrans.CancelOrder(r.Context(), item.OrderID); err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
	}
	if err = h.midtrans.CancelSnap(r.Context(), item.Token); err != nil {
		writeJSON(w, 502, map[string]string{"error": err.Error()})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if err = database.ReleasePaymentIntent(r.Context(), h.pool, item.ID, claims.UserID); err != nil {
		paymentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Sesi Midtrans dibatalkan; nominal tersedia kembali"})
}

func (h *PaymentHandler) Void(w http.ResponseWriter, r *http.Request) {
	id, ok := paymentID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodePaymentJSON(w, r, &body, true) {
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if len(body.Reason) < 5 || len(body.Reason) > 1000 {
		writeJSON(w, 400, map[string]string{"error": "Alasan pembatalan wajib diisi, 5–1000 karakter"})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if err := database.VoidManualPayment(r.Context(), h.pool, id, claims.UserID, body.Reason); err != nil {
		paymentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Catatan pembayaran manual dibatalkan; saldo dihitung ulang"})
}

func (h *PaymentHandler) Notification(w http.ResponseWriter, r *http.Request) {
	if h.midtrans == nil {
		writeJSON(w, 503, map[string]string{"error": "Midtrans tidak aktif"})
		return
	}
	var body services.MidtransStatus
	if !decodePaymentJSON(w, r, &body, false) {
		return
	}
	if !h.midtrans.VerifySignature(body) {
		writeJSON(w, 403, map[string]string{"error": "Signature tidak valid"})
		return
	}
	item, err := database.GetPaymentIntentByOrder(r.Context(), h.pool, body.OrderID)
	if err != nil {
		paymentError(w, err)
		return
	}
	if item.Environment != h.midtrans.Environment {
		writeJSON(w, 409, map[string]string{"error": "Environment berbeda"})
		return
	}
	// Never credit the webhook body or browser redirect; fetch the authoritative status.
	if _, err = h.verifiedStatus(r, item); err != nil {
		writeJSON(w, 503, map[string]string{"error": "Verifikasi belum berhasil; silakan kirim ulang notifikasi"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *PaymentHandler) ensureSnap(r *http.Request, id uuid.UUID) (*database.PaymentIntent, error) {
	return database.EnsurePaymentSnap(r.Context(), h.pool, id, func(item *database.PaymentIntent, customer string) (string, string, error) {
		if item.Environment != h.midtrans.Environment {
			return "", "", database.ErrPaymentConflict
		}
		cents, err := paymentlogic.Cents(item.Amount)
		if err != nil {
			return "", "", err
		}
		snap, err := h.midtrans.CreateSnap(r.Context(), item.OrderID, cents/100, customer, item.ExpiresAt)
		if err != nil {
			return "", "", err
		}
		return snap.Token, snap.RedirectURL, nil
	})
}

func (h *PaymentHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if h.midtrans == nil {
		writeJSON(w, 503, map[string]string{"error": "Midtrans belum aktif. Isi Server Key Sandbox dan MIDTRANS_ENABLED=true di backend, lalu restart server."})
		return
	}
	_, err := h.midtrans.Status(r.Context(), "FAS-CONNECTION-"+strings.ReplaceAll(uuid.NewString(), "-", ""))
	if errors.Is(err, services.ErrMidtransNotFound) {
		writeJSON(w, 200, map[string]string{"message": "Server Key diterima oleh API Midtrans " + h.midtrans.Environment + ". Koneksi backend berhasil; tidak ada transaksi pembayaran yang dibuat."})
		return
	}
	if err == nil {
		writeJSON(w, 502, map[string]string{"error": "Respons uji koneksi belum dapat dipastikan. Periksa Server Key dan environment."})
		return
	}
	writeJSON(w, 502, map[string]string{"error": err.Error()})
}
