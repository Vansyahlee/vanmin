package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"furi-backend/services"
)

type PredictRequest struct {
	Jumlah float64 `json:"jumlah"`
	ClQty  float64 `json:"cl_qty"`
}

type PredictResponse struct {
	ID          string  `json:"id"`
	Jumlah      float64 `json:"jumlah"`
	ClQty       float64 `json:"cl_qty"`
	Cluster     string  `json:"cluster"`
	Rekomendasi string  `json:"rekomendasi"`
	Waktu       string  `json:"waktu"`
}

var dbPool *pgxpool.Pool

func main() {
	dbURL := "postgres://postgres:postgres@localhost:5432/fas_db?sslmode=disable"

	var err error
	dbPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Gagal konek ke database: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Gagal ping database: %v", err)
	}
	log.Println("✅ Berhasil terhubung ke PostgreSQL (fas_db)!")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Server PT. FAS Full DB Mode jalan!",
		})
	})

	http.HandleFunc("/api/v1/predictions", handlePredict)

	port := ":8080"
	log.Printf("🚀 Server Backend PT. FAS jalan di http://localhost%s", port)
	log.Println("🧠 Otak Decision Tree + Database siap menerima transaksi...")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server mati: ", err)
	}
}

func handlePredict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Harus pakai method POST!", http.StatusMethodNotAllowed)
		return
	}

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Format JSON salah!", http.StatusBadRequest)
		return
	}

	cluster := services.TentukanPrioritas(req.Jumlah, req.ClQty)
	rekomendasi := services.RekomendasiDistribusi(cluster)
	predictID := uuid.NewString()
	now := time.Now()

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO invoices (id, invoice_number, store_name, subtotal, ppn_percent,
			ppn_amount, total, cl_qty, cluster, recommendation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		predictID,
		"PRED-"+now.Format("20060102150405"),
		"SIMULASI API",
		req.Jumlah, 12.00, req.Jumlah*0.12, req.Jumlah*1.12,
		int(req.ClQty), cluster, rekomendasi,
	)
	if err != nil {
		log.Printf("⚠️ Gagal simpan ke DB: %v", err)
		http.Error(w, "Gagal simpan ke database", http.StatusInternalServerError)
		return
	}
	log.Printf("💾 Transaksi tersimpan sebagai %s", cluster)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PredictResponse{
		ID:          predictID,
		Jumlah:      req.Jumlah,
		ClQty:       req.ClQty,
		Cluster:     cluster,
		Rekomendasi: rekomendasi,
		Waktu:       now.Format("2006-01-02 15:04:05"),
	})
}
