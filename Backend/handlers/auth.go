package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"furi-backend/internal/database"
	"furi-backend/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	pool *pgxpool.Pool
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  LoggedInUser `json:"user"`
}

type LoggedInUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func NewAuthHandler(pool *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{
		pool: pool,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request tidak valid",
		})
		return
	}

	request.Email = strings.TrimSpace(request.Email)

	if request.Email == "" || request.Password == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "email dan password wajib diisi",
		})
		return
	}

	user, err := database.FindUserByEmail(
		r.Context(),
		h.pool,
		request.Email,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "email atau password salah",
			})
			return
		}

		log.Printf("Gagal mencari user: %v", err)

		writeAuthJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "terjadi kesalahan server",
		})
		return
	}

	if !user.IsActive {
		writeAuthJSON(w, http.StatusForbidden, map[string]string{
			"error": "akun tidak aktif",
		})
		return
	}

	if err := services.CheckPassword(
		request.Password,
		user.PasswordHash,
	); err != nil {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "email atau password salah",
		})
		return
	}

	token, err := services.GenerateJWT(
		user.ID,
		user.Email,
		user.Role,
		user.AuthRevision,
	)

	if err != nil {
		log.Printf("Gagal membuat token: %v", err)

		writeAuthJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "gagal membuat token",
		})
		return
	}

	response := LoginResponse{
		Token: token,
		User: LoggedInUser{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}

	writeAuthJSON(w, http.StatusOK, response)
}

func writeAuthJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Gagal menulis response JSON: %v", err)
	}
}
