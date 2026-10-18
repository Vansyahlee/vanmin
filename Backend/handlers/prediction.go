package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"furi-backend/internal/database"
	"furi-backend/services"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PredictionHandler struct {
	pool *pgxpool.Pool
}

type PredictRequest struct {
	Jumlah *float64 `json:"jumlah"`
	ClQty  *float64 `json:"cl_qty"`
}

type PredictResponse struct {
	Jumlah      float64 `json:"jumlah"`
	ClQty       float64 `json:"cl_qty"`
	Cluster     string  `json:"cluster"`
	Rekomendasi string  `json:"rekomendasi"`
	Waktu       string  `json:"waktu"`
}

func NewPredictionHandler(pool *pgxpool.Pool) *PredictionHandler {
	return &PredictionHandler{
		pool: pool,
	}
}

func (h *PredictionHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req PredictRequest

	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "JSON tidak valid atau terdapat field yang tidak dikenal",
		})
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Body harus berisi tepat satu objek JSON",
		})
		return
	}

	if req.Jumlah == nil || req.ClQty == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "jumlah dan cl_qty wajib diisi",
		})
		return
	}

	if *req.Jumlah < 0 || *req.ClQty < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "jumlah dan cl_qty tidak boleh negatif",
		})
		return
	}

	cluster := services.TentukanPrioritas(
		*req.Jumlah,
		*req.ClQty,
	)

	rekomendasi := services.RekomendasiDistribusi(cluster)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := database.SavePrediction(
		ctx,
		h.pool,
		*req.Jumlah,
		*req.ClQty,
		cluster,
		rekomendasi,
	)
	if err != nil {
		log.Printf("Gagal menyimpan prediksi: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Gagal menyimpan hasil prediksi",
		})
		return
	}

	writeJSON(w, http.StatusOK, PredictResponse{
		Jumlah:      *req.Jumlah,
		ClQty:       *req.ClQty,
		Cluster:     cluster,
		Rekomendasi: rekomendasi,
		Waktu:       time.Now().Format(time.RFC3339),
	})
}

func (h *PredictionHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	predictions, err := database.GetPredictions(ctx, h.pool)
	if err != nil {
		log.Printf("Gagal mengambil riwayat prediksi: %v", err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Gagal mengambil riwayat prediksi",
		})
		return
	}

	writeJSON(w, http.StatusOK, predictions)
}

func (h *PredictionHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	idText := r.PathValue("id")

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "ID harus berupa angka positif",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	prediction, err := database.GetPredictionByID(
		ctx,
		h.pool,
		id,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Prediksi tidak ditemukan",
		})
		return
	}

	writeJSON(w, http.StatusOK, prediction)
}

func (h *PredictionHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	idText := r.PathValue("id")

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "ID harus berupa angka positif",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	deleted, err := database.DeletePredictionByID(
		ctx,
		h.pool,
		id,
	)
	if err != nil {
		log.Printf("Gagal menghapus prediksi ID %d: %v", id, err)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Gagal menghapus prediksi",
		})
		return
	}

	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Prediksi tidak ditemukan",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Prediksi berhasil dihapus",
	})
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Gagal menulis respons JSON: %v", err)
	}
}
