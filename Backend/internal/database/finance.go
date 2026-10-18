package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReceivableFilter struct {
	Query    string
	DateFrom string
	DateTo   string
	Status   string
	Aging    string
	Limit    int
	Offset   int
}

func ListFinanceReceivables(ctx context.Context, pool *pgxpool.Pool, filter ReceivableFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `
        WITH balances AS (
            SELECT i.id, i.invoice_number, COALESCE(i.order_number, '') AS order_number,
                i.customer_id, COALESCE(i.customer_code, '') AS customer_code,
                i.store_name, COALESCE(i.customer_phone, '') AS customer_phone,
                COALESCE(i.customer_tax_id_number, '') AS customer_tax_id_number,
                COALESCE(i.shipping_address, '') AS shipping_address,
                COALESCE(i.warehouse_name, '') AS warehouse_name,
                i.invoice_date::date AS invoice_date, i.due_date::date AS due_date,
                COALESCE(i.payment_terms_days, 0) AS payment_terms_days,
                COALESCE(i.total, 0)::numeric AS total,
                COALESCE(p.paid_amount, 0)::numeric AS paid_amount,
                GREATEST(COALESCE(i.total, 0)::numeric - COALESCE(p.paid_amount, 0), 0) AS outstanding,
                (NOW() AT TIME ZONE 'Asia/Jakarta')::date AS as_of,
                CASE WHEN i.due_date IS NULL THEN NULL
                    ELSE (NOW() AT TIME ZONE 'Asia/Jakarta')::date - i.due_date::date END AS overdue_days
            FROM invoices i
            LEFT JOIN LATERAL (
                SELECT SUM(amount-refunded_amount) AS paid_amount
                FROM invoice_payments
                WHERE invoice_id = i.id AND status = 'CONFIRMED'
            ) p ON TRUE
            WHERE i.sales_order_id IS NOT NULL
                AND (UPPER(COALESCE(i.payment_settlement, '')) = 'TERM' OR COALESCE(i.payment_terms_days, 0) > 0)
                AND UPPER(COALESCE(i.status, '')) NOT IN ('VOID', 'CANCELLED', 'CANCELED')
                AND ($2 = '' OR i.invoice_date::date >= NULLIF($2, '')::date)
                AND ($3 = '' OR i.invoice_date::date <= NULLIF($3, '')::date)
                AND ($1 = '' OR STRPOS(LOWER(CONCAT_WS(' ', i.invoice_number, i.order_number,
                    i.store_name, i.customer_code, i.customer_phone, i.customer_tax_id_number,
                    i.shipping_address, i.warehouse_name)), LOWER($1)) > 0)
        ), classified AS (
            SELECT *,
                CASE WHEN outstanding = 0 THEN 'PAID' WHEN paid_amount > 0 THEN 'PARTIAL' ELSE 'UNPAID' END AS payment_status,
                CASE WHEN outstanding = 0 THEN 'PAID' WHEN due_date IS NULL THEN 'UNSCHEDULED'
                    WHEN overdue_days > 0 THEN 'OVERDUE' WHEN overdue_days = 0 THEN 'DUE'
                    ELSE 'NOT_DUE' END AS due_status,
                CASE WHEN outstanding = 0 THEN 'PAID' WHEN due_date IS NULL THEN 'UNSCHEDULED'
                    WHEN overdue_days < 0 THEN 'NOT_DUE' WHEN overdue_days = 0 THEN 'DUE'
                    WHEN overdue_days <= 7 THEN '1_7' WHEN overdue_days <= 14 THEN '8_14'
                    WHEN overdue_days <= 30 THEN '15_30' ELSE 'OVER_30' END AS aging
            FROM balances
        ), filtered AS (
            SELECT * FROM classified
            WHERE ($4 = 'ALL' OR $4 = '' OR ($4 = 'OPEN' AND outstanding > 0) OR payment_status = $4 OR due_status = $4)
                AND ($5 = 'ALL' OR $5 = '' OR aging = $5)
        ), page AS (
            SELECT * FROM filtered ORDER BY due_date ASC NULLS LAST, invoice_number, id LIMIT $6 OFFSET $7
        )
        SELECT JSONB_BUILD_OBJECT(
            'data', COALESCE((SELECT JSONB_AGG(TO_JSONB(page) ORDER BY due_date ASC NULLS LAST, invoice_number, id) FROM page), '[]'::jsonb),
            'meta', JSONB_BUILD_OBJECT('total', (SELECT COUNT(*) FROM filtered), 'limit', $6::int, 'offset', $7::int,
                'as_of', (NOW() AT TIME ZONE 'Asia/Jakarta')::date),
            'summary', (SELECT JSONB_BUILD_OBJECT(
                'invoice_count', COUNT(*), 'customer_count', COUNT(DISTINCT customer_id),
                'total', COALESCE(SUM(total), 0)::text, 'paid_amount', COALESCE(SUM(paid_amount), 0)::text,
                'outstanding', COALESCE(SUM(outstanding), 0)::text,
                'overdue', COALESCE(SUM(outstanding) FILTER (WHERE due_status = 'OVERDUE'), 0)::text,
                'due_today', COALESCE(SUM(outstanding) FILTER (WHERE due_status = 'DUE'), 0)::text,
                'over_30', COALESCE(SUM(outstanding) FILTER (WHERE aging = 'OVER_30'), 0)::text
            ) FROM filtered),
            'aging', COALESCE((SELECT JSONB_AGG(TO_JSONB(groups)) FROM (
                SELECT aging AS key, COUNT(*) AS count, SUM(outstanding)::text AS amount
                FROM filtered WHERE outstanding > 0 GROUP BY aging
            ) groups), '[]'::jsonb)
        )
    `, filter.Query, filter.DateFrom, filter.DateTo, filter.Status, filter.Aging, filter.Limit, filter.Offset).Scan(&payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

// Product frequency uses distinct invoices so different packaging units are not mixed.
type ProductTrendFilter struct {
	DateFrom string
	DateTo   string
	Query    string
	Category string
	Sort     string
	Limit    int
	Offset   int
}

func ListFinanceProductTrend(ctx context.Context, pool *pgxpool.Pool, filter ProductTrendFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `
        WITH sold AS (
            SELECT ii.product_id, COUNT(DISTINCT i.id) AS invoice_count,
                COALESCE(SUM(ii.qty),0) AS qty,
                COALESCE(SUM(ii.subtotal),0) AS net_sales,
                MAX((i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date) AS last_sold
            FROM invoice_items ii JOIN invoices i ON i.id=ii.invoice_id
            WHERE UPPER(COALESCE(i.status,'')) NOT IN ('VOID','CANCELLED','CANCELED')
                AND ii.qty > 0
                AND ($1='' OR (i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($1,'')::date)
                AND ($2='' OR (i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($2,'')::date)
            GROUP BY ii.product_id
        ), catalog AS (
            SELECT p.id,p.product_code,p.sku,p.name,p.unit,p.is_active,
                COALESCE(p.stock,0) AS stock,COALESCE(p.min_stock,0) AS min_stock,
                COALESCE(p.purchase_price,0)::text AS purchase_price,
                COALESCE(s.invoice_count,0) AS invoice_count,COALESCE(s.qty,0) AS qty,
                COALESCE(s.net_sales,0)::text AS net_sales,s.last_sold
            FROM products p LEFT JOIN sold s ON s.product_id=p.id
            WHERE s.invoice_count>0 OR (p.is_active AND ($2='' OR (p.created_at AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($2,'')::date))
        ), classified AS (
            SELECT *, CASE WHEN invoice_count=0 THEN 'UNSOLD'
                WHEN invoice_count>=CEIL(MAX(invoice_count) OVER ()*0.5) THEN 'FAST'
                ELSE 'MEDIUM' END AS category
            FROM catalog
        ), ranked AS (
            SELECT *,ROW_NUMBER() OVER(PARTITION BY category ORDER BY
                CASE WHEN $5='quantity' THEN qty END DESC NULLS LAST,
                CASE WHEN $5='revenue' THEN net_sales::numeric END DESC NULLS LAST,
                CASE WHEN $5='frequency' THEN invoice_count END DESC NULLS LAST,
                CASE WHEN $5='name' THEN LOWER(name) END ASC NULLS LAST,
                invoice_count DESC,qty DESC,LOWER(name),id) AS category_rank
            FROM classified
        ), searched AS (
            SELECT * FROM ranked WHERE ($3='' OR STRPOS(LOWER(CONCAT_WS(' ',name,sku,product_code)),LOWER($3))>0)
        ), filtered AS (
            SELECT *,ROW_NUMBER() OVER(ORDER BY
                CASE category WHEN 'FAST' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,
                category_rank,id) AS display_order
            FROM searched WHERE ($4='' OR $4='ALL' OR category=$4)
        ), page AS (
            SELECT * FROM filtered ORDER BY display_order LIMIT $6 OFFSET $7
        )
        SELECT JSONB_BUILD_OBJECT(
            'data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page) ORDER BY display_order) FROM page),'[]'::jsonb),
            'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered),'limit',$6::int,'offset',$7::int,
                'date_from',$1::text,'date_to',$2::text,'sort',$5::text),
            'summary',(SELECT JSONB_BUILD_OBJECT('FAST',COUNT(*) FILTER(WHERE category='FAST'),
                'MEDIUM',COUNT(*) FILTER(WHERE category='MEDIUM'),'UNSOLD',COUNT(*) FILTER(WHERE category='UNSOLD')) FROM searched)
        )
    `, filter.DateFrom, filter.DateTo, filter.Query, filter.Category, filter.Sort, filter.Limit, filter.Offset).Scan(&payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}
