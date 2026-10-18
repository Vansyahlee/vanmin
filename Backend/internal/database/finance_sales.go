package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinanceSalesFilter struct {
	DateFrom   time.Time
	DateTo     time.Time
	Query      string
	Status     string
	PrintState string
	Limit      int
	Offset     int
}

type FinanceSale struct {
	ID                 uuid.UUID  `json:"id"`
	InvoiceNumber      string     `json:"invoice_number"`
	DeliveryNoteNumber string     `json:"delivery_note_number"`
	InvoiceDate        time.Time  `json:"invoice_date"`
	SalesOrderID       uuid.UUID  `json:"sales_order_id"`
	OrderNumber        string     `json:"order_number"`
	CustomerID         uuid.UUID  `json:"customer_id"`
	CustomerCode       string     `json:"customer_code"`
	StoreName          string     `json:"store_name"`
	CustomerPhone      string     `json:"customer_phone"`
	CustomerTaxID      string     `json:"customer_tax_id_number"`
	CustomerTaxName    string     `json:"customer_tax_name"`
	CustomerTaxAddress string     `json:"customer_tax_address"`
	ShippingAddress    string     `json:"shipping_address"`
	WarehouseID        *uuid.UUID `json:"warehouse_id"`
	WarehouseCode      string     `json:"warehouse_code"`
	WarehouseName      string     `json:"warehouse_name"`
	PaymentSettlement  string     `json:"payment_settlement"`
	PaymentChannel     string     `json:"payment_channel"`
	PaymentProvider    string     `json:"payment_provider"`
	PaymentTermsDays   int        `json:"payment_terms_days"`
	DueDate            *time.Time `json:"due_date"`
	Subtotal           float64    `json:"subtotal"`
	DiscountAmount     float64    `json:"discount_amount"`
	NetSales           float64    `json:"net_sales"`
	TaxMode            string     `json:"tax_mode"`
	TaxScheme          string     `json:"tax_scheme"`
	TaxRate            float64    `json:"tax_rate"`
	EffectiveTaxRate   float64    `json:"effective_tax_rate"`
	TaxBaseAmount      float64    `json:"tax_base_amount"`
	TaxAmount          float64    `json:"tax_amount"`
	Total              float64    `json:"total"`
	TotalQty           int        `json:"total_qty"`
	Cluster            string     `json:"cluster"`
	Priority           string     `json:"priority"`
	Status             string     `json:"status"`
	IsPrinted          bool       `json:"is_printed"`
	PrintCount         int        `json:"print_count"`
	PrintVersion       int        `json:"print_version"`
	PrintedAt          *time.Time `json:"printed_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type FinanceSalesSummary struct {
	InvoiceCount   int64   `json:"invoice_count"`
	CustomerCount  int64   `json:"customer_count"`
	TotalQty       int64   `json:"total_qty"`
	Subtotal       float64 `json:"subtotal"`
	DiscountAmount float64 `json:"discount_amount"`
	NetSales       float64 `json:"net_sales"`
	TaxBaseAmount  float64 `json:"tax_base_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	GrandTotal     float64 `json:"grand_total"`
	AverageInvoice float64 `json:"average_invoice"`
	PrintedCount   int64   `json:"printed_count"`
	UnprintedCount int64   `json:"unprinted_count"`
	TermCount      int64   `json:"term_count"`
	ImmediateCount int64   `json:"immediate_count"`
}

type FinanceSalesTrendPoint struct {
	PeriodStart   time.Time `json:"period_start"`
	InvoiceCount  int64     `json:"invoice_count"`
	CustomerCount int64     `json:"customer_count"`
	TotalQty      int64     `json:"total_qty"`
	Subtotal      float64   `json:"subtotal"`
	Discount      float64   `json:"discount_amount"`
	NetSales      float64   `json:"net_sales"`
	TaxBaseAmount float64   `json:"tax_base_amount"`
	TaxAmount     float64   `json:"tax_amount"`
	GrandTotal    float64   `json:"grand_total"`
}

type FinanceSalesTrend struct {
	Granularity string                   `json:"granularity"`
	Points      []FinanceSalesTrendPoint `json:"points"`
}

func ListFinanceSales(
	ctx context.Context,
	pool *pgxpool.Pool,
	filter FinanceSalesFilter,
) ([]FinanceSale, int64, error) {
	queryText := strings.TrimSpace(filter.Query)
	status := strings.ToUpper(strings.TrimSpace(filter.Status))
	printState := strings.ToLower(strings.TrimSpace(filter.PrintState))

	rows, err := pool.Query(
		ctx,
		`
		SELECT
			i.id,
			i.invoice_number,
			COALESCE(i.delivery_note_number, ''),
			i.invoice_date,
			i.sales_order_id,
			COALESCE(i.order_number, ''),
			i.customer_id,
			COALESCE(i.customer_code, ''),
			i.store_name,
			COALESCE(i.customer_phone, ''),
			COALESCE(i.customer_tax_id_number, ''),
			COALESCE(i.customer_tax_name, ''),
			COALESCE(i.customer_tax_address, ''),
			COALESCE(i.shipping_address, ''),
			i.warehouse_id,
			COALESCE(i.warehouse_code, ''),
			COALESCE(i.warehouse_name, ''),
			COALESCE(i.payment_settlement, 'IMMEDIATE'),
			COALESCE(i.payment_channel, 'UNSPECIFIED'),
			COALESCE(i.payment_provider, ''),
			COALESCE(i.payment_terms_days, 0),
			i.due_date,
			COALESCE(i.subtotal, 0),
			COALESCE(i.discount_amount, 0),
			GREATEST(
				COALESCE(i.subtotal, 0) -
				COALESCE(i.discount_amount, 0),
				0
			),
			COALESCE(i.tax_mode, 'NON_PPN'),
			COALESCE(i.tax_scheme, 'NON_PPN'),
			COALESCE(i.ppn_percent, 0),
			COALESCE(i.effective_tax_rate, 0),
			COALESCE(i.tax_base_amount, 0),
			COALESCE(i.ppn_amount, 0),
			COALESCE(i.total, 0),
			COALESCE(i.cl_qty, 0),
			COALESCE(i.cluster, ''),
			COALESCE(i.priority, ''),
			COALESCE(i.status, ''),
			COALESCE(i.is_printed, FALSE),
			COALESCE(i.print_count, 0),
			COALESCE(i.print_version, 1),
			i.printed_at,
			i.created_at,
			i.updated_at
		FROM invoices i
		WHERE i.sales_order_id IS NOT NULL
		  AND i.customer_id IS NOT NULL
		  AND i.invoice_date >= $1::date
		  AND i.invoice_date <= $2::date
		  AND (
			$3 = '' OR
			LOWER(
				CONCAT_WS(
					' ',
					i.invoice_number,
					i.delivery_note_number,
					i.order_number,
					i.customer_code,
					i.store_name,
					i.customer_phone,
					i.customer_tax_id_number,
					i.customer_tax_name,
					i.shipping_address,
					i.warehouse_code,
					i.warehouse_name
				)
			) LIKE '%' || LOWER($3) || '%'
		  )
		  AND (
			$4 = '' OR
			UPPER(COALESCE(i.status, '')) = $4
		  )
		  AND (
			$5 = '' OR
			$5 = 'all' OR
			($5 = 'printed' AND i.is_printed = TRUE) OR
			($5 = 'unprinted' AND i.is_printed = FALSE)
		  )
		ORDER BY
			i.invoice_date DESC,
			i.created_at DESC,
			i.invoice_number DESC
		LIMIT $6
		OFFSET $7
		`,
		filter.DateFrom,
		filter.DateTo,
		queryText,
		status,
		printState,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	sales := make([]FinanceSale, 0)

	for rows.Next() {
		var sale FinanceSale

		if err := rows.Scan(
			&sale.ID,
			&sale.InvoiceNumber,
			&sale.DeliveryNoteNumber,
			&sale.InvoiceDate,
			&sale.SalesOrderID,
			&sale.OrderNumber,
			&sale.CustomerID,
			&sale.CustomerCode,
			&sale.StoreName,
			&sale.CustomerPhone,
			&sale.CustomerTaxID,
			&sale.CustomerTaxName,
			&sale.CustomerTaxAddress,
			&sale.ShippingAddress,
			&sale.WarehouseID,
			&sale.WarehouseCode,
			&sale.WarehouseName,
			&sale.PaymentSettlement,
			&sale.PaymentChannel,
			&sale.PaymentProvider,
			&sale.PaymentTermsDays,
			&sale.DueDate,
			&sale.Subtotal,
			&sale.DiscountAmount,
			&sale.NetSales,
			&sale.TaxMode,
			&sale.TaxScheme,
			&sale.TaxRate,
			&sale.EffectiveTaxRate,
			&sale.TaxBaseAmount,
			&sale.TaxAmount,
			&sale.Total,
			&sale.TotalQty,
			&sale.Cluster,
			&sale.Priority,
			&sale.Status,
			&sale.IsPrinted,
			&sale.PrintCount,
			&sale.PrintVersion,
			&sale.PrintedAt,
			&sale.CreatedAt,
			&sale.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		sales = append(sales, sale)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int64

	err = pool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM invoices i
		WHERE i.sales_order_id IS NOT NULL
		  AND i.customer_id IS NOT NULL
		  AND i.invoice_date >= $1::date
		  AND i.invoice_date <= $2::date
		  AND (
			$3 = '' OR
			LOWER(
				CONCAT_WS(
					' ',
					i.invoice_number,
					i.delivery_note_number,
					i.order_number,
					i.customer_code,
					i.store_name,
					i.customer_phone,
					i.customer_tax_id_number,
					i.customer_tax_name,
					i.shipping_address,
					i.warehouse_code,
					i.warehouse_name
				)
			) LIKE '%' || LOWER($3) || '%'
		  )
		  AND (
			$4 = '' OR
			UPPER(COALESCE(i.status, '')) = $4
		  )
		  AND (
			$5 = '' OR
			$5 = 'all' OR
			($5 = 'printed' AND i.is_printed = TRUE) OR
			($5 = 'unprinted' AND i.is_printed = FALSE)
		  )
		`,
		filter.DateFrom,
		filter.DateTo,
		queryText,
		status,
		printState,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return sales, total, nil
}

func GetFinanceSalesSummary(
	ctx context.Context,
	pool *pgxpool.Pool,
	filter FinanceSalesFilter,
) (*FinanceSalesSummary, error) {
	queryText := strings.TrimSpace(filter.Query)
	status := strings.ToUpper(strings.TrimSpace(filter.Status))
	printState := strings.ToLower(strings.TrimSpace(filter.PrintState))

	var summary FinanceSalesSummary

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			COUNT(*),
			COUNT(DISTINCT i.customer_id),
			COALESCE(SUM(i.cl_qty), 0),
			COALESCE(SUM(i.subtotal), 0),
			COALESCE(SUM(i.discount_amount), 0),
			COALESCE(
				SUM(
					GREATEST(
						COALESCE(i.subtotal, 0) -
						COALESCE(i.discount_amount, 0),
						0
					)
				),
				0
			),
			COALESCE(SUM(i.tax_base_amount), 0),
			COALESCE(SUM(i.ppn_amount), 0),
			COALESCE(SUM(i.total), 0),
			COALESCE(AVG(i.total), 0),
			COUNT(*) FILTER (
				WHERE i.is_printed = TRUE
			),
			COUNT(*) FILTER (
				WHERE i.is_printed = FALSE
			),
			COUNT(*) FILTER (
				WHERE COALESCE(i.payment_settlement, 'IMMEDIATE') = 'TERM'
			),
			COUNT(*) FILTER (
				WHERE COALESCE(i.payment_settlement, 'IMMEDIATE') = 'IMMEDIATE'
			)
		FROM invoices i
		WHERE i.sales_order_id IS NOT NULL
		  AND i.customer_id IS NOT NULL
		  AND i.invoice_date >= $1::date
		  AND i.invoice_date <= $2::date
		  AND (
			$3 = '' OR
			LOWER(
				CONCAT_WS(
					' ',
					i.invoice_number,
					i.delivery_note_number,
					i.order_number,
					i.customer_code,
					i.store_name,
					i.customer_phone,
					i.customer_tax_id_number,
					i.customer_tax_name,
					i.shipping_address,
					i.warehouse_code,
					i.warehouse_name
				)
			) LIKE '%' || LOWER($3) || '%'
		  )
		  AND (
			$4 = '' OR
			UPPER(COALESCE(i.status, '')) = $4
		  )
		  AND (
			$5 = '' OR
			$5 = 'all' OR
			($5 = 'printed' AND i.is_printed = TRUE) OR
			($5 = 'unprinted' AND i.is_printed = FALSE)
		  )
		`,
		filter.DateFrom,
		filter.DateTo,
		queryText,
		status,
		printState,
	).Scan(
		&summary.InvoiceCount,
		&summary.CustomerCount,
		&summary.TotalQty,
		&summary.Subtotal,
		&summary.DiscountAmount,
		&summary.NetSales,
		&summary.TaxBaseAmount,
		&summary.TaxAmount,
		&summary.GrandTotal,
		&summary.AverageInvoice,
		&summary.PrintedCount,
		&summary.UnprintedCount,
		&summary.TermCount,
		&summary.ImmediateCount,
	)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func GetFinanceSalesTrend(
	ctx context.Context,
	pool *pgxpool.Pool,
	filter FinanceSalesFilter,
) (*FinanceSalesTrend, error) {
	days :=
		int(
			filter.DateTo.Sub(
				filter.DateFrom,
			).Hours()/24,
		) + 1

	granularity := "day"
	bucketSQL := `
		SELECT
			gs::date AS period_start,
			(gs + INTERVAL '1 day')::date AS period_end
		FROM generate_series(
			$1::date,
			$2::date,
			INTERVAL '1 day'
		) gs
	`

	if days > 45 && days <= 210 {
		granularity = "week"
		bucketSQL = `
			SELECT
				gs::date AS period_start,
				LEAST(
					(gs + INTERVAL '7 days')::date,
					($2::date + INTERVAL '1 day')::date
				) AS period_end
			FROM generate_series(
				$1::date,
				$2::date,
				INTERVAL '7 days'
			) gs
		`
	}

	if days > 210 {
		granularity = "month"
		bucketSQL = `
			SELECT
				gs::date AS period_start,
				LEAST(
					(gs + INTERVAL '1 month')::date,
					($2::date + INTERVAL '1 day')::date
				) AS period_end
			FROM generate_series(
				date_trunc(
					'month',
					$1::date
				),
				date_trunc(
					'month',
					$2::date
				),
				INTERVAL '1 month'
			) gs
		`
	}

	queryText := strings.TrimSpace(filter.Query)
	status := strings.ToUpper(strings.TrimSpace(filter.Status))
	printState := strings.ToLower(strings.TrimSpace(filter.PrintState))

	query := fmt.Sprintf(
		`
		WITH buckets AS (
			%s
		),
		filtered AS (
			SELECT
				i.id,
				i.invoice_date,
				i.customer_id,
				COALESCE(i.cl_qty, 0) AS cl_qty,
				COALESCE(i.subtotal, 0) AS subtotal,
				COALESCE(i.discount_amount, 0) AS discount_amount,
				GREATEST(
					COALESCE(i.subtotal, 0) -
					COALESCE(i.discount_amount, 0),
					0
				) AS net_sales,
				COALESCE(i.tax_base_amount, 0) AS tax_base_amount,
				COALESCE(i.ppn_amount, 0) AS tax_amount,
				COALESCE(i.total, 0) AS grand_total
			FROM invoices i
			WHERE i.sales_order_id IS NOT NULL
			  AND i.customer_id IS NOT NULL
			  AND i.invoice_date >= $1::date
			  AND i.invoice_date <= $2::date
			  AND (
				$3 = '' OR
				LOWER(
					CONCAT_WS(
						' ',
						i.invoice_number,
						i.delivery_note_number,
						i.order_number,
						i.customer_code,
						i.store_name,
						i.customer_phone,
						i.customer_tax_id_number,
						i.customer_tax_name,
						i.shipping_address,
						i.warehouse_code,
						i.warehouse_name
					)
				) LIKE '%%' || LOWER($3) || '%%'
			  )
			  AND (
				$4 = '' OR
				UPPER(COALESCE(i.status, '')) = $4
			  )
			  AND (
				$5 = '' OR
				$5 = 'all' OR
				($5 = 'printed' AND i.is_printed = TRUE) OR
				($5 = 'unprinted' AND i.is_printed = FALSE)
			  )
		)
		SELECT
			b.period_start,
			COUNT(f.id),
			COUNT(
				DISTINCT f.customer_id
			),
			COALESCE(
				SUM(f.cl_qty),
				0
			),
			COALESCE(
				SUM(f.subtotal),
				0
			),
			COALESCE(
				SUM(f.discount_amount),
				0
			),
			COALESCE(
				SUM(f.net_sales),
				0
			),
			COALESCE(
				SUM(f.tax_base_amount),
				0
			),
			COALESCE(
				SUM(f.tax_amount),
				0
			),
			COALESCE(
				SUM(f.grand_total),
				0
			)
		FROM buckets b
		LEFT JOIN filtered f
			ON f.invoice_date >= b.period_start
			AND f.invoice_date < b.period_end
		GROUP BY
			b.period_start
		ORDER BY
			b.period_start ASC
		`,
		bucketSQL,
	)

	rows, err := pool.Query(
		ctx,
		query,
		filter.DateFrom,
		filter.DateTo,
		queryText,
		status,
		printState,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points :=
		make(
			[]FinanceSalesTrendPoint,
			0,
		)

	for rows.Next() {
		var point FinanceSalesTrendPoint

		if err := rows.Scan(
			&point.PeriodStart,
			&point.InvoiceCount,
			&point.CustomerCount,
			&point.TotalQty,
			&point.Subtotal,
			&point.Discount,
			&point.NetSales,
			&point.TaxBaseAmount,
			&point.TaxAmount,
			&point.GrandTotal,
		); err != nil {
			return nil, err
		}

		points =
			append(
				points,
				point,
			)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &FinanceSalesTrend{
		Granularity: granularity,
		Points:      points,
	}, nil
}
