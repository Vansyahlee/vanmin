package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductHandler struct {
	pool *pgxpool.Pool
}

type updateProductRequest struct {
	WarehouseID    *uuid.UUID `json:"warehouse_id"`
	SKU            *string    `json:"sku"`
	Name           *string    `json:"name"`
	Classification *string    `json:"classification"`
	Unit           *string    `json:"unit"`
	PurchasePrice  *float64   `json:"purchase_price"`
	Price          *float64   `json:"price"`
	Stock          *int       `json:"stock"`
	MinStock       *int       `json:"min_stock"`
	Barcode        *string    `json:"barcode"`
	IsActive       *bool      `json:"is_active"`
}

func NewProductHandler(
	pool *pgxpool.Pool,
) *ProductHandler {
	return &ProductHandler{
		pool: pool,
	}
}

func (h *ProductHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	products, err := database.GetProducts(
		r.Context(),
		h.pool,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil data produk",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		products,
	)
}

func (h *ProductHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(
		r.PathValue("id"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "id produk tidak valid",
			},
		)
		return
	}

	product, err := database.GetProductByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil produk",
			},
		)
		return
	}

	if product == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "produk tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		product,
	)
}

func (h *ProductHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input database.CreateProductInput

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "format JSON tidak valid",
			},
		)
		return
	}

	input.SKU = strings.ToUpper(
		strings.TrimSpace(
			input.SKU,
		),
	)

	input.Name = strings.TrimSpace(
		input.Name,
	)

	input.Classification = strings.ToUpper(
		strings.TrimSpace(
			input.Classification,
		),
	)

	input.Unit = strings.ToUpper(
		strings.TrimSpace(
			input.Unit,
		),
	)

	if input.Barcode != nil {
		barcode := strings.TrimSpace(
			*input.Barcode,
		)

		if barcode == "" {
			input.Barcode = nil
		} else {
			input.Barcode = &barcode
		}
	}

	if input.SKU == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "sku wajib diisi",
			},
		)
		return
	}

	if input.Name == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "nama produk wajib diisi",
			},
		)
		return
	}

	if input.WarehouseID != nil {
		warehouse, warehouseErr := database.GetWarehouseByID(r.Context(), h.pool, *input.WarehouseID)
		if warehouseErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Data gudang tidak dapat diperiksa"})
			return
		}
		if warehouse == nil || !warehouse.IsActive {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pilih gudang yang aktif"})
			return
		}
		input.Classification = warehouse.Classification
	}

	if !validClassification(
		input.Classification,
	) {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "classification harus REGULER atau PREKURSOR",
			},
		)
		return
	}

	if input.Unit == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "unit wajib diisi",
			},
		)
		return
	}

	if input.PurchasePrice < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "harga masuk tidak boleh negatif",
			},
		)
		return
	}

	if input.Price < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "harga jual tidak boleh negatif",
			},
		)
		return
	}

	if input.Stock < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "stok tidak boleh negatif",
			},
		)
		return
	}

	if input.MinStock < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "batas pesan ulang tidak boleh negatif",
			},
		)
		return
	}

	product, err := database.CreateProduct(
		r.Context(),
		h.pool,
		input,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "sku atau barcode sudah digunakan produk lain",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal membuat produk",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		product,
	)
}

func (h *ProductHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(
		r.PathValue("id"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "id produk tidak valid",
			},
		)
		return
	}

	existing, err := database.GetProductByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil produk",
			},
		)
		return
	}

	if existing == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "produk tidak ditemukan",
			},
		)
		return
	}

	var request updateProductRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "format JSON tidak valid",
			},
		)
		return
	}

	input := database.UpdateProductInput{
		WarehouseID:    existing.WarehouseID,
		SKU:            existing.SKU,
		Name:           existing.Name,
		Classification: existing.Classification,
		Unit:           existing.Unit,
		PurchasePrice:  existing.PurchasePrice,
		Price:          existing.Price,
		Stock:          existing.Stock,
		MinStock:       existing.MinStock,
		Barcode:        existing.Barcode,
		IsActive:       existing.IsActive,
	}

	if request.WarehouseID != nil {
		input.WarehouseID = request.WarehouseID
	}

	if request.SKU != nil {
		input.SKU = strings.ToUpper(
			strings.TrimSpace(
				*request.SKU,
			),
		)
	}

	if request.Name != nil {
		input.Name = strings.TrimSpace(
			*request.Name,
		)
	}

	if request.Classification != nil {
		input.Classification = strings.ToUpper(
			strings.TrimSpace(
				*request.Classification,
			),
		)
	}

	if request.Unit != nil {
		input.Unit = strings.ToUpper(
			strings.TrimSpace(
				*request.Unit,
			),
		)
	}

	if request.PurchasePrice != nil {
		input.PurchasePrice =
			*request.PurchasePrice
	}

	if request.Price != nil {
		input.Price =
			*request.Price
	}

	if request.Stock != nil {
		input.Stock =
			*request.Stock
	}

	if request.MinStock != nil {
		input.MinStock =
			*request.MinStock
	}

	if request.Barcode != nil {
		barcode := strings.TrimSpace(
			*request.Barcode,
		)

		if barcode == "" {
			input.Barcode = nil
		} else {
			input.Barcode = &barcode
		}
	}

	if request.IsActive != nil {
		input.IsActive =
			*request.IsActive
	}

	if input.SKU == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "sku tidak boleh kosong",
			},
		)
		return
	}

	if input.Name == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "nama produk tidak boleh kosong",
			},
		)
		return
	}

	if input.WarehouseID != nil {
		warehouse, warehouseErr := database.GetWarehouseByID(r.Context(), h.pool, *input.WarehouseID)
		if warehouseErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Data gudang tidak dapat diperiksa"})
			return
		}
		if warehouse == nil || !warehouse.IsActive {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pilih gudang yang aktif"})
			return
		}
		input.Classification = warehouse.Classification
	}

	if !validClassification(
		input.Classification,
	) {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "classification harus REGULER atau PREKURSOR",
			},
		)
		return
	}

	if input.Unit == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "unit tidak boleh kosong",
			},
		)
		return
	}

	if input.PurchasePrice < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "harga masuk tidak boleh negatif",
			},
		)
		return
	}

	if input.Price < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "harga jual tidak boleh negatif",
			},
		)
		return
	}

	if input.Stock < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "stok tidak boleh negatif",
			},
		)
		return
	}

	if input.MinStock < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "batas pesan ulang tidak boleh negatif",
			},
		)
		return
	}

	product, err := database.UpdateProductByID(
		r.Context(),
		h.pool,
		id,
		input,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "sku atau barcode sudah digunakan produk lain",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal memperbarui produk",
			},
		)
		return
	}

	if product == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "produk tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		product,
	)
}

func (h *ProductHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(
		r.PathValue("id"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "id produk tidak valid",
			},
		)
		return
	}

	deleted, err := database.DeleteProductByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "produk sudah memiliki riwayat transaksi atau pergerakan stok. Produk tidak boleh dihapus permanen; gunakan Nonaktifkan agar histori tetap aman",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal menghapus produk",
			},
		)
		return
	}

	if !deleted {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "produk tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "produk berhasil dihapus permanen",
		},
	)
}

func validClassification(
	value string,
) bool {
	return value == "REGULER" ||
		value == "PREKURSOR"
}

func isUniqueViolation(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if errors.As(
		err,
		&pgErr,
	) {
		return pgErr.Code == "23505"
	}

	return false
}

func isForeignKeyViolation(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if errors.As(
		err,
		&pgErr,
	) {
		return pgErr.Code == "23503"
	}

	return false
}
