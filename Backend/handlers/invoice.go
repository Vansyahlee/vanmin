package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"furi-backend/internal/database"
	"furi-backend/services"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceHandler struct {
	service *services.InvoiceService
}

type updateInvoiceRequest struct {
	SalesName         string                     `json:"sales_name"`
	WarehouseID       string                     `json:"warehouse_id"`
	PaymentTermsDays  int                        `json:"payment_terms_days"`
	PaymentSettlement string                     `json:"payment_settlement"`
	PaymentChannel    string                     `json:"payment_channel"`
	PaymentProvider   string                     `json:"payment_provider"`
	TaxMode           string                     `json:"tax_mode"`
	TaxProfileID      string                     `json:"tax_profile_id"`
	Notes             string                     `json:"notes"`
	Items             []updateInvoiceItemRequest `json:"items"`
}

type updateInvoiceItemRequest struct {
	ID              string  `json:"id"`
	BatchNumber     string  `json:"batch_number"`
	ExpiryDate      string  `json:"expiry_date"`
	DiscountPercent float64 `json:"discount_percent"`
}

func NewInvoiceHandler(
	pool *pgxpool.Pool,
) *InvoiceHandler {
	return &InvoiceHandler{
		service: services.NewInvoiceService(
			pool,
		),
	}
}

func (h *InvoiceHandler) GetBySalesOrder(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseInvoiceSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	invoice, err :=
		h.service.GetBySalesOrderID(
			r.Context(),
			id,
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Faktur tidak dapat dimuat. Silakan coba kembali.",
			},
		)
		return
	}

	if invoice == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "Faktur belum tersedia untuk pesanan ini.",
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

func (h *InvoiceHandler) EnsureBySalesOrder(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseInvoiceSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	invoice, err :=
		h.service.EnsureForSalesOrder(
			r.Context(),
			id,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrInvoiceOrderNotFound,
		):
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "Pesanan tidak ditemukan.",
				},
			)

		case errors.Is(
			err,
			services.ErrInvoiceOrderNotShipped,
		):
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "Faktur tersedia setelah proses pengemasan selesai.",
				},
			)

		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "Faktur tidak dapat dibuat. Silakan coba kembali.",
				},
			)
		}
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		invoice,
	)
}

func (h *InvoiceHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseInvoiceID(
			w,
			r,
		)
	if !ok {
		return
	}

	var request updateInvoiceRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Data faktur tidak valid.",
			},
		)
		return
	}

	items := make(
		[]services.UpdateInvoiceItemInput,
		0,
		len(request.Items),
	)

	for _, requestItem := range request.Items {
		itemID, err :=
			uuid.Parse(
				strings.TrimSpace(
					requestItem.ID,
				),
			)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "Item faktur tidak valid.",
				},
			)
			return
		}

		var expiryDate *time.Time

		if strings.TrimSpace(
			requestItem.ExpiryDate,
		) != "" {
			value, err :=
				time.Parse(
					"2006-01-02",
					requestItem.ExpiryDate,
				)
			if err != nil {
				writeJSON(
					w,
					http.StatusBadRequest,
					map[string]string{
						"error": "Format tanggal kedaluwarsa tidak valid.",
					},
				)
				return
			}
			expiryDate = &value
		}

		items = append(
			items,
			services.UpdateInvoiceItemInput{
				ID:              itemID,
				BatchNumber:     requestItem.BatchNumber,
				ExpiryDate:      expiryDate,
				DiscountPercent: requestItem.DiscountPercent,
			},
		)
	}

	warehouseID, err :=
		uuid.Parse(
			strings.TrimSpace(
				request.WarehouseID,
			),
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Pilih gudang untuk faktur.",
			},
		)
		return
	}

	var taxProfileID *uuid.UUID

	if strings.TrimSpace(
		request.TaxProfileID,
	) != "" {
		parsedTaxProfileID, err :=
			uuid.Parse(
				strings.TrimSpace(
					request.TaxProfileID,
				),
			)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "Aturan pajak tidak valid.",
				},
			)
			return
		}

		taxProfileID =
			&parsedTaxProfileID
	}

	invoice, err :=
		h.service.Update(
			r.Context(),
			id,
			services.UpdateInvoiceInput{
				SalesName:         request.SalesName,
				WarehouseID:       warehouseID,
				PaymentTermsDays:  request.PaymentTermsDays,
				PaymentSettlement: request.PaymentSettlement,
				PaymentChannel:    request.PaymentChannel,
				PaymentProvider:   request.PaymentProvider,
				TaxMode:           request.TaxMode,
				TaxProfileID:      taxProfileID,
				Notes:             request.Notes,
				Items:             items,
			},
		)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvoiceHasPayments):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Faktur sudah memiliki catatan pembayaran atau tautan Midtrans aktif. Revisi faktur perlu direkonsiliasi dahulu."})

		case errors.Is(
			err,
			services.ErrInvoiceNotFound,
		):
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "Faktur tidak ditemukan.",
				},
			)

		case errors.Is(
			err,
			services.ErrInvoiceInvalidWarehouse,
		),
			errors.Is(
				err,
				services.ErrInvoiceWarehouseInactive,
			),
			errors.Is(
				err,
				services.ErrInvoiceInvalidTaxMode,
			),
			errors.Is(
				err,
				services.ErrInvoiceInvalidPaymentTerms,
			),
			errors.Is(
				err,
				services.ErrInvoiceInvalidPaymentMode,
			),
			errors.Is(
				err,
				services.ErrInvoiceInvalidChannel,
			),
			errors.Is(
				err,
				services.ErrInvoiceTaxProfileNotFound,
			),
			errors.Is(
				err,
				services.ErrInvoiceInvalidDiscount,
			):
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)

		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "Perubahan faktur tidak dapat disimpan. Silakan coba kembali.",
				},
			)
		}
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		invoice,
	)
}

func (h *InvoiceHandler) MarkPrinted(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseInvoiceID(
			w,
			r,
		)
	if !ok {
		return
	}

	invoice, err :=
		h.service.MarkPrinted(
			r.Context(),
			id,
		)
	if err != nil {
		if errors.Is(
			err,
			services.ErrInvoiceNotFound,
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
				"error": "Status cetak faktur tidak dapat diperbarui.",
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

func parseInvoiceSalesOrderID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	id, err := uuid.Parse(
		r.PathValue("id"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Pesanan tidak valid.",
			},
		)
		return uuid.Nil, false
	}

	return id, true
}

func parseInvoiceID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	id, err := uuid.Parse(
		r.PathValue("id"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Faktur tidak valid.",
			},
		)
		return uuid.Nil, false
	}

	return id, true
}
