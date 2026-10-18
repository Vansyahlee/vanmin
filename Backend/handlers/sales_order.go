package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"furi-backend/services"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SalesOrderHandler struct {
	service *services.SalesOrderService
}

type createSalesOrderRequest struct {
	CustomerID        string                        `json:"customer_id"`
	WarehouseID       string                        `json:"warehouse_id"`
	OrderDate         string                        `json:"order_date"`
	Notes             string                        `json:"notes"`
	PaymentTermsDays  int                           `json:"payment_terms_days"`
	PaymentSettlement string                        `json:"payment_settlement"`
	PaymentChannel    string                        `json:"payment_channel"`
	PaymentProvider   string                        `json:"payment_provider"`
	TaxMode           string                        `json:"tax_mode"`
	TaxProfileID      string                        `json:"tax_profile_id"`
	DiscountAmount    float64                       `json:"discount_amount"`
	TaxScheme         string                        `json:"tax_scheme"`
	Items             []createSalesOrderItemRequest `json:"items"`
}

type createSalesOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func NewSalesOrderHandler(
	pool *pgxpool.Pool,
) *SalesOrderHandler {
	return &SalesOrderHandler{
		service: services.NewSalesOrderService(
			pool,
		),
	}
}

func (h *SalesOrderHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	orders, err :=
		h.service.GetAll(
			r.Context(),
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil sales order",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		orders,
	)
}

func (h *SalesOrderHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	order, err :=
		h.service.GetByID(
			r.Context(),
			id,
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "gagal mengambil sales order",
			},
		)
		return
	}

	if order == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "sales order tidak ditemukan",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func (h *SalesOrderHandler) CreateDraft(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request createSalesOrderRequest

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

	customerID, err := uuid.Parse(
		strings.TrimSpace(
			request.CustomerID,
		),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "customer_id tidak valid",
			},
		)
		return
	}

	warehouseID, err := uuid.Parse(
		strings.TrimSpace(
			request.WarehouseID,
		),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Pilih gudang asal pesanan.",
			},
		)
		return
	}

	orderDate := time.Now()

	if strings.TrimSpace(
		request.OrderDate,
	) != "" {
		parsedDate, err :=
			time.Parse(
				"2006-01-02",
				request.OrderDate,
			)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "order_date harus berformat YYYY-MM-DD",
				},
			)
			return
		}

		orderDate = parsedDate
	}

	items := make(
		[]services.CreateSalesOrderItemInput,
		0,
		len(request.Items),
	)

	for _, requestItem := range request.Items {
		productID, err :=
			uuid.Parse(
				strings.TrimSpace(
					requestItem.ProductID,
				),
			)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "product_id tidak valid",
				},
			)
			return
		}

		items = append(
			items,
			services.CreateSalesOrderItemInput{
				ProductID: productID,
				Quantity:  requestItem.Quantity,
			},
		)
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

	order, err :=
		h.service.CreateDraft(
			r.Context(),
			services.CreateSalesOrderInput{
				CustomerID:        customerID,
				WarehouseID:       warehouseID,
				OrderDate:         orderDate,
				Notes:             request.Notes,
				PaymentTermsDays:  request.PaymentTermsDays,
				PaymentSettlement: request.PaymentSettlement,
				PaymentChannel:    request.PaymentChannel,
				PaymentProvider:   request.PaymentProvider,
				TaxMode:           request.TaxMode,
				TaxProfileID:      taxProfileID,
				DiscountAmount:    request.DiscountAmount,
				TaxScheme:         request.TaxScheme,
				Items:             items,
			},
		)
	if err != nil {
		writeSalesOrderError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}

func (h *SalesOrderHandler) Confirm(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	order, err :=
		h.service.Confirm(
			r.Context(),
			id,
		)
	if err != nil {
		writeSalesOrderError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func (h *SalesOrderHandler) StartPicking(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	order, err :=
		h.service.StartPicking(
			r.Context(),
			id,
		)
	if err != nil {
		writeSalesOrderError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func (h *SalesOrderHandler) FinishPacking(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	order, err :=
		h.service.FinishPacking(
			r.Context(),
			id,
		)
	if err != nil {
		writeSalesOrderError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func (h *SalesOrderHandler) Ship(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok :=
		parseSalesOrderID(
			w,
			r,
		)
	if !ok {
		return
	}

	order, err :=
		h.service.Ship(
			r.Context(),
			id,
		)
	if err != nil {
		writeSalesOrderError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func parseSalesOrderID(
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
				"error": "id sales order tidak valid",
			},
		)
		return uuid.Nil, false
	}

	return id, true
}

func writeSalesOrderError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrSalesOrderNoItems,
	):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		services.ErrSalesOrderCustomerNotFound,
	),
		errors.Is(
			err,
			services.ErrSalesOrderProductNotFound,
		),
		errors.Is(
			err,
			services.ErrSalesOrderWarehouseNotFound,
		),
		errors.Is(
			err,
			services.ErrSalesOrderNotFound,
		):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		services.ErrSalesOrderCustomerInactive,
	),
		errors.Is(
			err,
			services.ErrSalesOrderProductInactive,
		),
		errors.Is(
			err,
			services.ErrSalesOrderWarehouseInactive,
		):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		services.ErrSalesOrderInvalidTransition,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "Pesanan tidak dapat diproses pada tahap ini.",
			},
		)

	case errors.Is(
		err,
		services.ErrSalesOrderInsufficientStock,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		services.ErrSalesOrderInvalidQuantity,
	),
		errors.Is(
			err,
			services.ErrSalesOrderInvalidDiscount,
		),
		errors.Is(
			err,
			services.ErrSalesOrderInvalidTaxScheme,
		),
		errors.Is(
			err,
			services.ErrSalesOrderInvalidPaymentTerms,
		),
		errors.Is(
			err,
			services.ErrSalesOrderInvalidPaymentMode,
		),
		errors.Is(
			err,
			services.ErrSalesOrderInvalidChannel,
		),
		errors.Is(
			err,
			services.ErrSalesOrderTaxProfileNotFound,
		),
		errors.Is(
			err,
			services.ErrSalesOrderDuplicateProduct,
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
				"error": "gagal memproses sales order",
			},
		)
	}
}
