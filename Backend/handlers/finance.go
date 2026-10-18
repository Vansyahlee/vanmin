package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"furi-backend/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FinanceHandler struct {
	pool *pgxpool.Pool
}

func NewFinanceHandler(pool *pgxpool.Pool) *FinanceHandler {
	return &FinanceHandler{pool: pool}
}

func (h *FinanceHandler) GetReceivables(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := database.ReceivableFilter{
		Query:    strings.TrimSpace(query.Get("query")),
		DateFrom: query.Get("date_from"),
		DateTo:   query.Get("date_to"),
		Status:   strings.ToUpper(query.Get("status")),
		Aging:    strings.ToUpper(query.Get("aging")),
		Limit:    50,
	}
	for _, value := range []string{filter.DateFrom, filter.DateTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Tanggal harus berformat YYYY-MM-DD"})
				return
			}
		}
	}
	if filter.DateFrom != "" && filter.DateTo != "" && filter.DateFrom > filter.DateTo {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Tanggal awal melebihi tanggal akhir"})
		return
	}
	statuses := map[string]bool{"": true, "ALL": true, "OPEN": true, "UNPAID": true, "PARTIAL": true, "PAID": true, "OVERDUE": true, "DUE": true, "NOT_DUE": true, "UNSCHEDULED": true}
	ages := map[string]bool{"": true, "ALL": true, "NOT_DUE": true, "DUE": true, "1_7": true, "8_14": true, "15_30": true, "OVER_30": true, "UNSCHEDULED": true, "PAID": true}
	if !statuses[filter.Status] || !ages[filter.Aging] || len(filter.Query) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Filter piutang tidak valid"})
		return
	}
	if value := query.Get("limit"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 1000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Limit harus 1 sampai 1000"})
			return
		}
		filter.Limit = n
	}
	if value := query.Get("offset"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 || n > 10000000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Offset tidak valid"})
			return
		}
		filter.Offset = n
	}
	payload, err := database.ListFinanceReceivables(r.Context(), h.pool, filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Piutang belum dapat dimuat. Pastikan migrasi gudang dan piutang telah dijalankan."})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *FinanceHandler) GetProductTrend(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := database.ProductTrendFilter{
		DateFrom: query.Get("date_from"), DateTo: query.Get("date_to"),
		Query: strings.TrimSpace(query.Get("query")), Category: strings.ToUpper(query.Get("category")),
		Sort: strings.ToLower(query.Get("sort")), Limit: 25,
	}
	if filter.Sort == "" {
		filter.Sort = "frequency"
	}
	for _, value := range []string{filter.DateFrom, filter.DateTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				writeJSON(w, 400, map[string]string{"error": "Tanggal harus berformat YYYY-MM-DD"})
				return
			}
		}
	}
	if filter.DateFrom != "" && filter.DateTo != "" && filter.DateFrom > filter.DateTo {
		writeJSON(w, 400, map[string]string{"error": "Tanggal awal melebihi tanggal akhir"})
		return
	}
	categories := map[string]bool{"": true, "ALL": true, "FAST": true, "MEDIUM": true, "UNSOLD": true}
	sorts := map[string]bool{"frequency": true, "quantity": true, "revenue": true, "name": true}
	if !categories[filter.Category] || !sorts[filter.Sort] || len(filter.Query) > 200 {
		writeJSON(w, 400, map[string]string{"error": "Filter tren produk tidak valid"})
		return
	}
	for _, key := range []string{"limit", "offset"} {
		if value := query.Get(key); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || (key == "limit" && (n < 1 || n > 1000)) || (key == "offset" && (n < 0 || n > 10000000)) {
				writeJSON(w, 400, map[string]string{"error": "Limit atau offset tidak valid"})
				return
			}
			if key == "limit" {
				filter.Limit = n
			} else {
				filter.Offset = n
			}
		}
	}
	payload, err := database.ListFinanceProductTrend(r.Context(), h.pool, filter)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Statistik produk belum dapat dimuat dari data produk dan faktur"})
		return
	}
	writeJSON(w, 200, payload)
}
