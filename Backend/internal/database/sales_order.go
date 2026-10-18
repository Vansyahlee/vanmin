package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSalesOrderNotPacked         = errors.New("sales order belum siap dikirim")
	ErrSalesOrderInsufficientStock = errors.New("stok tidak mencukupi")
)

type SalesOrderCustomer struct {
	ID           uuid.UUID `json:"id"`
	CustomerCode string    `json:"customer_code"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
}

type SalesOrderWarehouse struct {
	ID             uuid.UUID `json:"id"`
	WarehouseCode  string    `json:"warehouse_code"`
	Name           string    `json:"name"`
	Classification string    `json:"classification"`
	IsActive       bool      `json:"is_active"`
}

type SalesOrderProduct struct {
	ID          uuid.UUID `json:"id"`
	ProductCode string    `json:"product_code"`
	SKU         string    `json:"sku"`
	Barcode     string    `json:"barcode"`
	Name        string    `json:"name"`
	Unit        string    `json:"unit"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	IsActive    bool      `json:"is_active"`
}

type SalesOrderItem struct {
	ID             uuid.UUID `json:"id"`
	SalesOrderID   uuid.UUID `json:"sales_order_id"`
	ProductID      uuid.UUID `json:"product_id"`
	ProductCode    string    `json:"product_code"`
	SKU            string    `json:"sku"`
	Barcode        string    `json:"barcode"`
	ProductName    string    `json:"product_name"`
	Unit           string    `json:"unit"`
	Quantity       int       `json:"quantity"`
	UnitPrice      float64   `json:"unit_price"`
	DiscountAmount float64   `json:"discount_amount"`
	Subtotal       float64   `json:"subtotal"`
	CreatedAt      time.Time `json:"created_at"`
}

type SalesOrder struct {
	ID                uuid.UUID        `json:"id"`
	OrderNumber       string           `json:"order_number"`
	CustomerID        uuid.UUID        `json:"customer_id"`
	CustomerCode      string           `json:"customer_code"`
	CustomerName      string           `json:"customer_name"`
	WarehouseID       *uuid.UUID       `json:"warehouse_id"`
	WarehouseCode     string           `json:"warehouse_code"`
	WarehouseName     string           `json:"warehouse_name"`
	WarehouseClass    string           `json:"warehouse_classification"`
	OrderDate         time.Time        `json:"order_date"`
	Status            string           `json:"status"`
	Notes             string           `json:"notes"`
	PaymentTermsDays  int              `json:"payment_terms_days"`
	PaymentSettlement string           `json:"payment_settlement"`
	PaymentChannel    string           `json:"payment_channel"`
	PaymentProvider   string           `json:"payment_provider"`
	DueDate           *time.Time       `json:"due_date"`
	TaxMode           string           `json:"tax_mode"`
	TaxProfileID      *uuid.UUID       `json:"tax_profile_id"`
	TaxProfileName    string           `json:"tax_profile_name"`
	TaxEffectiveYear  int              `json:"tax_effective_year"`
	EffectiveTaxRate  float64          `json:"effective_tax_rate"`
	TaxBaseFactor     float64          `json:"tax_base_factor"`
	Subtotal          float64          `json:"subtotal"`
	DiscountAmount    float64          `json:"discount_amount"`
	TaxScheme         string           `json:"tax_scheme"`
	TaxRate           float64          `json:"tax_rate"`
	TaxBaseAmount     float64          `json:"tax_base_amount"`
	TaxAmount         float64          `json:"tax_amount"`
	Total             float64          `json:"total"`
	ClQty             int              `json:"cl_qty"`
	DSSAmount         float64          `json:"dss_amount"`
	Cluster           string           `json:"cluster"`
	Priority          string           `json:"priority"`
	Recommendation    string           `json:"recommendation"`
	ConfirmedAt       *time.Time       `json:"confirmed_at"`
	PickingAt         *time.Time       `json:"picking_at"`
	PackedAt          *time.Time       `json:"packed_at"`
	ShippedAt         *time.Time       `json:"shipped_at"`
	CompletedAt       *time.Time       `json:"completed_at"`
	CancelledAt       *time.Time       `json:"cancelled_at"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	Items             []SalesOrderItem `json:"items,omitempty"`
	HasStockShortage  bool             `json:"has_stock_shortage"`
}

type InsertSalesOrderInput struct {
	CustomerID        uuid.UUID
	WarehouseID       uuid.UUID
	WarehouseCode     string
	WarehouseName     string
	WarehouseClass    string
	OrderDate         time.Time
	Notes             string
	PaymentTermsDays  int
	PaymentSettlement string
	PaymentChannel    string
	PaymentProvider   string
	DueDate           *time.Time
	TaxMode           string
	TaxProfileID      *uuid.UUID
	TaxProfileName    string
	TaxEffectiveYear  int
	EffectiveTaxRate  float64
	TaxBaseFactor     float64
	Subtotal          float64
	DiscountAmount    float64
	TaxScheme         string
	TaxRate           float64
	TaxBaseAmount     float64
	TaxAmount         float64
	Total             float64
	ClQty             int
	DSSAmount         float64
	Cluster           string
	Priority          string
	Recommendation    string
}

type InsertSalesOrderItemInput struct {
	ProductID      uuid.UUID
	ProductCode    string
	SKU            string
	Barcode        string
	ProductName    string
	Unit           string
	Quantity       int
	UnitPrice      float64
	DiscountAmount float64
	Subtotal       float64
}

func GetSalesOrderCustomer(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*SalesOrderCustomer, error) {
	var customer SalesOrderCustomer

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			customer_code,
			name,
			is_active
		FROM customers
		WHERE id = $1
		`,
		id,
	).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.Name,
		&customer.IsActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func GetSalesOrderWarehouse(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*SalesOrderWarehouse, error) {
	var warehouse SalesOrderWarehouse

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			warehouse_code,
			name,
			classification,
			is_active
		FROM warehouses
		WHERE id = $1
		`,
		id,
	).Scan(
		&warehouse.ID,
		&warehouse.WarehouseCode,
		&warehouse.Name,
		&warehouse.Classification,
		&warehouse.IsActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &warehouse, nil
}

func GetSalesOrderProduct(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (*SalesOrderProduct, error) {
	var product SalesOrderProduct

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id,
			product_code,
			sku,
			COALESCE(barcode, ''),
			name,
			unit,
			price,
			stock,
			is_active
		FROM products
		WHERE id = $1
		`,
		id,
	).Scan(
		&product.ID,
		&product.ProductCode,
		&product.SKU,
		&product.Barcode,
		&product.Name,
		&product.Unit,
		&product.Price,
		&product.Stock,
		&product.IsActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func InsertSalesOrderDraft(
	ctx context.Context,
	tx pgx.Tx,
	input InsertSalesOrderInput,
	items []InsertSalesOrderItemInput,
) (uuid.UUID, error) {
	var orderID uuid.UUID

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO sales_orders (
			customer_id,
			warehouse_id,
			warehouse_code_snapshot,
			warehouse_name_snapshot,
			warehouse_classification_snapshot,
			order_date,
			status,
			notes,
			payment_terms_days,
			payment_settlement,
			payment_channel,
			payment_provider,
			due_date,
			tax_mode,
			tax_profile_id,
			tax_profile_name_snapshot,
			tax_effective_year,
			effective_tax_rate,
			tax_base_factor,
			subtotal,
			discount_amount,
			tax_scheme,
			tax_rate,
			tax_base_amount,
			tax_amount,
			total,
			cl_qty,
			dss_amount,
			cluster,
			priority,
			recommendation
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			'DRAFT',
			NULLIF($7, ''),
			$8,
			$9,
			$10,
			NULLIF($11, ''),
			$12,
			$13,
			$14,
			NULLIF($15, ''),
			$16,
			$17,
			$18,
			$19,
			$20,
			$21,
			$22,
			$23,
			$24,
			$25,
			$26,
			$27,
			$28,
			$29,
			$30
		)
		RETURNING id
		`,
		input.CustomerID,
		input.WarehouseID,
		input.WarehouseCode,
		input.WarehouseName,
		input.WarehouseClass,
		input.OrderDate,
		input.Notes,
		input.PaymentTermsDays,
		input.PaymentSettlement,
		input.PaymentChannel,
		input.PaymentProvider,
		input.DueDate,
		input.TaxMode,
		input.TaxProfileID,
		input.TaxProfileName,
		input.TaxEffectiveYear,
		input.EffectiveTaxRate,
		input.TaxBaseFactor,
		input.Subtotal,
		input.DiscountAmount,
		input.TaxScheme,
		input.TaxRate,
		input.TaxBaseAmount,
		input.TaxAmount,
		input.Total,
		input.ClQty,
		input.DSSAmount,
		input.Cluster,
		input.Priority,
		input.Recommendation,
	).Scan(&orderID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, item := range items {
		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO sales_order_items (
				sales_order_id,
				product_id,
				product_code,
				sku,
				barcode,
				product_name,
				unit,
				quantity,
				unit_price,
				discount_amount,
				subtotal
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				NULLIF($5, ''),
				$6,
				$7,
				$8,
				$9,
				$10,
				$11
			)
			`,
			orderID,
			item.ProductID,
			item.ProductCode,
			item.SKU,
			item.Barcode,
			item.ProductName,
			item.Unit,
			item.Quantity,
			item.UnitPrice,
			item.DiscountAmount,
			item.Subtotal,
		)
		if err != nil {
			return uuid.Nil, err
		}
	}

	return orderID, nil
}

func GetSalesOrders(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]SalesOrder, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			so.id,
			so.order_number,
			so.customer_id,
			c.customer_code,
			c.name,
			so.warehouse_id,
			COALESCE(so.warehouse_code_snapshot, ''),
			COALESCE(so.warehouse_name_snapshot, ''),
			COALESCE(so.warehouse_classification_snapshot, ''),
			so.order_date,
			so.status,
			COALESCE(so.notes, ''),
			so.payment_terms_days,
			COALESCE(so.payment_settlement, 'IMMEDIATE'),
			COALESCE(so.payment_channel, 'UNSPECIFIED'),
			COALESCE(so.payment_provider, ''),
			so.due_date,
			COALESCE(so.tax_mode, 'NON_PPN'),
			so.tax_profile_id,
			COALESCE(so.tax_profile_name_snapshot, ''),
			COALESCE(so.tax_effective_year, 0),
			COALESCE(so.effective_tax_rate, 0),
			COALESCE(so.tax_base_factor, 0),
			so.subtotal,
			so.discount_amount,
			so.tax_scheme,
			so.tax_rate,
			so.tax_base_amount,
			so.tax_amount,
			so.total,
			so.cl_qty,
			so.dss_amount,
			COALESCE(so.cluster, ''),
			COALESCE(so.priority, ''),
			COALESCE(so.recommendation, ''),
			so.confirmed_at,
			so.picking_at,
			so.packed_at,
			so.shipped_at,
			so.completed_at,
			so.cancelled_at,
			so.created_at,
			so.updated_at
		FROM sales_orders so
		JOIN customers c
			ON c.id = so.customer_id
		ORDER BY so.created_at DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]SalesOrder, 0)

	for rows.Next() {
		var order SalesOrder

		if err := scanSalesOrder(rows, &order); err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func GetSalesOrderByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*SalesOrder, error) {
	var order SalesOrder

	row := pool.QueryRow(
		ctx,
		`
		SELECT
			so.id,
			so.order_number,
			so.customer_id,
			c.customer_code,
			c.name,
			so.warehouse_id,
			COALESCE(so.warehouse_code_snapshot, ''),
			COALESCE(so.warehouse_name_snapshot, ''),
			COALESCE(so.warehouse_classification_snapshot, ''),
			so.order_date,
			so.status,
			COALESCE(so.notes, ''),
			so.payment_terms_days,
			COALESCE(so.payment_settlement, 'IMMEDIATE'),
			COALESCE(so.payment_channel, 'UNSPECIFIED'),
			COALESCE(so.payment_provider, ''),
			so.due_date,
			COALESCE(so.tax_mode, 'NON_PPN'),
			so.tax_profile_id,
			COALESCE(so.tax_profile_name_snapshot, ''),
			COALESCE(so.tax_effective_year, 0),
			COALESCE(so.effective_tax_rate, 0),
			COALESCE(so.tax_base_factor, 0),
			so.subtotal,
			so.discount_amount,
			so.tax_scheme,
			so.tax_rate,
			so.tax_base_amount,
			so.tax_amount,
			so.total,
			so.cl_qty,
			so.dss_amount,
			COALESCE(so.cluster, ''),
			COALESCE(so.priority, ''),
			COALESCE(so.recommendation, ''),
			so.confirmed_at,
			so.picking_at,
			so.packed_at,
			so.shipped_at,
			so.completed_at,
			so.cancelled_at,
			so.created_at,
			so.updated_at
		FROM sales_orders so
		JOIN customers c
			ON c.id = so.customer_id
		WHERE so.id = $1
		`,
		id,
	)

	if err := scanSalesOrder(row, &order); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	items, err := getSalesOrderItems(
		ctx,
		pool,
		id,
	)
	if err != nil {
		return nil, err
	}

	order.Items = items

	return &order, nil
}

type salesOrderShipmentItem struct {
	ProductID   uuid.UUID
	ProductName string
	Quantity    int
	Stock       int
}

func ShipSalesOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (bool, error) {
	tx, err := pool.BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var (
		status      string
		orderNumber string
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			status,
			order_number
		FROM sales_orders
		WHERE id = $1
		FOR UPDATE
		`,
		id,
	).Scan(
		&status,
		&orderNumber,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if status != "PACKED" {
		return false, fmt.Errorf(
			"%w: status saat ini %s",
			ErrSalesOrderNotPacked,
			status,
		)
	}

	rows, err := tx.Query(
		ctx,
		`
		SELECT
			soi.product_id,
			soi.product_name,
			soi.quantity,
			p.stock
		FROM sales_order_items soi
		JOIN products p
			ON p.id = soi.product_id
		WHERE soi.sales_order_id = $1
		ORDER BY soi.created_at ASC, soi.id ASC
		FOR UPDATE OF p
		`,
		id,
	)
	if err != nil {
		return false, err
	}

	items := make(
		[]salesOrderShipmentItem,
		0,
	)

	for rows.Next() {
		var item salesOrderShipmentItem

		if err := rows.Scan(
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.Stock,
		); err != nil {
			rows.Close()
			return false, err
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}

	rows.Close()

	if len(items) == 0 {
		return false, errors.New(
			"sales order tidak memiliki item",
		)
	}

	for _, item := range items {
		if item.Stock < item.Quantity {
			return false, fmt.Errorf(
				"%w: %s tersedia %d, diperlukan %d",
				ErrSalesOrderInsufficientStock,
				item.ProductName,
				item.Stock,
				item.Quantity,
			)
		}
	}

	for _, item := range items {
		result, err := tx.Exec(
			ctx,
			`
			UPDATE products
			SET
				stock = stock - $2,
				updated_at = NOW()
			WHERE id = $1
			  AND stock >= $2
			`,
			item.ProductID,
			item.Quantity,
		)
		if err != nil {
			return false, err
		}

		if result.RowsAffected() != 1 {
			return false, fmt.Errorf(
				"%w: %s",
				ErrSalesOrderInsufficientStock,
				item.ProductName,
			)
		}

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO stock_movements (
				product_id,
				movement_type,
				qty,
				reference_type,
				reference_id,
				note
			)
			VALUES (
				$1,
				'OUT',
				$2,
				'SALES_ORDER',
				$3,
				$4
			)
			`,
			item.ProductID,
			item.Quantity,
			id,
			"Pengiriman "+orderNumber,
		)
		if err != nil {
			return false, err
		}
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE sales_orders
		SET
			status = 'SHIPPED',
			shipped_at = COALESCE(shipped_at, NOW()),
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'PACKED'
		`,
		id,
	)
	if err != nil {
		return false, err
	}

	if result.RowsAffected() != 1 {
		return false, ErrSalesOrderNotPacked
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

func TransitionSalesOrderStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	nextStatus string,
) (bool, error) {
	var query string

	switch nextStatus {
	case "CONFIRMED":
		query = `
			UPDATE sales_orders
			SET
				status = 'CONFIRMED',
				confirmed_at = COALESCE(confirmed_at, NOW()),
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'DRAFT'
		`

	case "PICKING":
		query = `
			UPDATE sales_orders
			SET
				status = 'PICKING',
				picking_at = COALESCE(picking_at, NOW()),
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'CONFIRMED'
		`

	case "PACKED":
		query = `
			UPDATE sales_orders
			SET
				status = 'PACKED',
				packed_at = COALESCE(packed_at, NOW()),
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'PICKING'
		`

	default:
		return false, errors.New("status tujuan sales order tidak didukung")
	}

	result, err := pool.Exec(
		ctx,
		query,
		id,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

func getSalesOrderItems(
	ctx context.Context,
	pool *pgxpool.Pool,
	orderID uuid.UUID,
) ([]SalesOrderItem, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			sales_order_id,
			product_id,
			product_code,
			sku,
			COALESCE(barcode, ''),
			product_name,
			unit,
			quantity,
			unit_price,
			discount_amount,
			subtotal,
			created_at
		FROM sales_order_items
		WHERE sales_order_id = $1
		ORDER BY created_at ASC, id ASC
		`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SalesOrderItem, 0)

	for rows.Next() {
		var item SalesOrderItem

		if err := rows.Scan(
			&item.ID,
			&item.SalesOrderID,
			&item.ProductID,
			&item.ProductCode,
			&item.SKU,
			&item.Barcode,
			&item.ProductName,
			&item.Unit,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.Subtotal,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSalesOrder(
	row rowScanner,
	order *SalesOrder,
) error {
	return row.Scan(
		&order.ID,
		&order.OrderNumber,
		&order.CustomerID,
		&order.CustomerCode,
		&order.CustomerName,
		&order.WarehouseID,
		&order.WarehouseCode,
		&order.WarehouseName,
		&order.WarehouseClass,
		&order.OrderDate,
		&order.Status,
		&order.Notes,
		&order.PaymentTermsDays,
		&order.PaymentSettlement,
		&order.PaymentChannel,
		&order.PaymentProvider,
		&order.DueDate,
		&order.TaxMode,
		&order.TaxProfileID,
		&order.TaxProfileName,
		&order.TaxEffectiveYear,
		&order.EffectiveTaxRate,
		&order.TaxBaseFactor,
		&order.Subtotal,
		&order.DiscountAmount,
		&order.TaxScheme,
		&order.TaxRate,
		&order.TaxBaseAmount,
		&order.TaxAmount,
		&order.Total,
		&order.ClQty,
		&order.DSSAmount,
		&order.Cluster,
		&order.Priority,
		&order.Recommendation,
		&order.ConfirmedAt,
		&order.PickingAt,
		&order.PackedAt,
		&order.ShippedAt,
		&order.CompletedAt,
		&order.CancelledAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
}
