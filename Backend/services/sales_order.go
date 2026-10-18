package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSalesOrderNoItems             = errors.New("sales order minimal memiliki satu item")
	ErrSalesOrderCustomerNotFound    = errors.New("customer tidak ditemukan")
	ErrSalesOrderCustomerInactive    = errors.New("customer sedang nonaktif")
	ErrSalesOrderWarehouseNotFound   = errors.New("gudang tidak ditemukan")
	ErrSalesOrderWarehouseInactive   = errors.New("gudang sedang nonaktif")
	ErrSalesOrderProductNotFound     = errors.New("produk tidak ditemukan")
	ErrSalesOrderProductInactive     = errors.New("produk sedang nonaktif")
	ErrSalesOrderInvalidQuantity     = errors.New("quantity harus lebih dari 0")
	ErrSalesOrderInvalidDiscount     = errors.New("diskon tidak valid")
	ErrSalesOrderInvalidTaxScheme    = errors.New("skema pajak tidak valid")
	ErrSalesOrderDuplicateProduct    = errors.New("produk yang sama tidak boleh dimasukkan dua kali")
	ErrSalesOrderInvalidPaymentTerms = errors.New("tempo pembayaran tidak valid")
	ErrSalesOrderInvalidPaymentMode  = errors.New("skema pembayaran tidak valid")
	ErrSalesOrderInvalidChannel      = errors.New("kanal pembayaran tidak valid")
	ErrSalesOrderTaxProfileNotFound  = errors.New("aturan PPN aktif tidak ditemukan")
	ErrSalesOrderNotFound            = errors.New("sales order tidak ditemukan")
	ErrSalesOrderInvalidTransition   = errors.New("perubahan status sales order tidak valid")
	ErrSalesOrderInsufficientStock   = errors.New("stok tidak mencukupi")
)

type SalesOrderService struct {
	pool *pgxpool.Pool
}

type CreateSalesOrderItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type CreateSalesOrderInput struct {
	CustomerID        uuid.UUID
	WarehouseID       uuid.UUID
	OrderDate         time.Time
	Notes             string
	PaymentTermsDays  int
	PaymentSettlement string
	PaymentChannel    string
	PaymentProvider   string
	TaxMode           string
	TaxProfileID      *uuid.UUID
	DiscountAmount    float64
	TaxScheme         string
	Items             []CreateSalesOrderItemInput
}

func NewSalesOrderService(
	pool *pgxpool.Pool,
) *SalesOrderService {
	return &SalesOrderService{
		pool: pool,
	}
}

func (s *SalesOrderService) CreateDraft(
	ctx context.Context,
	input CreateSalesOrderInput,
) (*database.SalesOrder, error) {
	if len(input.Items) == 0 {
		return nil, ErrSalesOrderNoItems
	}

	customer, err := database.GetSalesOrderCustomer(
		ctx,
		s.pool,
		input.CustomerID,
	)
	if err != nil {
		return nil, err
	}

	if customer == nil {
		return nil, ErrSalesOrderCustomerNotFound
	}

	if !customer.IsActive {
		return nil, ErrSalesOrderCustomerInactive
	}

	if input.WarehouseID == uuid.Nil {
		return nil, ErrSalesOrderWarehouseNotFound
	}

	warehouse, err :=
		database.GetSalesOrderWarehouse(
			ctx,
			s.pool,
			input.WarehouseID,
		)
	if err != nil {
		return nil, err
	}

	if warehouse == nil {
		return nil, ErrSalesOrderWarehouseNotFound
	}

	if !warehouse.IsActive {
		return nil, ErrSalesOrderWarehouseInactive
	}

	if input.OrderDate.IsZero() {
		input.OrderDate = time.Now()
	}

	input.Notes = strings.TrimSpace(
		input.Notes,
	)

	if input.PaymentTermsDays < 0 ||
		input.PaymentTermsDays > 365 {
		return nil, ErrSalesOrderInvalidPaymentTerms
	}

	paymentSettlement :=
		strings.ToUpper(
			strings.TrimSpace(
				input.PaymentSettlement,
			),
		)

	switch paymentSettlement {
	case "IMMEDIATE":
		input.PaymentTermsDays = 0

	case "TERM":
		if input.PaymentTermsDays <= 0 {
			return nil, ErrSalesOrderInvalidPaymentTerms
		}

	default:
		return nil, ErrSalesOrderInvalidPaymentMode
	}

	paymentChannel :=
		strings.ToUpper(
			strings.TrimSpace(
				input.PaymentChannel,
			),
		)

	switch paymentChannel {
	case "CASH",
		"BANK_TRANSFER",
		"VIRTUAL_ACCOUNT",
		"QRIS",
		"E_WALLET",
		"CARD",
		"OTHER":
	default:
		return nil, ErrSalesOrderInvalidChannel
	}

	paymentProvider :=
		strings.TrimSpace(
			input.PaymentProvider,
		)

	var dueDate *time.Time

	if paymentSettlement == "TERM" {
		value :=
			input.OrderDate.AddDate(
				0,
				0,
				input.PaymentTermsDays,
			)
		dueDate = &value
	}

	if input.DiscountAmount < 0 {
		return nil, ErrSalesOrderInvalidDiscount
	}

	taxMode := strings.ToUpper(
		strings.TrimSpace(
			input.TaxMode,
		),
	)

	if taxMode == "" {
		taxMode = "NON_PPN"
	}

	var (
		taxProfileID     *uuid.UUID
		taxProfileName   string
		taxEffectiveYear int
		taxRate          float64
		effectiveTaxRate float64
		taxBaseFactor    float64
		taxScheme        string
	)

	switch taxMode {
	case "NON_PPN":
		taxScheme = "NON_PPN"

	case "STANDARD_PPN":
		var profile *database.TaxProfile

		if input.TaxProfileID != nil &&
			*input.TaxProfileID != uuid.Nil {
			profile, err =
				database.GetTaxProfileByID(
					ctx,
					s.pool,
					*input.TaxProfileID,
				)
		} else {
			profile, err =
				database.GetDefaultTaxProfile(
					ctx,
					s.pool,
				)
		}
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, ErrSalesOrderTaxProfileNotFound
		}

		value := profile.ID
		taxProfileID = &value
		taxProfileName = profile.Name
		taxEffectiveYear =
			profile.EffectiveYear
		taxRate =
			profile.NominalRate
		effectiveTaxRate =
			profile.EffectiveRate
		taxBaseFactor =
			profile.DPPFactor
		taxScheme =
			profile.ProfileCode

	default:
		return nil, ErrSalesOrderInvalidTaxScheme
	}

	tx, err := s.pool.BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	items := make(
		[]database.InsertSalesOrderItemInput,
		0,
		len(input.Items),
	)

	var (
		subtotal         float64
		clQty            int
		hasStockShortage bool
	)

	seenProducts := make(
		map[uuid.UUID]struct{},
		len(input.Items),
	)

	for _, requestedItem := range input.Items {
		if requestedItem.ProductID == uuid.Nil {
			return nil, ErrSalesOrderProductNotFound
		}

		if requestedItem.Quantity <= 0 {
			return nil, ErrSalesOrderInvalidQuantity
		}

		if _, exists :=
			seenProducts[requestedItem.ProductID]; exists {
			return nil, ErrSalesOrderDuplicateProduct
		}

		seenProducts[requestedItem.ProductID] =
			struct{}{}

		product, err := database.GetSalesOrderProduct(
			ctx,
			tx,
			requestedItem.ProductID,
		)
		if err != nil {
			return nil, err
		}

		if product == nil {
			return nil, ErrSalesOrderProductNotFound
		}

		if !product.IsActive {
			return nil, ErrSalesOrderProductInactive
		}

		itemSubtotal := roundMoney(
			product.Price *
				float64(requestedItem.Quantity),
		)

		if requestedItem.Quantity >
			product.Stock {
			hasStockShortage = true
		}

		subtotal += itemSubtotal
		clQty += requestedItem.Quantity

		items = append(
			items,
			database.InsertSalesOrderItemInput{
				ProductID:      product.ID,
				ProductCode:    product.ProductCode,
				SKU:            product.SKU,
				Barcode:        product.Barcode,
				ProductName:    product.Name,
				Unit:           product.Unit,
				Quantity:       requestedItem.Quantity,
				UnitPrice:      roundMoney(product.Price),
				DiscountAmount: 0,
				Subtotal:       itemSubtotal,
			},
		)
	}

	subtotal = roundMoney(
		subtotal,
	)

	if input.DiscountAmount > subtotal {
		return nil, ErrSalesOrderInvalidDiscount
	}

	discountAmount := roundMoney(
		input.DiscountAmount,
	)

	dssAmount := roundMoney(
		subtotal - discountAmount,
	)

	var (
		taxBaseAmount float64
		taxAmount     float64
	)

	if taxMode == "STANDARD_PPN" {
		taxBaseAmount = roundMoney(
			dssAmount *
				taxBaseFactor,
		)

		taxAmount = roundMoney(
			taxBaseAmount *
				taxRate /
				100,
		)
	}

	total := roundMoney(
		dssAmount + taxAmount,
	)

	cluster := TentukanPrioritas(
		dssAmount,
		float64(clQty),
	)

	priority := priorityFromCluster(
		cluster,
		clQty,
	)

	recommendation := recommendationFromPriority(
		cluster,
		priority,
	)

	orderID, err :=
		database.InsertSalesOrderDraft(
			ctx,
			tx,
			database.InsertSalesOrderInput{
				CustomerID:        input.CustomerID,
				WarehouseID:       warehouse.ID,
				WarehouseCode:     warehouse.WarehouseCode,
				WarehouseName:     warehouse.Name,
				WarehouseClass:    warehouse.Classification,
				OrderDate:         input.OrderDate,
				Notes:             input.Notes,
				PaymentTermsDays:  input.PaymentTermsDays,
				PaymentSettlement: paymentSettlement,
				PaymentChannel:    paymentChannel,
				PaymentProvider:   paymentProvider,
				DueDate:           dueDate,
				TaxMode:           taxMode,
				TaxProfileID:      taxProfileID,
				TaxProfileName:    taxProfileName,
				TaxEffectiveYear:  taxEffectiveYear,
				EffectiveTaxRate:  effectiveTaxRate,
				TaxBaseFactor:     taxBaseFactor,
				Subtotal:          subtotal,
				DiscountAmount:    discountAmount,
				TaxScheme:         taxScheme,
				TaxRate:           taxRate,
				TaxBaseAmount:     taxBaseAmount,
				TaxAmount:         taxAmount,
				Total:             total,
				ClQty:             clQty,
				DSSAmount:         dssAmount,
				Cluster:           cluster,
				Priority:          priority,
				Recommendation:    recommendation,
			},
			items,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	order, err :=
		database.GetSalesOrderByID(
			ctx,
			s.pool,
			orderID,
		)
	if err != nil {
		return nil, err
	}

	order.HasStockShortage =
		hasStockShortage

	return order, nil
}

func (s *SalesOrderService) GetAll(
	ctx context.Context,
) ([]database.SalesOrder, error) {
	return database.GetSalesOrders(
		ctx,
		s.pool,
	)
}

func (s *SalesOrderService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*database.SalesOrder, error) {
	return database.GetSalesOrderByID(
		ctx,
		s.pool,
		id,
	)
}

func (s *SalesOrderService) Confirm(
	ctx context.Context,
	id uuid.UUID,
) (*database.SalesOrder, error) {
	return s.transition(
		ctx,
		id,
		"DRAFT",
		"CONFIRMED",
	)
}

func (s *SalesOrderService) StartPicking(
	ctx context.Context,
	id uuid.UUID,
) (*database.SalesOrder, error) {
	return s.transition(
		ctx,
		id,
		"CONFIRMED",
		"PICKING",
	)
}

func (s *SalesOrderService) FinishPacking(
	ctx context.Context,
	id uuid.UUID,
) (*database.SalesOrder, error) {
	return s.transition(
		ctx,
		id,
		"PICKING",
		"PACKED",
	)
}

func (s *SalesOrderService) Ship(
	ctx context.Context,
	id uuid.UUID,
) (*database.SalesOrder, error) {
	shipped, err :=
		database.ShipSalesOrder(
			ctx,
			s.pool,
			id,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			database.ErrSalesOrderNotPacked,
		):
			return nil, fmt.Errorf(
				"%w: pesanan harus selesai dikemas sebelum dikirim",
				ErrSalesOrderInvalidTransition,
			)

		case errors.Is(
			err,
			database.ErrSalesOrderInsufficientStock,
		):
			detail :=
				strings.TrimPrefix(
					err.Error(),
					database.ErrSalesOrderInsufficientStock.Error()+": ",
				)

			return nil, fmt.Errorf(
				"%w: %s",
				ErrSalesOrderInsufficientStock,
				detail,
			)

		default:
			return nil, err
		}
	}

	if !shipped {
		return nil, ErrSalesOrderNotFound
	}

	order, err :=
		database.GetSalesOrderByID(
			ctx,
			s.pool,
			id,
		)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, ErrSalesOrderNotFound
	}

	return order, nil
}

func (s *SalesOrderService) transition(
	ctx context.Context,
	id uuid.UUID,
	expectedStatus string,
	nextStatus string,
) (*database.SalesOrder, error) {
	updated, err :=
		database.TransitionSalesOrderStatus(
			ctx,
			s.pool,
			id,
			nextStatus,
		)
	if err != nil {
		return nil, err
	}

	if !updated {
		order, getErr :=
			database.GetSalesOrderByID(
				ctx,
				s.pool,
				id,
			)
		if getErr != nil {
			return nil, getErr
		}

		if order == nil {
			return nil, ErrSalesOrderNotFound
		}

		return nil, fmt.Errorf(
			"%w: status saat ini %s, harus %s",
			ErrSalesOrderInvalidTransition,
			order.Status,
			expectedStatus,
		)
	}

	order, err :=
		database.GetSalesOrderByID(
			ctx,
			s.pool,
			id,
		)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, ErrSalesOrderNotFound
	}

	return order, nil
}

func priorityFromCluster(
	cluster string,
	clQty int,
) string {
	if cluster == "Klaster 2 (VVIP)" {
		return "VVIP"
	}

	if clQty >= 60 {
		return "HIGH_VOLUME"
	}

	if cluster == "Klaster 1 (Retail)" {
		return "RETAIL"
	}

	return "REGULAR"
}

func recommendationFromPriority(
	cluster string,
	priority string,
) string {
	switch priority {
	case "VVIP":
		return RekomendasiDistribusi(
			cluster,
		)

	case "HIGH_VOLUME":
		return "Pesanan volume besar. Prioritaskan kapasitas picking, packing, dan ruang kendaraan distribusi."

	case "RETAIL":
		return RekomendasiDistribusi(
			cluster,
		)

	default:
		return RekomendasiDistribusi(
			cluster,
		)
	}
}

func roundMoney(
	value float64,
) float64 {
	return math.Round(value*100) / 100
}
