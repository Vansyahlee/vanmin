package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"furi-backend/services"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinanceSalesHandler struct {
	service *services.FinanceSalesService
}

type financeSalesMeta struct {
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	Query      string `json:"query"`
	Status     string `json:"status"`
	PrintState string `json:"print_state"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Total      int64  `json:"total"`
}

type financeSalesListResponse struct {
	Data any              `json:"data"`
	Meta financeSalesMeta `json:"meta"`
}

type financeSalesSummaryResponse struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Query    string `json:"query"`
	Status   string `json:"status"`
	Data     any    `json:"data"`
}

type financeSalesTrendResponse struct {
	DateFrom    string `json:"date_from"`
	DateTo      string `json:"date_to"`
	Query       string `json:"query"`
	Status      string `json:"status"`
	Granularity string `json:"granularity"`
	Data        any    `json:"data"`
}

func NewFinanceSalesHandler(
	pool *pgxpool.Pool,
) *FinanceSalesHandler {
	return &FinanceSalesHandler{
		service: services.NewFinanceSalesService(
			pool,
		),
	}
}

func (h *FinanceSalesHandler) GetSales(
	w http.ResponseWriter,
	r *http.Request,
) {
	filter, err := parseFinanceSalesFilter(r)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	sales, total, err := h.service.List(
		r.Context(),
		filter,
	)
	if err != nil {
		if errors.Is(
			err,
			services.ErrFinanceInvalidDateRange,
		) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "Tanggal awal tidak boleh melewati tanggal akhir.",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Data penjualan Finance belum dapat dimuat.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		financeSalesListResponse{
			Data: sales,
			Meta: financeSalesMeta{
				DateFrom:   filter.DateFrom.Format("2006-01-02"),
				DateTo:     filter.DateTo.Format("2006-01-02"),
				Query:      filter.Query,
				Status:     filter.Status,
				PrintState: filter.PrintState,
				Limit:      filter.Limit,
				Offset:     filter.Offset,
				Total:      total,
			},
		},
	)
}

func (h *FinanceSalesHandler) GetSummary(
	w http.ResponseWriter,
	r *http.Request,
) {
	filter, err := parseFinanceSalesFilter(r)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	summary, err := h.service.Summary(
		r.Context(),
		filter,
	)
	if err != nil {
		if errors.Is(
			err,
			services.ErrFinanceInvalidDateRange,
		) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "Tanggal awal tidak boleh melewati tanggal akhir.",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Ringkasan penjualan Finance belum dapat dimuat.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		financeSalesSummaryResponse{
			DateFrom: filter.DateFrom.Format("2006-01-02"),
			DateTo:   filter.DateTo.Format("2006-01-02"),
			Query:    filter.Query,
			Status:   filter.Status,
			Data:     summary,
		},
	)
}

func (h *FinanceSalesHandler) GetTrend(
	w http.ResponseWriter,
	r *http.Request,
) {
	filter, err := parseFinanceSalesFilter(r)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	trend, err := h.service.Trend(
		r.Context(),
		filter,
	)
	if err != nil {
		if errors.Is(
			err,
			services.ErrFinanceInvalidDateRange,
		) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "Tanggal awal tidak boleh melewati tanggal akhir.",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Grafik penjualan Finance belum dapat dimuat.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		financeSalesTrendResponse{
			DateFrom:    filter.DateFrom.Format("2006-01-02"),
			DateTo:      filter.DateTo.Format("2006-01-02"),
			Query:       filter.Query,
			Status:      filter.Status,
			Granularity: trend.Granularity,
			Data:        trend.Points,
		},
	)
}

func (h *FinanceSalesHandler) GetInvoiceByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(
		strings.TrimSpace(
			r.PathValue("id"),
		),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "ID faktur tidak valid.",
			},
		)
		return
	}

	invoice, err := h.service.GetInvoiceByID(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			services.ErrFinanceInvoiceNotFound,
		) {
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "Faktur tidak ditemukan.",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Faktur Finance belum dapat dimuat.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		invoice,
	)
}

func parseFinanceSalesFilter(
	r *http.Request,
) (services.FinanceSalesFilter, error) {
	now := time.Now()

	dateFrom := time.Date(
		now.Year(),
		now.Month(),
		1,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	dateTo := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)

	query := r.URL.Query()

	if raw := strings.TrimSpace(query.Get("date_from")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return services.FinanceSalesFilter{}, errors.New(
				"Format date_from harus YYYY-MM-DD.",
			)
		}
		dateFrom = parsed
	}

	if raw := strings.TrimSpace(query.Get("date_to")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return services.FinanceSalesFilter{}, errors.New(
				"Format date_to harus YYYY-MM-DD.",
			)
		}
		dateTo = parsed
	}

	if dateFrom.After(dateTo) {
		return services.FinanceSalesFilter{}, errors.New(
			"Tanggal awal tidak boleh melewati tanggal akhir.",
		)
	}

	limit := 50
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return services.FinanceSalesFilter{}, errors.New(
				"Limit harus berupa angka lebih dari 0.",
			)
		}
		limit = parsed
	}
	if limit > 200 {
		limit = 200
	}

	offset := 0
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return services.FinanceSalesFilter{}, errors.New(
				"Offset harus berupa angka 0 atau lebih.",
			)
		}
		offset = parsed
	}

	printState := strings.ToLower(
		strings.TrimSpace(
			query.Get("print_state"),
		),
	)
	if printState == "" {
		printState = "all"
	}

	switch printState {
	case "all", "printed", "unprinted":
	default:
		return services.FinanceSalesFilter{}, errors.New(
			"print_state harus all, printed, atau unprinted.",
		)
	}

	return services.FinanceSalesFilter{
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Query:      strings.TrimSpace(query.Get("q")),
		Status:     strings.ToUpper(strings.TrimSpace(query.Get("status"))),
		PrintState: printState,
		Limit:      limit,
		Offset:     offset,
	}, nil
}
