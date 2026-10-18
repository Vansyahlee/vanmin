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

type CustomerHandler struct {
	pool *pgxpool.Pool
}

type updateCustomerRequest struct {
	Name             *string `json:"name"`
	Phone            *string `json:"phone"`
	Email            *string `json:"email"`
	ShippingAddress  *string `json:"shipping_address"`
	TaxIDType        *string `json:"tax_id_type"`
	TaxIDNumber      *string `json:"tax_id_number"`
	TaxName          *string `json:"tax_name"`
	TaxAddress       *string `json:"tax_address"`
	IsPKP            *bool   `json:"is_pkp"`
	TKUID            *string `json:"tku_id"`
	PaymentTermsDays *int    `json:"payment_terms_days"`
	IsActive         *bool   `json:"is_active"`
}

func NewCustomerHandler(
	pool *pgxpool.Pool,
) *CustomerHandler {
	return &CustomerHandler{
		pool: pool,
	}
}

func (h *CustomerHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	customers, err := database.GetCustomers(
		r.Context(),
		h.pool,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil data customer",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		customers,
	)
}

func (h *CustomerHandler) GetByID(
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
				"error": "id customer tidak valid",
			},
		)
		return
	}

	customer, err := database.GetCustomerByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil customer",
			},
		)
		return
	}

	if customer == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "customer tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		customer,
	)
}

func (h *CustomerHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input database.CreateCustomerInput

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

	normalizeCreateCustomerInput(
		&input,
	)

	if input.Name == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "nama customer wajib diisi",
			},
		)
		return
	}

	if !validCustomerTaxIDType(
		input.TaxIDType,
	) {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "tax_id_type harus NPWP, NIK, atau kosong",
			},
		)
		return
	}

	if input.PaymentTermsDays < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "tempo pembayaran tidak boleh negatif",
			},
		)
		return
	}

	customer, err := database.CreateCustomer(
		r.Context(),
		h.pool,
		input,
	)
	if err != nil {
		if isCustomerUniqueViolation(
			err,
		) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "NPWP atau NIK sudah digunakan customer lain",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal membuat customer",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		customer,
	)
}

func (h *CustomerHandler) Update(
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
				"error": "id customer tidak valid",
			},
		)
		return
	}

	existing, err := database.GetCustomerByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil customer",
			},
		)
		return
	}

	if existing == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "customer tidak ditemukan",
			},
		)
		return
	}

	var request updateCustomerRequest

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

	input := database.UpdateCustomerInput{
		Name:             existing.Name,
		Phone:            existing.Phone,
		Email:            existing.Email,
		ShippingAddress:  existing.ShippingAddress,
		TaxIDType:        existing.TaxIDType,
		TaxIDNumber:      existing.TaxIDNumber,
		TaxName:          existing.TaxName,
		TaxAddress:       existing.TaxAddress,
		IsPKP:            existing.IsPKP,
		TKUID:            existing.TKUID,
		PaymentTermsDays: existing.PaymentTermsDays,
		IsActive:         existing.IsActive,
	}

	if request.Name != nil {
		input.Name =
			strings.TrimSpace(
				*request.Name,
			)
	}

	if request.Phone != nil {
		input.Phone =
			strings.TrimSpace(
				*request.Phone,
			)
	}

	if request.Email != nil {
		input.Email =
			strings.ToLower(
				strings.TrimSpace(
					*request.Email,
				),
			)
	}

	if request.ShippingAddress != nil {
		input.ShippingAddress =
			strings.TrimSpace(
				*request.ShippingAddress,
			)
	}

	if request.TaxIDType != nil {
		input.TaxIDType =
			strings.ToUpper(
				strings.TrimSpace(
					*request.TaxIDType,
				),
			)
	}

	if request.TaxIDNumber != nil {
		input.TaxIDNumber =
			normalizeTaxNumber(
				*request.TaxIDNumber,
			)
	}

	if request.TaxName != nil {
		input.TaxName =
			strings.TrimSpace(
				*request.TaxName,
			)
	}

	if request.TaxAddress != nil {
		input.TaxAddress =
			strings.TrimSpace(
				*request.TaxAddress,
			)
	}

	if request.IsPKP != nil {
		input.IsPKP =
			*request.IsPKP
	}

	if request.TKUID != nil {
		input.TKUID =
			strings.TrimSpace(
				*request.TKUID,
			)
	}

	if request.PaymentTermsDays != nil {
		input.PaymentTermsDays =
			*request.PaymentTermsDays
	}

	if request.IsActive != nil {
		input.IsActive =
			*request.IsActive
	}

	if input.Name == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "nama customer tidak boleh kosong",
			},
		)
		return
	}

	if !validCustomerTaxIDType(
		input.TaxIDType,
	) {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "tax_id_type harus NPWP, NIK, atau kosong",
			},
		)
		return
	}

	if input.PaymentTermsDays < 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "tempo pembayaran tidak boleh negatif",
			},
		)
		return
	}

	customer, err := database.UpdateCustomerByID(
		r.Context(),
		h.pool,
		id,
		input,
	)
	if err != nil {
		if isCustomerUniqueViolation(
			err,
		) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "NPWP atau NIK sudah digunakan customer lain",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal memperbarui customer",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		customer,
	)
}

func (h *CustomerHandler) Delete(
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
				"error": "id customer tidak valid",
			},
		)
		return
	}

	deleted, err := database.DeleteCustomerByID(
		r.Context(),
		h.pool,
		id,
	)
	if err != nil {
		if isCustomerForeignKeyViolation(
			err,
		) {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "customer sudah memiliki riwayat pesanan. Gunakan Nonaktifkan agar histori tetap aman",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal menghapus customer",
			},
		)
		return
	}

	if !deleted {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "customer tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "customer berhasil dihapus permanen",
		},
	)
}

func normalizeCreateCustomerInput(
	input *database.CreateCustomerInput,
) {
	input.Name =
		strings.TrimSpace(
			input.Name,
		)

	input.Phone =
		strings.TrimSpace(
			input.Phone,
		)

	input.Email =
		strings.ToLower(
			strings.TrimSpace(
				input.Email,
			),
		)

	input.ShippingAddress =
		strings.TrimSpace(
			input.ShippingAddress,
		)

	input.TaxIDType =
		strings.ToUpper(
			strings.TrimSpace(
				input.TaxIDType,
			),
		)

	input.TaxIDNumber =
		normalizeTaxNumber(
			input.TaxIDNumber,
		)

	input.TaxName =
		strings.TrimSpace(
			input.TaxName,
		)

	input.TaxAddress =
		strings.TrimSpace(
			input.TaxAddress,
		)

	input.TKUID =
		strings.TrimSpace(
			input.TKUID,
		)
}

func normalizeTaxNumber(
	value string,
) string {
	replacer :=
		strings.NewReplacer(
			".", "",
			"-", "",
			" ", "",
		)

	return replacer.Replace(
		strings.TrimSpace(
			value,
		),
	)
}

func validCustomerTaxIDType(
	value string,
) bool {
	return value == "" ||
		value == "NPWP" ||
		value == "NIK"
}

func isCustomerUniqueViolation(
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

func isCustomerForeignKeyViolation(
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
