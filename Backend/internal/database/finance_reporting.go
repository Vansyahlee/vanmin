package database

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrReportingRevision = errors.New("data berubah; muat ulang sebelum menyimpan")

// Sales are before VAT; expenses paid and inventory purchases are separate cash information.
func FinancePeriodReport(ctx context.Context, pool *pgxpool.Pool, from, to string) (json.RawMessage, error) {
	var result []byte
	err := pool.QueryRow(ctx, `WITH months AS (
 SELECT generate_series(date_trunc('month',$1::date),date_trunc('month',$2::date),'1 month')::date AS period
 ), sales AS (
 SELECT date_trunc('month',invoice_date AT TIME ZONE 'Asia/Jakarta')::date period,
 COUNT(*) invoice_count,SUM(ROUND((total-COALESCE(ppn_amount,0))::numeric,2)) revenue,SUM(ROUND(COALESCE(ppn_amount,0)::numeric,2)) vat,SUM(ROUND(total::numeric,2)) billed
 FROM invoices WHERE (invoice_date AT TIME ZONE 'Asia/Jakarta')::date BETWEEN $1::date AND $2::date
 AND sales_order_id IS NOT NULL AND UPPER(COALESCE(status,'')) NOT IN ('VOID','CANCELLED','CANCELED','DRAFT') GROUP BY 1
 ), paid AS (
 SELECT date_trunc('month',expense_date)::date period,
 SUM(amount) FILTER(WHERE expense_kind='SALARY') salary_paid,
 SUM(amount) FILTER(WHERE expense_kind='SUPPLIER') supplier_paid,
 SUM(amount) FILTER(WHERE expense_kind='GENERAL') general_paid
 FROM finance_expenses WHERE status='POSTED' AND expense_date BETWEEN $1::date AND $2::date GROUP BY 1
 ), cash AS (
 SELECT date_trunc('month',occurred_at AT TIME ZONE 'Asia/Jakarta')::date period,
 SUM(amount) FILTER(WHERE direction='IN') incoming,SUM(amount) FILTER(WHERE direction='OUT') outgoing
 FROM finance_cash_movements WHERE (occurred_at AT TIME ZONE 'Asia/Jakarta')::date BETWEEN $1::date AND $2::date GROUP BY 1
 ), rows AS (
 SELECT to_char(m.period,'YYYY-MM') AS period,($1::date<=m.period AND $2::date>=(m.period+INTERVAL '1 month - 1 day')::date) full_month,COALESCE(s.invoice_count,0) invoice_count,
 COALESCE(s.revenue,0)::text revenue,COALESCE(s.vat,0)::text vat,COALESCE(s.billed,0)::text billed,
 COALESCE(p.salary_paid,0)::text salary_paid,COALESCE(p.supplier_paid,0)::text supplier_paid,COALESCE(p.general_paid,0)::text general_paid,
 COALESCE(c.incoming,0)::text incoming,COALESCE(c.outgoing,0)::text outgoing,
 COALESCE(f.settings,'{}'::jsonb) settings,COALESCE(f.revision,0) revision,f.updated_at,
 COALESCE(u.name,f.updated_by,'') updated_by_name
 FROM months m LEFT JOIN sales s USING(period) LEFT JOIN paid p USING(period) LEFT JOIN cash c USING(period)
 LEFT JOIN finance_profit_periods f USING(period) LEFT JOIN users u ON u.id::text=f.updated_by
 ) SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(rows ORDER BY period) FROM rows),'[]'::jsonb),
 'date_from',$1::text,'date_to',$2::text,'generated_at',NOW(),
 'cash_opening',COALESCE((SELECT SUM(CASE WHEN direction='IN' THEN amount ELSE -amount END) FROM finance_cash_movements WHERE (occurred_at AT TIME ZONE 'Asia/Jakarta')::date<$1::date),0)::text)
 `, from, to).Scan(&result)
	return json.RawMessage(result), err
}

func SaveProfitPeriod(ctx context.Context, pool *pgxpool.Pool, period string, settings json.RawMessage, revision int64, actor string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO finance_profit_periods(period) VALUES($1::date) ON CONFLICT DO NOTHING`, period)
	if err != nil {
		return err
	}
	var before, after json.RawMessage
	var current int64
	err = tx.QueryRow(ctx, `SELECT TO_JSONB(p),revision FROM finance_profit_periods p WHERE period=$1::date FOR UPDATE`, period).Scan(&before, &current)
	if err != nil {
		return err
	}
	if current != revision {
		return ErrReportingRevision
	}
	err = tx.QueryRow(ctx, `UPDATE finance_profit_periods SET settings=$2::jsonb,revision=revision+1,updated_by=$3,updated_at=NOW() WHERE period=$1::date RETURNING TO_JSONB(finance_profit_periods)`, period, string(settings), actor).Scan(&after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance_reporting_audit(kind,reference,actor,before_data,after_data) VALUES('PROFIT',$1,$2,$3::jsonb,$4::jsonb)`, period, actor, string(before), string(after))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// A fingerprint covers tax-relevant invoice data, not payment/print timestamps.
type taxSourceReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func FinanceTaxSource(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (json.RawMessage, string, error) {
	return readFinanceTaxSource(ctx, pool, id)
}
func readFinanceTaxSource(ctx context.Context, pool taxSourceReader, id uuid.UUID) (json.RawMessage, string, error) {
	var raw []byte
	var fingerprint string
	err := pool.QueryRow(ctx, `WITH source AS (
 SELECT JSONB_BUILD_OBJECT('id',i.id,'invoice_number',i.invoice_number,'invoice_date',to_char(i.invoice_date AT TIME ZONE 'Asia/Jakarta','YYYY-MM-DD'),
 'issuer_profile',i.issuer_profile,'store_name',i.store_name,'buyer_tin',COALESCE(i.customer_tax_id_number,''),'buyer_name',COALESCE(NULLIF(i.customer_tax_name,''),i.store_name),
 'buyer_address',COALESCE(NULLIF(i.customer_tax_address,''),i.shipping_address,''),'tax_mode',i.tax_mode,'tax_scheme',i.tax_scheme,
 'total',ROUND(i.total::numeric,2)::text,'vat',ROUND(COALESCE(i.ppn_amount,0)::numeric,2)::text,'tax_base',ROUND(COALESCE(i.tax_base_amount,0)::numeric,2)::text,
 'tax_rate',ROUND(COALESCE(i.ppn_percent,0)::numeric,2)::text,
 'items',COALESCE((SELECT JSONB_AGG(JSONB_BUILD_OBJECT('id',ii.id,'name',ii.product_name,'qty',ii.qty::text,'price',ROUND(ii.price::numeric,2)::text,'discount',ROUND(COALESCE(ii.discount_amount,0)::numeric,2)::text,'subtotal',ROUND(ii.subtotal::numeric,2)::text,'unit',ii.unit) ORDER BY ii.id) FROM invoice_items ii WHERE ii.invoice_id=i.id),'[]'::jsonb)) AS doc
 FROM invoices i WHERE i.id=$1 AND i.sales_order_id IS NOT NULL AND UPPER(COALESCE(i.status,'')) NOT IN ('VOID','CANCELLED','CANCELED','DRAFT')
 ) SELECT doc,md5(doc::text) FROM source`, id).Scan(&raw, &fingerprint)
	return json.RawMessage(raw), fingerprint, err
}
func GetTaxDraft(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (json.RawMessage, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT JSONB_BUILD_OBJECT('document',document,'source_fingerprint',source_fingerprint,'revision',revision,'updated_at',updated_at) FROM finance_tax_drafts WHERE invoice_id=$1`, id).Scan(&raw)
	return json.RawMessage(raw), err
}
func SaveTaxDraft(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, fingerprint string, document json.RawMessage, revision int64, actor string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Locks block concurrent invoice updates while checking the exact source fingerprint.
	var locked uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM invoices WHERE id=$1 FOR UPDATE`, id).Scan(&locked)
	if err != nil {
		return err
	}
	// Re-read through the transaction while holding the invoice row lock.
	_, currentFingerprint, err := readFinanceTaxSource(ctx, tx, id)
	if err != nil {
		return err
	}
	if currentFingerprint != fingerprint {
		return ErrReportingRevision
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance_tax_drafts(invoice_id,source_fingerprint) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, fingerprint)
	if err != nil {
		return err
	}
	var before, after json.RawMessage
	var current int64
	err = tx.QueryRow(ctx, `SELECT TO_JSONB(d),revision FROM finance_tax_drafts d WHERE invoice_id=$1 FOR UPDATE`, id).Scan(&before, &current)
	if err != nil {
		return err
	}
	if current != revision {
		return ErrReportingRevision
	}
	err = tx.QueryRow(ctx, `UPDATE finance_tax_drafts SET document=$2::jsonb,source_fingerprint=$3,revision=revision+1,updated_by=$4,updated_at=NOW() WHERE invoice_id=$1 RETURNING TO_JSONB(finance_tax_drafts)`, id, string(document), fingerprint, actor).Scan(&after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance_reporting_audit(kind,reference,actor,before_data,after_data) VALUES('TAX',$1,$2,$3::jsonb,$4::jsonb)`, id.String(), actor, string(before), string(after))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func ReportingHistory(ctx context.Context, pool *pgxpool.Pool, kind, reference string) (json.RawMessage, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT COALESCE(JSONB_AGG(x ORDER BY changed_at DESC,id DESC),'[]'::jsonb) FROM (SELECT a.id::text,a.changed_at,COALESCE(u.name,a.actor) actor_name,a.before_data,a.after_data FROM finance_reporting_audit a LEFT JOIN users u ON u.id::text=a.actor WHERE a.kind=$1 AND a.reference=$2 ORDER BY a.changed_at DESC,a.id DESC LIMIT 50) x`, kind, reference).Scan(&raw)
	return json.RawMessage(raw), err
}

func FinanceTaxRegister(ctx context.Context, pool *pgxpool.Pool, f LedgerFilter) (json.RawMessage, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `WITH filtered AS (
 SELECT i.id,i.invoice_number,(i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date invoice_date,i.store_name,
 COALESCE(i.customer_tax_id_number,'') buyer_tin,COALESCE(i.tax_mode,'NON_PPN') tax_mode,
 COALESCE(i.tax_scheme,'') tax_scheme,ROUND((i.total-COALESCE(i.ppn_amount,0))::numeric,2)::text net_sales,
 ROUND(COALESCE(i.tax_base_amount,0)::numeric,2)::text tax_base,ROUND(COALESCE(i.ppn_amount,0)::numeric,2)::text vat,ROUND(i.total::numeric,2)::text total,
 COALESCE(d.revision,0) draft_revision
 FROM invoices i LEFT JOIN finance_tax_drafts d ON d.invoice_id=i.id
 WHERE i.sales_order_id IS NOT NULL AND UPPER(COALESCE(i.status,'')) NOT IN ('VOID','CANCELLED','CANCELED','DRAFT')
 AND ($1='' OR (i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($1,'')::date)
 AND ($2='' OR (i.invoice_date AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($2,'')::date)
 AND ($3='' OR STRPOS(LOWER(CONCAT_WS(' ',i.invoice_number,i.store_name,i.customer_tax_id_number)),LOWER($3))>0)
 AND ($4='' OR ($4='PPN' AND COALESCE(i.tax_mode,'NON_PPN')<>'NON_PPN') OR ($4='NON_PPN' AND COALESCE(i.tax_mode,'NON_PPN')='NON_PPN'))
 ),page AS(SELECT * FROM filtered ORDER BY invoice_date DESC,invoice_number,id LIMIT $5 OFFSET $6)
 SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(page ORDER BY invoice_date DESC,invoice_number,id) FROM page),'[]'::jsonb),
 'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered)),
 'summary',JSONB_BUILD_OBJECT('net_sales',COALESCE((SELECT SUM(net_sales::numeric) FROM filtered),0)::text,'vat',COALESCE((SELECT SUM(vat::numeric) FROM filtered),0)::text,'tax_base',COALESCE((SELECT SUM(tax_base::numeric) FROM filtered),0)::text))`, f.DateFrom, f.DateTo, f.Query, f.Category, f.Limit, f.Offset).Scan(&raw)
	return json.RawMessage(raw), err
}
