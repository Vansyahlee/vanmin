package handlers

import (
	"encoding/json"
	"errors"
	"furi-backend/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
)

type CompanyHandler struct{ pool *pgxpool.Pool }

func NewCompanyHandler(pool *pgxpool.Pool) *CompanyHandler { return &CompanyHandler{pool: pool} }
func (h *CompanyHandler) canEdit(r *http.Request) bool {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		return false
	}
	owner := strings.TrimSpace(os.Getenv("COMPANY_OWNER_USER_ID"))
	if owner == "" || claims.UserID != owner {
		return false
	}
	var enabled bool
	err := h.pool.QueryRow(r.Context(), `SELECT is_active AND role='admin' FROM users WHERE id::text=$1`, owner).Scan(&enabled)
	return err == nil && enabled
}
func (h *CompanyHandler) Get(w http.ResponseWriter, r *http.Request) {
	var profile json.RawMessage
	err := h.pool.QueryRow(r.Context(), `SELECT (TO_JSONB(c)-'id'-'updated_by') FROM company_profile c WHERE id=TRUE`).Scan(&profile)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "Profil perusahaan belum tersedia. Jalankan migrasi profil perusahaan."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": profile, "can_edit": h.canEdit(r)})
}
func (h *CompanyHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		writeJSON(w, 403, map[string]string{"error": "Hanya akun owner yang ditetapkan di backend dapat mengubah profil perusahaan."})
		return
	}
	var b struct {
		Name     string `json:"name"`
		Address  string `json:"address"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		TaxID    string `json:"tax_id"`
		TaxNITKU string `json:"tax_nitku"`
		License  string `json:"license_number"`
		Maps     string `json:"maps_url"`
		Revision int64  `json:"revision"`
	}
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	b.Name = strings.TrimSpace(b.Name)
	b.Address = strings.TrimSpace(b.Address)
	b.Phone = strings.TrimSpace(b.Phone)
	b.Email = strings.TrimSpace(b.Email)
	b.TaxID = strings.TrimSpace(b.TaxID)
	b.TaxNITKU = strings.TrimSpace(b.TaxNITKU)
	if b.TaxNITKU != "" {
		tin := strings.NewReplacer(".", "", "-", "", " ", "").Replace(b.TaxID)
		validNITKU := len(tin) == 16 && len(b.TaxNITKU) == 22 && strings.HasPrefix(b.TaxNITKU, tin)
		for _, c := range b.TaxNITKU {
			validNITKU = validNITKU && c >= '0' && c <= '9'
		}
		if !validNITKU {
			writeJSON(w, 400, map[string]string{"error": "NITKU opsional harus 22 digit sesuai NPWP perusahaan 16 digit yang terdaftar."})
			return
		}
	}
	b.License = strings.TrimSpace(b.License)
	b.Maps = strings.TrimSpace(b.Maps)
	valid := len(b.Name) > 0 && len(b.Name) <= 200 && len(b.Address) > 0 && len(b.Address) <= 1000 && len(b.Phone) <= 80 && len(b.Email) <= 254 && len(b.TaxID) <= 80 && len(b.License) <= 100 && len(b.Maps) <= 2000 && b.Revision > 0
	if b.Email != "" {
		parsed, err := mail.ParseAddress(b.Email)
		valid = valid && err == nil
		if parsed != nil {
			valid = valid && parsed.Address == b.Email
		}
	}
	if b.Maps != "" {
		u, err := url.Parse(b.Maps)
		valid = valid && err == nil && u != nil
		if u != nil {
			valid = valid && u.Scheme == "https" && u.Host != "" && u.User == nil
		}
	}
	if !valid {
		writeJSON(w, 400, map[string]string{"error": "Nama dan alamat wajib. Periksa panjang isian, format email, dan tautan HTTPS."})
		return
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		ledgerError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var before, after json.RawMessage
	var revision int64
	err = tx.QueryRow(r.Context(), `SELECT TO_JSONB(c),revision FROM company_profile c WHERE id=TRUE FOR UPDATE`).Scan(&before, &revision)
	if err != nil {
		ledgerError(w, err)
		return
	}
	if revision != b.Revision {
		writeJSON(w, 409, map[string]string{"error": "Profil telah berubah. Muat ulang profil sebelum menyimpan."})
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	err = tx.QueryRow(r.Context(), `UPDATE company_profile SET name=$1,address=$2,phone=$3,email=$4,tax_id=$5,license_number=$6,maps_url=$7,tax_nitku=$9,revision=revision+1,updated_by=$8,updated_at=NOW() WHERE id=TRUE RETURNING TO_JSONB(company_profile)`, b.Name, b.Address, b.Phone, b.Email, b.TaxID, b.License, b.Maps, claims.UserID, b.TaxNITKU).Scan(&after)
	if err != nil {
		ledgerError(w, err)
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO company_profile_audit(actor,before_profile,after_profile) VALUES($1,$2::jsonb,$3::jsonb)`, claims.UserID, string(before), string(after))
	if err != nil {
		ledgerError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		ledgerError(w, err)
		return
	}
	h.Get(w, r)
}
func (h *CompanyHandler) History(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		writeJSON(w, 403, map[string]string{"error": "Riwayat profil hanya untuk owner"})
		return
	}
	var payload json.RawMessage
	err := h.pool.QueryRow(r.Context(), `SELECT COALESCE(JSONB_AGG(x ORDER BY changed_at DESC,id DESC),'[]'::jsonb) FROM
 (SELECT a.id::text,a.changed_at,COALESCE(u.name,a.actor) AS actor_name,a.before_profile-'updated_by' AS before_profile,a.after_profile-'updated_by' AS after_profile FROM company_profile_audit a LEFT JOIN users u ON u.id::text=a.actor ORDER BY a.changed_at DESC,a.id DESC LIMIT 50) x`).Scan(&payload)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		ledgerError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": payload})
}
