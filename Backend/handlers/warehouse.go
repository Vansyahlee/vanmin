package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WarehouseHandler struct {
	pool *pgxpool.Pool
}

type warehouseRequest struct {
	Name           string `json:"name"`
	Classification string `json:"classification"`
	Address        string `json:"address"`
	IsActive       *bool  `json:"is_active"`
}

func NewWarehouseHandler(
	pool *pgxpool.Pool,
) *WarehouseHandler {
	return &WarehouseHandler{
		pool: pool,
	}
}

func (h *WarehouseHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	warehouses, err :=
		database.GetWarehouses(
			r.Context(),
			h.pool,
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Data gudang tidak dapat dimuat. Silakan coba kembali.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		warehouses,
	)
}

func (h *WarehouseHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request warehouseRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Data gudang tidak valid.",
			},
		)
		return
	}

	name :=
		strings.TrimSpace(
			request.Name,
		)

	if len(name) < 3 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Nama gudang minimal 3 karakter.",
			},
		)
		return
	}

	classification :=
		strings.ToUpper(
			strings.TrimSpace(
				request.Classification,
			),
		)

	if classification == "" {
		classification = "REGULER"
	}

	if classification != "REGULER" &&
		classification != "PREKURSOR" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Jenis gudang tidak valid.",
			},
		)
		return
	}

	warehouse, err :=
		database.CreateWarehouse(
			r.Context(),
			h.pool,
			database.CreateWarehouseInput{
				Name:           name,
				Classification: classification,
				Address:        request.Address,
			},
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Gudang baru tidak dapat ditambahkan. Silakan coba kembali.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		warehouse,
	)
}

func (h *WarehouseHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseWarehouseID(
			w,
			r,
		)
	if !ok {
		return
	}

	current, err :=
		database.GetWarehouseByID(
			r.Context(),
			h.pool,
			id,
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Data gudang tidak dapat dimuat.",
			},
		)
		return
	}
	if current == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "Gudang tidak ditemukan.",
			},
		)
		return
	}

	var request warehouseRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Data gudang tidak valid.",
			},
		)
		return
	}

	name :=
		strings.TrimSpace(
			request.Name,
		)
	if name == "" {
		name = current.Name
	}

	classification :=
		strings.ToUpper(
			strings.TrimSpace(
				request.Classification,
			),
		)
	if classification == "" {
		classification =
			current.Classification
	}

	if classification != "REGULER" &&
		classification != "PREKURSOR" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Jenis gudang tidak valid.",
			},
		)
		return
	}

	isActive :=
		current.IsActive

	if request.IsActive != nil {
		isActive =
			*request.IsActive
	}

	warehouse, err :=
		database.UpdateWarehouse(
			r.Context(),
			h.pool,
			id,
			database.UpdateWarehouseInput{
				Name:           name,
				Classification: classification,
				Address:        request.Address,
				IsActive:       isActive,
			},
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Perubahan gudang tidak dapat disimpan.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		warehouse,
	)
}

func (h *WarehouseHandler) Deactivate(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseWarehouseID(
			w,
			r,
		)
	if !ok {
		return
	}

	warehouse, err :=
		database.DeactivateWarehouse(
			r.Context(),
			h.pool,
			id,
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Gudang tidak dapat dinonaktifkan.",
			},
		)
		return
	}
	if warehouse == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "Gudang tidak ditemukan.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		warehouse,
	)
}

func parseWarehouseID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	id, err :=
		uuid.Parse(
			r.PathValue("id"),
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Gudang tidak valid.",
			},
		)
		return uuid.Nil, false
	}

	return id, true
}
