package services

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvoiceOrderNotFound       = errors.New("pesanan tidak ditemukan")
	ErrInvoiceOrderNotShipped     = errors.New("pesanan belum siap dibuatkan faktur")
	ErrInvoiceNotFound            = errors.New("faktur tidak ditemukan")
	ErrInvoiceInvalidWarehouse    = errors.New("gudang tidak valid")
	ErrInvoiceWarehouseInactive   = errors.New("gudang sedang nonaktif")
	ErrInvoiceInvalidTaxMode      = errors.New("perlakuan pajak tidak valid")
	ErrInvoiceTaxProfileNotFound  = errors.New("aturan PPN tidak ditemukan")
	ErrInvoiceInvalidPaymentTerms = errors.New("tempo pembayaran tidak valid")
	ErrInvoiceInvalidPaymentMode  = errors.New("skema pembayaran tidak valid")
	ErrInvoiceInvalidChannel      = errors.New("kanal pembayaran tidak valid")
	ErrInvoiceInvalidDiscount     = errors.New("diskon tidak valid")
)

type InvoiceService struct {
	pool *pgxpool.Pool
}

type UpdateInvoiceItemInput struct {
	ID              uuid.UUID
	BatchNumber     string
	ExpiryDate      *time.Time
	DiscountPercent float64
}

type UpdateInvoiceInput struct {
	SalesName         string
	WarehouseID       uuid.UUID
	PaymentTermsDays  int
	PaymentSettlement string
	PaymentChannel    string
	PaymentProvider   string
	TaxMode           string
	TaxProfileID      *uuid.UUID
	Notes             string
	Items             []UpdateInvoiceItemInput
}

func NewInvoiceService(
	pool *pgxpool.Pool,
) *InvoiceService {
	return &InvoiceService{
		pool: pool,
	}
}

func (s *InvoiceService) GetBySalesOrderID(
	ctx context.Context,
	salesOrderID uuid.UUID,
) (*database.Invoice, error) {
	return database.GetInvoiceBySalesOrderID(
		ctx,
		s.pool,
		salesOrderID,
	)
}

func (s *InvoiceService) EnsureForSalesOrder(
	ctx context.Context,
	salesOrderID uuid.UUID,
) (*database.Invoice, error) {
	invoice, err :=
		database.EnsureInvoiceForSalesOrder(
			ctx,
			s.pool,
			salesOrderID,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			database.ErrInvoiceSalesOrderNotFound,
		):
			return nil, ErrInvoiceOrderNotFound

		case errors.Is(
			err,
			database.ErrInvoiceOrderNotShipped,
		):
			return nil, ErrInvoiceOrderNotShipped

		default:
			return nil, err
		}
	}

	return invoice, nil
}

func (s *InvoiceService) Update(
	ctx context.Context,
	id uuid.UUID,
	input UpdateInvoiceInput,
) (*database.Invoice, error) {
	current, err := database.GetInvoiceByID(
		ctx,
		s.pool,
		id,
	)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrInvoiceNotFound
	}

	if input.WarehouseID == uuid.Nil {
		return nil, ErrInvoiceInvalidWarehouse
	}

	warehouse, err :=
		database.GetWarehouseByID(
			ctx,
			s.pool,
			input.WarehouseID,
		)
	if err != nil {
		return nil, err
	}
	if warehouse == nil {
		return nil, ErrInvoiceInvalidWarehouse
	}
	if !warehouse.IsActive {
		return nil, ErrInvoiceWarehouseInactive
	}

	if input.PaymentTermsDays < 0 ||
		input.PaymentTermsDays > 365 {
		return nil, ErrInvoiceInvalidPaymentTerms
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
			return nil, ErrInvoiceInvalidPaymentTerms
		}

	default:
		return nil, ErrInvoiceInvalidPaymentMode
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
		return nil, ErrInvoiceInvalidChannel
	}

	paymentProvider :=
		strings.TrimSpace(
			input.PaymentProvider,
		)

	taxMode := strings.ToUpper(
		strings.TrimSpace(
			input.TaxMode,
		),
	)

	var (
		taxProfileID     *uuid.UUID
		taxProfileName   string
		taxEffectiveYear int
		taxScheme        string
		taxRate          float64
		effectiveTaxRate float64
		taxBaseFactor    float64
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
			return nil, ErrInvoiceTaxProfileNotFound
		}

		value := profile.ID
		taxProfileID = &value
		taxProfileName = profile.Name
		taxEffectiveYear =
			profile.EffectiveYear
		taxScheme =
			profile.ProfileCode
		taxRate =
			profile.NominalRate
		effectiveTaxRate =
			profile.EffectiveRate
		taxBaseFactor =
			profile.DPPFactor

	default:
		return nil, ErrInvoiceInvalidTaxMode
	}

	itemByID := make(
		map[uuid.UUID]UpdateInvoiceItemInput,
		len(input.Items),
	)

	for _, item := range input.Items {
		if item.ID == uuid.Nil {
			continue
		}

		if item.DiscountPercent < 0 ||
			item.DiscountPercent > 100 {
			return nil, ErrInvoiceInvalidDiscount
		}

		itemByID[item.ID] = item
	}

	var (
		subtotal       float64
		discountAmount float64
		updateItems    []database.UpdateInvoiceItemInput
	)

	updateItems = make(
		[]database.UpdateInvoiceItemInput,
		0,
		len(current.Items),
	)

	for _, currentItem := range current.Items {
		requested, exists :=
			itemByID[currentItem.ID]

		discountPercent :=
			currentItem.DiscountPercent

		batchNumber :=
			currentItem.BatchNumber

		expiryDate :=
			currentItem.ExpiryDate

		if exists {
			discountPercent =
				requested.DiscountPercent
			batchNumber =
				strings.TrimSpace(
					requested.BatchNumber,
				)
			expiryDate =
				requested.ExpiryDate
		}

		gross := roundInvoiceMoney(
			currentItem.Price *
				float64(currentItem.Qty),
		)

		discount := roundInvoiceMoney(
			gross *
				discountPercent /
				100,
		)

		lineTotal := roundInvoiceMoney(
			gross -
				discount,
		)

		subtotal += gross
		discountAmount += discount

		updateItems = append(
			updateItems,
			database.UpdateInvoiceItemInput{
				ID:              currentItem.ID,
				BatchNumber:     batchNumber,
				ExpiryDate:      expiryDate,
				DiscountPercent: discountPercent,
				DiscountAmount:  discount,
				Subtotal:        lineTotal,
			},
		)
	}

	subtotal = roundInvoiceMoney(
		subtotal,
	)

	discountAmount =
		roundInvoiceMoney(
			discountAmount,
		)

	netAmount := roundInvoiceMoney(
		subtotal -
			discountAmount,
	)
	if netAmount < 0 {
		netAmount = 0
	}

	var (
		taxBaseAmount float64
		taxAmount     float64
		total         float64
	)

	if taxMode == "STANDARD_PPN" {
		taxBaseAmount =
			roundInvoiceMoney(
				netAmount *
					taxBaseFactor,
			)

		taxAmount =
			roundInvoiceMoney(
				taxBaseAmount *
					taxRate /
					100,
			)
	}

	total =
		roundInvoiceMoney(
			netAmount +
				taxAmount,
		)

	var dueDate *time.Time

	if paymentSettlement == "TERM" {
		value :=
			current.InvoiceDate.AddDate(
				0,
				0,
				input.PaymentTermsDays,
			)
		dueDate = &value
	}

	return database.UpdateInvoice(
		ctx,
		s.pool,
		id,
		database.UpdateInvoiceInput{
			SalesName: strings.TrimSpace(
				input.SalesName,
			),
			WarehouseID:       warehouse.ID,
			WarehouseCode:     warehouse.WarehouseCode,
			WarehouseName:     warehouse.Name,
			WarehouseClass:    warehouse.Classification,
			PaymentTermsDays:  input.PaymentTermsDays,
			PaymentSettlement: paymentSettlement,
			PaymentChannel:    paymentChannel,
			PaymentProvider:   paymentProvider,
			DueDate:           dueDate,
			TaxMode:           taxMode,
			TaxProfileID:      taxProfileID,
			TaxProfileName:    taxProfileName,
			TaxEffectiveYear:  taxEffectiveYear,
			Subtotal:          subtotal,
			DiscountAmount:    discountAmount,
			TaxScheme:         taxScheme,
			TaxRate:           taxRate,
			EffectiveTaxRate:  effectiveTaxRate,
			TaxBaseFactor:     taxBaseFactor,
			TaxBaseAmount:     taxBaseAmount,
			TaxAmount:         taxAmount,
			Total:             total,
			Notes: strings.TrimSpace(
				input.Notes,
			),
			Items: updateItems,
		},
	)
}

func (s *InvoiceService) MarkPrinted(
	ctx context.Context,
	id uuid.UUID,
) (*database.Invoice, error) {
	invoice, err :=
		database.MarkInvoicePrinted(
			ctx,
			s.pool,
			id,
		)
	if errors.Is(
		err,
		database.ErrInvoiceNotFound,
	) {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}

	return invoice, nil
}

func roundInvoiceMoney(
	value float64,
) float64 {
	return math.Round(value)
}
