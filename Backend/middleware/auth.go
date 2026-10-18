package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"

	"furi-backend/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const claimsKey contextKey = "claims"

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "token diperlukan",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "format token tidak valid",
			})
			return
		}

		claims, err := services.ValidateJWT(parts[1])
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "token tidak valid atau sudah kedaluwarsa",
			})
			return
		}

		ctx := context.WithValue(
			r.Context(),
			claimsKey,
			claims,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	}
}

func GetClaims(ctx context.Context) (*services.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*services.Claims)
	return claims, ok
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func RequireRole(
	next http.HandlerFunc,
	allowedRoles ...string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "data autentikasi tidak ditemukan",
			})
			return
		}

		for _, role := range allowedRoles {
			if claims.Role == role {
				next.ServeHTTP(w, r)
				return
			}
		}

		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "akses tidak diizinkan untuk role ini",
		})
	}
}

// Verify current account state on every protected request; old tokens are
// invalidated immediately by password/role/status edits.
func AuthWithUsers(pool *pgxpool.Pool, next http.HandlerFunc) http.HandlerFunc {
	return Auth(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := GetClaims(r.Context())
		var active bool
		var role string
		var revision int64
		err := pool.QueryRow(r.Context(), `SELECT is_active,role,auth_revision FROM users WHERE id::text=$1`, claims.UserID).Scan(&active, &role, &revision)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, 503, map[string]string{"error": "Pemeriksaan sesi belum tersedia. Coba kembali."})
			return
		}
		if err != nil || !active || role != claims.Role || revision != claims.AuthRevision {
			writeJSON(w, 401, map[string]string{"error": "Sesi akun sudah berubah atau tidak aktif. Silakan masuk kembali."})
			return
		}
		next(w, r)
	})
}
