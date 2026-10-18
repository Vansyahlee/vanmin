package handlers

import (
	"encoding/json"
	"errors"
	"furi-backend/middleware"
	"furi-backend/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type UsersHandler struct{ pool *pgxpool.Pool }

func NewUsersHandler(pool *pgxpool.Pool) *UsersHandler { return &UsersHandler{pool: pool} }

type managedUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	Revision  int64     `json:"revision"`
}
type userChange struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
	Password string `json:"password"`
	Revision int64  `json:"revision"`
}

func validUserChange(b *userChange, passwordOnly, create bool) string {
	b.Name = strings.TrimSpace(b.Name)
	b.Email = strings.ToLower(strings.TrimSpace(b.Email))
	if !passwordOnly {
		a, e := mail.ParseAddress(b.Email)
		if b.Name == "" || utf8.RuneCountInString(b.Name) > 150 || e != nil || a.Address != b.Email || len(b.Email) > 254 {
			return "Isi nama dan alamat email yang valid."
		}
		if b.Role != "admin" && b.Role != "finance" && b.Role != "pegawai" {
			return "Peran tidak valid."
		}
	}
	if create || passwordOnly {
		if utf8.RuneCountInString(b.Password) < 8 || len(b.Password) > 72 {
			return "Password minimal 8 karakter dan maksimal 72 byte."
		}
	} else if b.Password != "" {
		return "Gunakan aksi Reset Password untuk mengganti password."
	}
	if !create && b.Revision < 0 {
		return "Revisi akun tidak valid."
	}
	return ""
}
func userError(w http.ResponseWriter, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		writeJSON(w, 409, map[string]string{"error": "Email sudah digunakan. Muat ulang daftar sebelum mencoba lagi."})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"error": "Akun tidak ditemukan."})
		return
	}
	writeJSON(w, 500, map[string]string{"error": "Data akun belum dapat diproses. Periksa migrasi dan koneksi database."})
}
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	role, status := q.Get("role"), q.Get("status")
	limit, offset := 50, 0
	if q.Has("limit") {
		n, e := strconv.Atoi(q.Get("limit"))
		if e != nil || n < 1 || n > 200 {
			writeJSON(w, 400, map[string]string{"error": "Limit 1–200."})
			return
		}
		limit = n
	}
	if q.Has("offset") {
		n, e := strconv.Atoi(q.Get("offset"))
		if e != nil || n < 0 {
			writeJSON(w, 400, map[string]string{"error": "Offset tidak valid."})
			return
		}
		offset = n
	}
	if (role != "" && role != "admin" && role != "finance" && role != "pegawai") || (status != "" && status != "active" && status != "inactive") || len(q.Get("query")) > 200 {
		writeJSON(w, 400, map[string]string{"error": "Filter tidak valid."})
		return
	}
	var payload []byte
	err := h.pool.QueryRow(r.Context(), `WITH filtered AS (
 SELECT id::text,name,email,role,is_active,created_at,auth_revision AS revision FROM users
 WHERE ($1='' OR STRPOS(LOWER(name||' '||email),LOWER($1))>0) AND ($2='' OR role=$2)
 AND ($3='' OR is_active=($3='active'))
 ), page AS(SELECT * FROM filtered ORDER BY name,id LIMIT $4 OFFSET $5)
 SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page) ORDER BY name,id) FROM page),'[]'::jsonb),
 'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered)),
 'summary',JSONB_BUILD_OBJECT('active',(SELECT COUNT(*) FROM filtered WHERE is_active),'inactive',(SELECT COUNT(*) FROM filtered WHERE NOT is_active),'admins',(SELECT COUNT(*) FROM filtered WHERE role='admin' AND is_active)))`, strings.TrimSpace(q.Get("query")), role, status, limit, offset).Scan(&payload)
	if err != nil {
		userError(w, err)
		return
	}
	writeJSON(w, 200, json.RawMessage(payload))
}
func (h *UsersHandler) History(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(r.PathValue("id"))
	if e != nil {
		writeJSON(w, 400, map[string]string{"error": "ID akun tidak valid."})
		return
	}
	var payload []byte
	e = h.pool.QueryRow(r.Context(), `SELECT COALESCE(JSONB_AGG(TO_JSONB(x) ORDER BY created_at DESC,id DESC),'[]'::jsonb) FROM
 (SELECT a.id::text,a.action,a.before_data,a.after_data,a.created_at,u.name AS actor_name FROM user_management_audit a JOIN users u ON u.id=a.actor_id WHERE a.user_id=$1 ORDER BY a.created_at DESC,a.id DESC LIMIT 50) x`, id).Scan(&payload)
	if e != nil {
		userError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"data": json.RawMessage(payload)})
}
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) { h.change(w, r, true, false) }
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) { h.change(w, r, false, false) }
func (h *UsersHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	h.change(w, r, false, true)
}
func (h *UsersHandler) change(w http.ResponseWriter, r *http.Request, create, passwordOnly bool) {
	var b userChange
	if !decodePaymentJSON(w, r, &b, true) {
		return
	}
	if msg := validUserChange(&b, passwordOnly, create); msg != "" {
		writeJSON(w, 400, map[string]string{"error": msg})
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.Role != "admin" {
		writeJSON(w, 403, map[string]string{"error": "Khusus administrator."})
		return
	}
	actor, e := uuid.Parse(claims.UserID)
	if e != nil {
		writeJSON(w, 401, map[string]string{"error": "Sesi tidak valid."})
		return
	}
	id := uuid.New()
	if !create {
		id, e = uuid.Parse(r.PathValue("id"))
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "ID akun tidak valid."})
			return
		}
	}
	hash := ""
	if create || passwordOnly {
		hash, e = services.HashPassword(b.Password)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "Password belum dapat diproses."})
			return
		}
	}
	tx, e := h.pool.Begin(r.Context())
	if e != nil {
		userError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize account edits to preserve the last active administrator.
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(20261016)`); e != nil {
		userError(w, e)
		return
	}
	var actorEnabled bool
	e = tx.QueryRow(r.Context(), `SELECT is_active AND role='admin' AND auth_revision=$2 FROM users WHERE id=$1`, actor, claims.AuthRevision).Scan(&actorEnabled)
	if e != nil || !actorEnabled {
		writeJSON(w, 401, map[string]string{"error": "Sesi administrator berubah. Masuk kembali."})
		return
	}
	var before *managedUser
	action := "CREATE"
	if !create {
		before = &managedUser{}
		e = tx.QueryRow(r.Context(), `SELECT id::text,name,email,role,is_active,created_at,auth_revision FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&before.ID, &before.Name, &before.Email, &before.Role, &before.IsActive, &before.CreatedAt, &before.Revision)
		if e != nil {
			userError(w, e)
			return
		}
		if before.Revision != b.Revision {
			writeJSON(w, 409, map[string]string{"error": "Akun sudah diubah petugas lain. Muat ulang sebelum menyimpan."})
			return
		}
		if passwordOnly && id.String() == strings.TrimSpace(os.Getenv("COMPANY_OWNER_USER_ID")) && id != actor {
			writeJSON(w, 403, map[string]string{"error": "Password akun owner tidak dapat direset oleh admin lain."})
			return
		}
		if passwordOnly && id == actor {
			writeJSON(w, 400, map[string]string{"error": "Reset password ini untuk akun pengguna lain."})
			return
		}
		if !passwordOnly && (id == actor || id.String() == os.Getenv("COMPANY_OWNER_USER_ID")) && (!b.IsActive || b.Role != "admin") {
			writeJSON(w, 400, map[string]string{"error": "Akun sendiri atau owner tidak boleh dinonaktifkan/demotasi di sini."})
			return
		}
		if !passwordOnly && before.IsActive && before.Role == "admin" && (!b.IsActive || b.Role != "admin") {
			var n int
			e = tx.QueryRow(r.Context(), `SELECT COUNT(*) FROM users WHERE is_active AND role='admin'`).Scan(&n)
			if e != nil {
				userError(w, e)
				return
			}
			if n <= 1 {
				writeJSON(w, 409, map[string]string{"error": "Minimal satu administrator aktif harus dipertahankan."})
				return
			}
		}
		action = "UPDATE"
		if passwordOnly {
			action = "RESET_PASSWORD"
		}
	}
	var after managedUser
	const fields = ` RETURNING id::text,name,email,role,is_active,created_at,auth_revision`
	var result pgx.Row
	if create {
		result = tx.QueryRow(r.Context(), `INSERT INTO users(id,name,email,role,is_active,password_hash) VALUES($1,$2,$3,$4,$5,$6)`+fields, id, b.Name, b.Email, b.Role, b.IsActive, hash)
	} else if passwordOnly {
		result = tx.QueryRow(r.Context(), `UPDATE users SET password_hash=$2,auth_revision=auth_revision+1 WHERE id=$1`+fields, id, hash)
	} else {
		result = tx.QueryRow(r.Context(), `UPDATE users SET name=$2,email=$3,role=$4,is_active=$5,auth_revision=auth_revision+1 WHERE id=$1`+fields, id, b.Name, b.Email, b.Role, b.IsActive)
	}
	e = result.Scan(&after.ID, &after.Name, &after.Email, &after.Role, &after.IsActive, &after.CreatedAt, &after.Revision)
	if e != nil {
		userError(w, e)
		return
	}
	oldJSON, _ := json.Marshal(before)
	newJSON, _ := json.Marshal(after)
	_, e = tx.Exec(r.Context(), `INSERT INTO user_management_audit(user_id,actor_id,action,before_data,after_data) VALUES($1,$2,$3,$4::jsonb,$5::jsonb)`, id, actor, action, string(oldJSON), string(newJSON))
	if e != nil {
		userError(w, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		userError(w, e)
		return
	}
	status := 200
	if create {
		status = 201
	}
	writeJSON(w, status, map[string]any{"data": after, "message": "Perubahan akun tersimpan.", "reauthenticate": !create && id == actor})
}
