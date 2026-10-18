package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvoiceSalesOrderNotFound = errors.New("sales order tidak ditemukan")
	ErrInvoiceOrderNotShipped    = errors.New("sales order belum dikirim")
	ErrInvoiceNotFound           = errors.New("faktur tidak ditemukan")
	ErrInvoiceItemNotFound       = errors.New("item faktur tidak ditemukan")
	ErrInvoiceHasPayments        = errors.New("faktur sudah memiliki catatan pembayaran atau tautan Midtrans aktif")
)

type InvoiceItem struct {
	ID              uuid.UUID  `json:"id"`
	InvoiceID       uuid.UUID  `json:"invoice_id"`
	ProductID       uuid.UUID  `json:"product_id"`
	ProductCode     string     `json:"product_code"`
	SKU             string     `json:"sku"`
	Barcode         string     `json:"barcode"`
	ProductName     string     `json:"product_name"`
	Unit            string     `json:"unit"`
	BatchNumber     string     `json:"batch_number"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	Qty             int        `json:"qty"`
	Price           float64    `json:"price"`
	DiscountPercent float64    `json:"discount_percent"`
	DiscountAmount  float64    `json:"discount_amount"`
	Subtotal        float64    `json:"subtotal"`
}

type Invoice struct {
	IssuerProfile      json.RawMessage `json:"issuer_profile"`
	ID                 uuid.UUID       `json:"id"`
	InvoiceNumber      string          `json:"invoice_number"`
	DeliveryNoteNumber string          `json:"delivery_note_number"`
	SalesOrderID       uuid.UUID       `json:"sales_order_id"`
	OrderNumber        string          `json:"order_number"`
	CustomerID         uuid.UUID       `json:"customer_id"`
	CustomerCode       string          `json:"customer_code"`
	StoreName          string          `json:"store_name"`
	CustomerPhone      string          `json:"customer_phone"`
	CustomerTaxID      string          `json:"customer_tax_id_number"`
	CustomerTaxName    string          `json:"customer_tax_name"`
	CustomerTaxAddress string          `json:"customer_tax_address"`
	ShippingAddress    string          `json:"shipping_address"`
	InvoiceDate        time.Time       `json:"invoice_date"`
	PaymentTermsDays   int             `json:"payment_terms_days"`
	PaymentSettlement  string          `json:"payment_settlement"`
	PaymentChannel     string          `json:"payment_channel"`
	PaymentProvider    string          `json:"payment_provider"`
	DueDate            *time.Time      `json:"due_date"`
	SalesName          string          `json:"sales_name"`
	WarehouseID        *uuid.UUID      `json:"warehouse_id"`
	WarehouseCode      string          `json:"warehouse_code"`
	WarehouseName      string          `json:"warehouse_name"`
	WarehouseClass     string          `json:"warehouse_classification"`
	WarehouseType      string          `json:"warehouse_type"`
	TaxMode            string          `json:"tax_mode"`
	TaxProfileID       *uuid.UUID      `json:"tax_profile_id"`
	TaxProfileName     string          `json:"tax_profile_name"`
	TaxEffectiveYear   int             `json:"tax_effective_year"`
	Subtotal           float64         `json:"subtotal"`
	DiscountAmount     float64         `json:"discount_amount"`
	TaxScheme          string          `json:"tax_scheme"`
	TaxRate            float64         `json:"tax_rate"`
	EffectiveTaxRate   float64         `json:"effective_tax_rate"`
	TaxBaseFactor      float64         `json:"tax_base_factor"`
	TaxBaseAmount      float64         `json:"tax_base_amount"`
	TaxAmount          float64         `json:"tax_amount"`
	Total              float64         `json:"total"`
	ClQty              int             `json:"cl_qty"`
	Cluster            string          `json:"cluster"`
	Priority           string          `json:"priority"`
	Recommendation     string          `json:"recommendation"`
	Status             string          `json:"status"`
	BarcodeValue       string          `json:"barcode_value"`
	Notes              string          `json:"notes"`
	IsPrinted          bool            `json:"is_printed"`
	PrintCount         int             `json:"print_count"`
	PrintVersion       int             `json:"print_version"`
	PrintedAt          *time.Time      `json:"printed_at"`
	LastEditedAt       *time.Time      `json:"last_edited_at"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Items              []InvoiceItem   `json:"items"`
}

type invoiceSalesOrderSnapshot struct {
	ID                 uuid.UUID
	OrderNumber        string
	CustomerID         uuid.UUID
	CustomerCode       string
	StoreName          string
	CustomerPhone      string
	CustomerTaxID      string
	CustomerTaxName    string
	CustomerTaxAddress string
	ShippingAddress    string
	WarehouseID        *uuid.UUID
	WarehouseCode      string
	WarehouseName      string
	WarehouseClass     string
	Status             string
	ShippedAt          *time.Time
	PaymentTermsDays   int
	PaymentSettlement  string
	PaymentChannel     string
	PaymentProvider    string
	DueDate            *time.Time
	TaxMode            string
	TaxProfileID       *uuid.UUID
	TaxProfileName     string
	TaxEffectiveYear   int
	TaxScheme          string
	TaxRate            float64
	EffectiveTaxRate   float64
	TaxBaseFactor      float64
	TaxBaseAmount      float64
	TaxAmount          float64
	Total              float64
	Subtotal           float64
	DiscountAmount     float64
	ClQty              int
	Cluster            string
	Priority           string
	Recommendation     string
}

type UpdateInvoiceItemInput struct {
	ID              uuid.UUID
	BatchNumber     string
	ExpiryDate      *time.Time
	DiscountPercent float64
	DiscountAmount  float64
	Subtotal        float64
}

type UpdateInvoiceInput struct {
	SalesName         string
	WarehouseID       uuid.UUID
	WarehouseCode     string
	WarehouseName     string
	WarehouseClass    string
	PaymentTermsDays  int
	PaymentSettlement string
	PaymentChannel    string
	PaymentProvider   string
	DueDate           *time.Time
	TaxMode           string
	TaxProfileID      *uuid.UUID
	TaxProfileName    string
	TaxEffectiveYear  int
	Subtotal          float64
	DiscountAmount    float64
	TaxScheme         string
	TaxRate           float64
	EffectiveTaxRate  float64
	TaxBaseFactor     float64
	TaxBaseAmount     float64
	TaxAmount         float64
	Total             float64
	Notes             string
	Items             []UpdateInvoiceItemInput
}

func GetInvoiceBySalesOrderID(
	ctx context.Context,
	pool *pgxpool.Pool,
	salesOrderID uuid.UUID,
) (*Invoice, error) {
	return getInvoice(
		ctx,
		pool,
		`
		WHERE sales_order_id = $1
		LIMIT 1
		`,
		salesOrderID,
	)
}

func GetInvoiceByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Invoice, error) {
	return getInvoice(
		ctx,
		pool,
		`
		WHERE id = $1
		LIMIT 1
		`,
		id,
	)
}

func getInvoice(
	ctx context.Context,
	pool *pgxpool.Pool,
	whereClause string,
	arg any,
) (*Invoice, error) {
	var invoice Invoice

	query := `
		SELECT
			COALESCE(issuer_profile,(SELECT TO_JSONB(c)-'id'-'updated_by' FROM company_profile c WHERE id=TRUE)),
			id,
			invoice_number,
			COALESCE(delivery_note_number, ''),
			sales_order_id,
			COALESCE(order_number, ''),
			customer_id,
			COALESCE(customer_code, ''),
			store_name,
			COALESCE(customer_phone, ''),
			COALESCE(customer_tax_id_number, ''),
			COALESCE(customer_tax_name, ''),
			COALESCE(customer_tax_address, ''),
			COALESCE(shipping_address, ''),
			invoice_date,
			payment_terms_days,
			COALESCE(payment_settlement, 'IMMEDIATE'),
			COALESCE(payment_channel, 'UNSPECIFIED'),
			COALESCE(payment_provider, ''),
			due_date,
			COALESCE(sales_name, ''),
			warehouse_id,
			COALESCE(warehouse_code, ''),
			COALESCE(warehouse_name, ''),
			COALESCE(warehouse_classification, ''),
			COALESCE(warehouse_type, 'REGULER'),
			COALESCE(tax_mode, 'STANDARD_PPN'),
			tax_profile_id,
			COALESCE(tax_profile_name, ''),
			COALESCE(tax_effective_year, 0),
			subtotal,
			discount_amount,
			COALESCE(tax_scheme, 'PPN_12_DPP_11_12'),
			ppn_percent,
			effective_tax_rate,
			tax_base_factor,
			tax_base_amount,
			ppn_amount,
			total,
			cl_qty,
			COALESCE(cluster, ''),
			COALESCE(priority, ''),
			COALESCE(recommendation, ''),
			status,
			COALESCE(barcode_value, invoice_number),
			COALESCE(notes, ''),
			is_printed,
			print_count,
			print_version,
			printed_at,
			last_edited_at,
			created_at,
			updated_at
		FROM invoices
	` + whereClause

	err := pool.QueryRow(
		ctx,
		query,
		arg,
	).Scan(
		&invoice.IssuerProfile,
		&invoice.ID,
		&invoice.InvoiceNumber,
		&invoice.DeliveryNoteNumber,
		&invoice.SalesOrderID,
		&invoice.OrderNumber,
		&invoice.CustomerID,
		&invoice.CustomerCode,
		&invoice.StoreName,
		&invoice.CustomerPhone,
		&invoice.CustomerTaxID,
		&invoice.CustomerTaxName,
		&invoice.CustomerTaxAddress,
		&invoice.ShippingAddress,
		&invoice.InvoiceDate,
		&invoice.PaymentTermsDays,
		&invoice.PaymentSettlement,
		&invoice.PaymentChannel,
		&invoice.PaymentProvider,
		&invoice.DueDate,
		&invoice.SalesName,
		&invoice.WarehouseID,
		&invoice.WarehouseCode,
		&invoice.WarehouseName,
		&invoice.WarehouseClass,
		&invoice.WarehouseType,
		&invoice.TaxMode,
		&invoice.TaxProfileID,
		&invoice.TaxProfileName,
		&invoice.TaxEffectiveYear,
		&invoice.Subtotal,
		&invoice.DiscountAmount,
		&invoice.TaxScheme,
		&invoice.TaxRate,
		&invoice.EffectiveTaxRate,
		&invoice.TaxBaseFactor,
		&invoice.TaxBaseAmount,
		&invoice.TaxAmount,
		&invoice.Total,
		&invoice.ClQty,
		&invoice.Cluster,
		&invoice.Priority,
		&invoice.Recommendation,
		&invoice.Status,
		&invoice.BarcodeValue,
		&invoice.Notes,
		&invoice.IsPrinted,
		&invoice.PrintCount,
		&invoice.PrintVersion,
		&invoice.PrintedAt,
		&invoice.LastEditedAt,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	items, err := getInvoiceItems(
		ctx,
		pool,
		invoice.ID,
	)
	if err != nil {
		return nil, err
	}

	invoice.Items = items

	return &invoice, nil
}

func EnsureInvoiceForSalesOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	salesOrderID uuid.UUID,
) (*Invoice, error) {
	existing, err := GetInvoiceBySalesOrderID(
		ctx,
		pool,
		salesOrderID,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	tx, err := pool.BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var snapshot invoiceSalesOrderSnapshot

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			so.id,
			so.order_number,
			so.customer_id,
			c.customer_code,
			c.name,
			COALESCE(c.phone, ''),
			COALESCE(c.tax_id_number, ''),
			COALESCE(c.tax_name, ''),
			COALESCE(c.tax_address, ''),
			COALESCE(c.shipping_address, ''),
			so.warehouse_id,
			COALESCE(so.warehouse_code_snapshot, ''),
			COALESCE(so.warehouse_name_snapshot, ''),
			COALESCE(so.warehouse_classification_snapshot, ''),
			so.status,
			so.shipped_at,
			so.payment_terms_days,
			COALESCE(so.payment_settlement, 'IMMEDIATE'),
			COALESCE(so.payment_channel, 'UNSPECIFIED'),
			COALESCE(so.payment_provider, ''),
			so.due_date,
			COALESCE(so.tax_mode, 'NON_PPN'),
			so.tax_profile_id,
			COALESCE(so.tax_profile_name_snapshot, ''),
			COALESCE(so.tax_effective_year, 0),
			COALESCE(so.tax_scheme, 'NON_PPN'),
			COALESCE(so.tax_rate, 0),
			COALESCE(so.effective_tax_rate, 0),
			COALESCE(so.tax_base_factor, 0),
			COALESCE(so.tax_base_amount, 0),
			COALESCE(so.tax_amount, 0),
			COALESCE(so.total, 0),
			so.subtotal,
			so.discount_amount,
			so.cl_qty,
			COALESCE(so.cluster, ''),
			COALESCE(so.priority, ''),
			COALESCE(so.recommendation, '')
		FROM sales_orders so
		JOIN customers c
			ON c.id = so.customer_id
		WHERE so.id = $1
		FOR UPDATE OF so
		`,
		salesOrderID,
	).Scan(
		&snapshot.ID,
		&snapshot.OrderNumber,
		&snapshot.CustomerID,
		&snapshot.CustomerCode,
		&snapshot.StoreName,
		&snapshot.CustomerPhone,
		&snapshot.CustomerTaxID,
		&snapshot.CustomerTaxName,
		&snapshot.CustomerTaxAddress,
		&snapshot.ShippingAddress,
		&snapshot.WarehouseID,
		&snapshot.WarehouseCode,
		&snapshot.WarehouseName,
		&snapshot.WarehouseClass,
		&snapshot.Status,
		&snapshot.ShippedAt,
		&snapshot.PaymentTermsDays,
		&snapshot.PaymentSettlement,
		&snapshot.PaymentChannel,
		&snapshot.PaymentProvider,
		&snapshot.DueDate,
		&snapshot.TaxMode,
		&snapshot.TaxProfileID,
		&snapshot.TaxProfileName,
		&snapshot.TaxEffectiveYear,
		&snapshot.TaxScheme,
		&snapshot.TaxRate,
		&snapshot.EffectiveTaxRate,
		&snapshot.TaxBaseFactor,
		&snapshot.TaxBaseAmount,
		&snapshot.TaxAmount,
		&snapshot.Total,
		&snapshot.Subtotal,
		&snapshot.DiscountAmount,
		&snapshot.ClQty,
		&snapshot.Cluster,
		&snapshot.Priority,
		&snapshot.Recommendation,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceSalesOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	if snapshot.Status != "PACKED" &&
		snapshot.Status != "SHIPPED" &&
		snapshot.Status != "COMPLETED" {
		return nil, fmt.Errorf(
			"%w: status saat ini %s",
			ErrInvoiceOrderNotShipped,
			snapshot.Status,
		)
	}

	var existingID uuid.UUID

	err = tx.QueryRow(
		ctx,
		`
		SELECT id
		FROM invoices
		WHERE sales_order_id = $1
		LIMIT 1
		FOR UPDATE
		`,
		salesOrderID,
	).Scan(&existingID)

	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return GetInvoiceBySalesOrderID(
			ctx,
			pool,
			salesOrderID,
		)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	invoiceDate := time.Now()
	if snapshot.ShippedAt != nil {
		invoiceDate = *snapshot.ShippedAt
	}

	var (
		invoiceNumber      string
		deliveryNoteNumber string
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			'INV-' ||
				to_char($1::date, 'YYYYMMDD') ||
				'-' ||
				lpad(
					nextval('invoices_invoice_number_seq')::text,
					6,
					'0'
				),
			'SJ-' ||
				to_char($1::date, 'YYYYMMDD') ||
				'-' ||
				lpad(
					nextval('delivery_notes_number_seq')::text,
					6,
					'0'
				)
		`,
		invoiceDate,
	).Scan(
		&invoiceNumber,
		&deliveryNoteNumber,
	)
	if err != nil {
		return nil, err
	}

	warehouseType := snapshot.WarehouseClass
	if warehouseType == "" {
		warehouseType = "REGULER"
	}

	dueDate := snapshot.DueDate

	var invoiceID uuid.UUID

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO invoices (
			invoice_number,
			store_name,
			invoice_date,
			subtotal,
			ppn_percent,
			ppn_amount,
			total,
			cl_qty,
			cluster,
			recommendation,
			status,
			sales_order_id,
			customer_id,
			customer_code,
			order_number,
			shipping_address,
			customer_phone,
			customer_tax_id_number,
			customer_tax_name,
			customer_tax_address,
			payment_terms_days,
			payment_settlement,
			payment_channel,
			payment_provider,
			due_date,
			discount_amount,
			tax_scheme,
			tax_base_amount,
			priority,
			delivery_note_number,
			barcode_value,
			sales_name,
			warehouse_id,
			warehouse_code,
			warehouse_name,
			warehouse_classification,
			warehouse_type,
			tax_mode,
			tax_profile_id,
			tax_profile_name,
			tax_effective_year,
			effective_tax_rate,
			tax_base_factor,
			notes,
			is_printed,
			print_count,
			print_version,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			NULLIF($10, ''),
			'UNPAID',
			$11,
			$12,
			$13,
			$14,
			NULLIF($15, ''),
			NULLIF($16, ''),
			NULLIF($17, ''),
			NULLIF($18, ''),
			NULLIF($19, ''),
			$20,
			$21,
			$22,
			NULLIF($23, ''),
			$24,
			$25,
			$26,
			$27,
			NULLIF($28, ''),
			$29,
			$30,
			'',
			$31,
			NULLIF($32, ''),
			NULLIF($33, ''),
			NULLIF($34, ''),
			$35,
			$36,
			$37,
			NULLIF($38, ''),
			$39,
			$40,
			$41,
			NULL,
			FALSE,
			0,
			1,
			NOW()
		)
		RETURNING id
		`,
		invoiceNumber,
		snapshot.StoreName,
		invoiceDate,
		snapshot.Subtotal,
		snapshot.TaxRate,
		snapshot.TaxAmount,
		snapshot.Total,
		snapshot.ClQty,
		snapshot.Cluster,
		snapshot.Recommendation,
		salesOrderID,
		snapshot.CustomerID,
		snapshot.CustomerCode,
		snapshot.OrderNumber,
		snapshot.ShippingAddress,
		snapshot.CustomerPhone,
		snapshot.CustomerTaxID,
		snapshot.CustomerTaxName,
		snapshot.CustomerTaxAddress,
		snapshot.PaymentTermsDays,
		snapshot.PaymentSettlement,
		snapshot.PaymentChannel,
		snapshot.PaymentProvider,
		dueDate,
		snapshot.DiscountAmount,
		snapshot.TaxScheme,
		snapshot.TaxBaseAmount,
		snapshot.Priority,
		deliveryNoteNumber,
		invoiceNumber,
		snapshot.WarehouseID,
		snapshot.WarehouseCode,
		snapshot.WarehouseName,
		snapshot.WarehouseClass,
		warehouseType,
		snapshot.TaxMode,
		snapshot.TaxProfileID,
		snapshot.TaxProfileName,
		snapshot.TaxEffectiveYear,
		snapshot.EffectiveTaxRate,
		snapshot.TaxBaseFactor,
	).Scan(&invoiceID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO invoice_items (
			invoice_id,
			product_id,
			qty,
			price,
			subtotal,
			product_code,
			sku,
			barcode,
			product_name,
			unit,
			discount_percent,
			discount_amount,
			batch_number,
			expiry_date
		)
		SELECT
			$1,
			soi.product_id,
			soi.quantity,
			soi.unit_price,
			soi.subtotal,
			soi.product_code,
			soi.sku,
			NULLIF(soi.barcode, ''),
			soi.product_name,
			soi.unit,
			0,
			soi.discount_amount,
			NULL,
			NULL
		FROM sales_order_items soi
		WHERE soi.sales_order_id = $2
		ORDER BY soi.created_at ASC, soi.id ASC
		`,
		invoiceID,
		salesOrderID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return GetInvoiceBySalesOrderID(
		ctx,
		pool,
		salesOrderID,
	)
}

func UpdateInvoice(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	input UpdateInvoiceInput,
) (*Invoice, error) {
	tx, err := pool.BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var (
		isPrinted    bool
		printVersion int
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			is_printed,
			print_version
		FROM invoices
		WHERE id = $1
		FOR UPDATE
		`,
		id,
	).Scan(
		&isPrinted,
		&printVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}

	var paymentExists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoice_payments WHERE invoice_id=$1)
        OR EXISTS(SELECT 1 FROM finance_payment_intents WHERE invoice_id=$1 AND source='MIDTRANS' AND state<>'RELEASED')`, id).Scan(&paymentExists)
	if err != nil {
		return nil, err
	}
	if paymentExists {
		return nil, ErrInvoiceHasPayments
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO invoice_revisions (
			invoice_id,
			revision_number,
			reason,
			previous_data
		)
		SELECT
			i.id,
			i.print_version,
			CASE
				WHEN i.is_printed
					THEN 'Revisi setelah cetak'
				ELSE 'Perubahan sebelum cetak'
			END,
			jsonb_build_object(
				'invoice',
				to_jsonb(i),
				'items',
				COALESCE(
					(
						SELECT jsonb_agg(
							to_jsonb(ii)
							ORDER BY ii.id
						)
						FROM invoice_items ii
						WHERE ii.invoice_id = i.id
					),
					'[]'::jsonb
				)
			)
		FROM invoices i
		WHERE i.id = $1
		`,
		id,
	)
	if err != nil {
		return nil, err
	}

	nextVersion := printVersion
	if isPrinted {
		nextVersion++
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE invoices
		SET
			sales_name = NULLIF($2, ''),
			warehouse_id = $3,
			warehouse_code = $4,
			warehouse_name = $5,
			warehouse_classification = $6,
			warehouse_type = $6,
			payment_terms_days = $7,
			payment_settlement = $8,
			payment_channel = $9,
			payment_provider = NULLIF($10, ''),
			due_date = $11,
			tax_mode = $12,
			tax_profile_id = $13,
			tax_profile_name = NULLIF($14, ''),
			tax_effective_year = $15,
			subtotal = $16,
			discount_amount = $17,
			tax_scheme = $18,
			ppn_percent = $19,
			effective_tax_rate = $20,
			tax_base_factor = $21,
			tax_base_amount = $22,
			ppn_amount = $23,
			total = $24,
			notes = NULLIF($25, ''),
			print_version = $26,
			last_edited_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		`,
		id,
		strings.TrimSpace(input.SalesName),
		input.WarehouseID,
		input.WarehouseCode,
		input.WarehouseName,
		input.WarehouseClass,
		input.PaymentTermsDays,
		input.PaymentSettlement,
		input.PaymentChannel,
		strings.TrimSpace(input.PaymentProvider),
		input.DueDate,
		input.TaxMode,
		input.TaxProfileID,
		input.TaxProfileName,
		input.TaxEffectiveYear,
		input.Subtotal,
		input.DiscountAmount,
		input.TaxScheme,
		input.TaxRate,
		input.EffectiveTaxRate,
		input.TaxBaseFactor,
		input.TaxBaseAmount,
		input.TaxAmount,
		input.Total,
		strings.TrimSpace(input.Notes),
		nextVersion,
	)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected() != 1 {
		return nil, ErrInvoiceNotFound
	}

	for _, item := range input.Items {
		result, err := tx.Exec(
			ctx,
			`
			UPDATE invoice_items
			SET
				batch_number = NULLIF($3, ''),
				expiry_date = $4,
				discount_percent = $5,
				discount_amount = $6,
				subtotal = $7
			WHERE id = $1
			  AND invoice_id = $2
			`,
			item.ID,
			id,
			strings.TrimSpace(item.BatchNumber),
			item.ExpiryDate,
			item.DiscountPercent,
			item.DiscountAmount,
			item.Subtotal,
		)
		if err != nil {
			return nil, err
		}

		if result.RowsAffected() != 1 {
			return nil, ErrInvoiceItemNotFound
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return GetInvoiceByID(
		ctx,
		pool,
		id,
	)
}

func MarkInvoicePrinted(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Invoice, error) {
	result, err := pool.Exec(
		ctx,
		`
		UPDATE invoices
		SET
			is_printed = TRUE,
			print_count = print_count + 1,
			printed_at = COALESCE(printed_at, NOW()),
			updated_at = NOW()
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected() != 1 {
		return nil, ErrInvoiceNotFound
	}

	return GetInvoiceByID(
		ctx,
		pool,
		id,
	)
}

func getInvoiceItems(
	ctx context.Context,
	pool *pgxpool.Pool,
	invoiceID uuid.UUID,
) ([]InvoiceItem, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			invoice_id,
			product_id,
			COALESCE(product_code, ''),
			COALESCE(sku, ''),
			COALESCE(barcode, ''),
			COALESCE(product_name, ''),
			COALESCE(unit, ''),
			COALESCE(batch_number, ''),
			expiry_date,
			qty,
			price,
			discount_percent,
			discount_amount,
			subtotal
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY id ASC
		`,
		invoiceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(
		[]InvoiceItem,
		0,
	)

	for rows.Next() {
		var item InvoiceItem

		if err := rows.Scan(
			&item.ID,
			&item.InvoiceID,
			&item.ProductID,
			&item.ProductCode,
			&item.SKU,
			&item.Barcode,
			&item.ProductName,
			&item.Unit,
			&item.BatchNumber,
			&item.ExpiryDate,
			&item.Qty,
			&item.Price,
			&item.DiscountPercent,
			&item.DiscountAmount,
			&item.Subtotal,
		); err != nil {
			return nil, err
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func roundMoney(
	value float64,
) float64 {
	return math.Round(value)
}
