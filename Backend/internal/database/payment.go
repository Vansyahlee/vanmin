package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"furi-backend/internal/paymentlogic"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPaymentConflict = errors.New("pembayaran bertentangan dengan saldo atau permintaan sebelumnya")
var ErrPaymentInvoice = errors.New("faktur tempo aktif tidak ditemukan")

type PaymentInput struct {
	InvoiceID   uuid.UUID
	RequestKey  uuid.UUID
	Amount      string
	Source      string
	Method      string
	Reference   string
	Notes       string
	Actor       string
	PaidAt      time.Time
	Environment string
}

type PaymentIntent struct {
	ID            uuid.UUID `json:"id"`
	InvoiceID     uuid.UUID `json:"invoice_id"`
	RequestKey    uuid.UUID `json:"request_key"`
	Amount        string    `json:"amount"`
	Source        string    `json:"source"`
	State         string    `json:"state"`
	Method        string    `json:"method"`
	Reference     string    `json:"reference_number"`
	Notes         string    `json:"notes"`
	Actor         string    `json:"created_by"`
	PaidAt        time.Time `json:"paid_at"`
	OrderID       string    `json:"gateway_order_id"`
	GatewayStatus string    `json:"gateway_status"`
	Token         string    `json:"-"`
	RedirectURL   string    `json:"redirect_url"`
	Environment   string    `json:"environment"`
	ExpiresAt     time.Time `json:"expires_at"`
}

const intentColumns = `id, invoice_id, request_key, amount::text, source, state, method,
    reference_number, notes, created_by, paid_at, COALESCE(gateway_order_id,''),
    gateway_status, snap_token, redirect_url, environment, expires_at`

func scanPaymentIntent(row pgx.Row) (*PaymentIntent, error) {
	var item PaymentIntent
	err := row.Scan(&item.ID, &item.InvoiceID, &item.RequestKey, &item.Amount, &item.Source, &item.State,
		&item.Method, &item.Reference, &item.Notes, &item.Actor, &item.PaidAt, &item.OrderID,
		&item.GatewayStatus, &item.Token, &item.RedirectURL, &item.Environment, &item.ExpiresAt)
	return &item, err
}

func GetPaymentIntent(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*PaymentIntent, error) {
	return scanPaymentIntent(pool.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE id=$1`, id))
}

func GetPaymentIntentByOrder(ctx context.Context, pool *pgxpool.Pool, order string) (*PaymentIntent, error) {
	return scanPaymentIntent(pool.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE gateway_order_id=$1`, order))
}

// Every payment writer locks the invoice first; this serializes installments and reservations.
func lockPaymentInvoice(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	var total string
	err := tx.QueryRow(ctx, `SELECT total::text FROM invoices WHERE id=$1
        AND sales_order_id IS NOT NULL AND (UPPER(COALESCE(payment_settlement,''))='TERM' OR payment_terms_days>0)
        AND UPPER(COALESCE(status,'')) NOT IN ('VOID','CANCELLED','CANCELED') FOR UPDATE`, id).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPaymentInvoice
	}
	if err != nil {
		return 0, err
	}
	return paymentlogic.Cents(total)
}

func syncPaymentInvoiceStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE invoices SET status=CASE WHEN
        COALESCE((SELECT SUM(amount-refunded_amount) FROM invoice_payments WHERE invoice_id=$1 AND status='CONFIRMED'),0)>=total
        THEN 'PAID' ELSE 'UNPAID' END,updated_at=NOW() WHERE id=$1`, id)
	return err
}

func CreatePaymentIntent(ctx context.Context, pool *pgxpool.Pool, input PaymentInput) (*PaymentIntent, error) {
	amount, err := paymentlogic.Cents(input.Amount)
	if err != nil || amount <= 0 {
		return nil, paymentlogic.ErrAmount
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	total, err := lockPaymentInvoice(ctx, tx, input.InvoiceID)
	if err != nil {
		return nil, err
	}
	existing, err := scanPaymentIntent(tx.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE request_key=$1`, input.RequestKey))
	if err == nil {
		oldAmount, _ := paymentlogic.Cents(existing.Amount)
		if existing.InvoiceID != input.InvoiceID || oldAmount != amount || existing.Source != input.Source || existing.Actor != input.Actor || (input.Source == "MANUAL" && existing.Method != input.Method) || (input.Source == "MIDTRANS" && existing.Environment != input.Environment) || existing.Reference != input.Reference || existing.Notes != input.Notes || (input.Source == "MANUAL" && !existing.PaidAt.Equal(input.PaidAt)) {
			return nil, ErrPaymentConflict
		}
		return existing, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var paidText, reservedText string
	err = tx.QueryRow(ctx, `SELECT
        COALESCE((SELECT SUM(amount-refunded_amount) FROM invoice_payments WHERE invoice_id=$1 AND status='CONFIRMED'),0)::text,
        COALESCE((SELECT SUM(amount) FROM finance_payment_intents WHERE invoice_id=$1 AND source='MIDTRANS' AND state IN ('CREATING','PENDING')),0)::text`, input.InvoiceID).Scan(&paidText, &reservedText)
	if err != nil {
		return nil, err
	}
	paid, err := paymentlogic.Cents(paidText)
	if err != nil {
		return nil, err
	}
	reserved, err := paymentlogic.Cents(reservedText)
	if err != nil {
		return nil, err
	}
	if amount > total-paid-reserved {
		return nil, ErrPaymentConflict
	}
	id := uuid.New()
	order := ""
	state := "CONFIRMED"
	if input.Source == "MIDTRANS" {
		order = "FAS-" + strings.ReplaceAll(id.String(), "-", "")
		state = "CREATING"
	}
	expires := time.Now().UTC().Add(30 * time.Minute)
	_, err = tx.Exec(ctx, `INSERT INTO finance_payment_intents
        (id,request_key,invoice_id,amount,source,state,method,reference_number,notes,paid_at,created_by,gateway_order_id,environment,expires_at)
        VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14)`,
		id, input.RequestKey, input.InvoiceID, paymentlogic.Decimal(amount), input.Source, state, input.Method, input.Reference, input.Notes, input.PaidAt, input.Actor, order, input.Environment, expires)
	if err != nil {
		return nil, err
	}
	if input.Source == "MANUAL" {
		_, err = tx.Exec(ctx, `INSERT INTO invoice_payments(id,invoice_id,payment_number,amount,paid_at,status,notes,source,method,reference_number,created_by,intent_id)
            VALUES ($1,$2,$3,$4::numeric,$5,'CONFIRMED',$6,'MANUAL',$7,$8,$9,$1)`, id, input.InvoiceID, "PAY-"+id.String(), paymentlogic.Decimal(amount), input.PaidAt, input.Notes, input.Method, input.Reference, input.Actor)
		if err != nil {
			return nil, err
		}
		if err = syncPaymentInvoiceStatus(ctx, tx, input.InvoiceID); err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance_payment_events(invoice_id,intent_id,payment_id,action,actor,detail)
        VALUES ($1,$2,$3,$4,$5,$6)`, input.InvoiceID, id, func() any {
		if input.Source == "MANUAL" {
			return id
		}
		return nil
	}(), "CREATE_"+input.Source, input.Actor, paymentlogic.Decimal(amount))
	if err != nil {
		return nil, err
	}
	item, err := scanPaymentIntent(tx.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE id=$1`, id))
	if err != nil {
		return nil, err
	}
	return item, tx.Commit(ctx)
}

func SavePaymentSnap(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, token, redirect string) error {
	_, err := pool.Exec(ctx, `UPDATE finance_payment_intents SET snap_token=$2,redirect_url=$3,
        state=CASE WHEN state='CREATING' THEN 'PENDING' ELSE state END,updated_at=NOW()
        WHERE id=$1 AND state IN ('CREATING','PENDING')`, id, token, redirect)
	return err
}

type GatewayPaymentUpdate struct {
	OrderID       string
	Amount        string
	Status        string
	Method        string
	TransactionID string
	Paid          bool
	Refund        string
	PaidAt        time.Time
}

// Verified provider status only. Replay cannot duplicate the ledger or reverse a refund.
func ApplyGatewayPayment(ctx context.Context, pool *pgxpool.Pool, update GatewayPaymentUpdate) error {
	intent, err := GetPaymentIntentByOrder(ctx, pool, update.OrderID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = lockPaymentInvoice(ctx, tx, intent.InvoiceID)
	if err != nil {
		return err
	}
	intent, err = scanPaymentIntent(tx.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE id=$1 FOR UPDATE`, intent.ID))
	if err != nil {
		return err
	}
	amount, err := paymentlogic.Cents(update.Amount)
	if err != nil {
		return err
	}
	expected, _ := paymentlogic.Cents(intent.Amount)
	if amount != expected {
		return ErrPaymentConflict
	}
	if update.Status == "partial_refund" && update.Refund == "" {
		return ErrPaymentConflict
	}
	refunded := int64(0)
	if update.Refund != "" {
		refunded, err = paymentlogic.Cents(update.Refund)
		if err != nil || refunded > amount {
			return ErrPaymentConflict
		}
	}
	state := intent.State
	if update.Paid || update.Status == "refund" || update.Status == "partial_refund" {
		if state == "RELEASED" {
			return ErrPaymentConflict
		}
		if update.Status == "refund" {
			refunded = amount
		}
		var changed int64
		// Insert one receipt per intent. A partial refund keeps the original receipt amount.
		result, insertErr := tx.Exec(ctx, `INSERT INTO invoice_payments
            (id,invoice_id,payment_number,amount,paid_at,status,notes,source,method,reference_number,created_by,intent_id,refunded_amount)
            VALUES ($1,$2,$3,$4::numeric,$5,'CONFIRMED',$6,'MIDTRANS',$7,$8,$9,$1,$10::numeric)
            ON CONFLICT(intent_id) DO UPDATE SET refunded_amount=GREATEST(invoice_payments.refunded_amount,EXCLUDED.refunded_amount),
                status=CASE WHEN GREATEST(invoice_payments.refunded_amount,EXCLUDED.refunded_amount)>=invoice_payments.amount THEN 'VOID' ELSE 'CONFIRMED' END
            WHERE EXCLUDED.refunded_amount>invoice_payments.refunded_amount`,
			intent.ID, intent.InvoiceID, "PAY-"+intent.ID.String(), intent.Amount, update.PaidAt, intent.Notes, update.Method, update.TransactionID, intent.Actor, paymentlogic.Decimal(refunded))
		if insertErr != nil {
			return insertErr
		}
		changed = result.RowsAffected()
		var net, originalRefund string
		err = tx.QueryRow(ctx, `SELECT refunded_amount::text,(amount-refunded_amount)::text FROM invoice_payments WHERE intent_id=$1`, intent.ID).Scan(&originalRefund, &net)
		if err != nil {
			return err
		}
		refundNow, _ := paymentlogic.Cents(originalRefund)
		if refundNow >= amount {
			state = "REFUNDED"
			_, err = tx.Exec(ctx, `UPDATE invoice_payments SET status='VOID' WHERE intent_id=$1`, intent.ID)
		} else {
			state = "CONFIRMED"
		}
		if err != nil {
			return err
		}
		if changed > 0 {
			_, err = tx.Exec(ctx, `INSERT INTO finance_payment_events(invoice_id,intent_id,payment_id,action,actor,detail)
                VALUES ($1,$2,$2,'MIDTRANS_VERIFIED','MIDTRANS',$3)`, intent.InvoiceID, intent.ID, update.Status+"; net="+net)
			if err != nil {
				return err
			}
		}
	} else if (update.Status == "cancel" || update.Status == "deny") && (state == "CONFIRMED" || state == "REFUNDED") {
		// Card capture can be voided before settlement. Ignore plain deny/pending regressions.
		if update.Status == "cancel" && state != "REFUNDED" {
			_, err = tx.Exec(ctx, `UPDATE invoice_payments SET status='VOID',refunded_amount=amount WHERE intent_id=$1`, intent.ID)
			if err != nil {
				return err
			}
			state = "REFUNDED"
			_, err = tx.Exec(ctx, `INSERT INTO finance_payment_events(invoice_id,intent_id,payment_id,action,actor,detail) VALUES ($1,$2,$2,'MIDTRANS_VOID','MIDTRANS','cancel verified')`, intent.InvoiceID, intent.ID)
			if err != nil {
				return err
			}
		}
	} else if state == "CREATING" {
		state = "PENDING"
	}
	// Failed attempts keep their reservation until Snap itself is cancelled.
	// Snap may permit another attempt with the same order; status=404 alone never releases money.
	_, err = tx.Exec(ctx, `UPDATE finance_payment_intents SET state=$2,gateway_status=$3,
        gateway_transaction_id=$4,method=CASE WHEN $5='' THEN method ELSE $5 END,updated_at=NOW() WHERE id=$1`,
		intent.ID, state, update.Status, update.TransactionID, update.Method)
	if err != nil {
		return err
	}
	if state == "CONFIRMED" || state == "REFUNDED" {
		if err = syncPaymentInvoiceStatus(ctx, tx, intent.InvoiceID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func ReleasePaymentIntent(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actor string) error {
	item, err := GetPaymentIntent(ctx, pool, id)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = lockPaymentInvoice(ctx, tx, item.InvoiceID)
	if err != nil {
		return err
	}
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoice_payments WHERE intent_id=$1 AND status='CONFIRMED')`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrPaymentConflict
	}
	result, err := tx.Exec(ctx, `UPDATE finance_payment_intents SET state='RELEASED',gateway_status='session_cancelled',updated_at=NOW()
        WHERE id=$1 AND source='MIDTRANS' AND state IN ('CREATING','PENDING')`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		_, err = tx.Exec(ctx, `INSERT INTO finance_payment_events(invoice_id,intent_id,action,actor,detail) VALUES ($1,$2,'CANCEL_SNAP',$3,'Sesi dibatalkan di Midtrans')`, item.InvoiceID, id, actor)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func VoidManualPayment(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actor, reason string) error {
	var invoiceID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT invoice_id FROM invoice_payments WHERE id=$1`, id).Scan(&invoiceID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = lockPaymentInvoice(ctx, tx, invoiceID)
	if err != nil {
		return err
	}
	var source, status string
	var intentID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT source,status,intent_id FROM invoice_payments WHERE id=$1 FOR UPDATE`, id).Scan(&source, &status, &intentID)
	if err != nil {
		return err
	}
	if source != "MANUAL" {
		return ErrPaymentConflict
	}
	if status == "VOID" {
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `UPDATE invoice_payments SET status='VOID',voided_by=$2,void_reason=$3,voided_at=NOW() WHERE id=$1`, id, actor, reason)
	if err != nil {
		return err
	}
	if intentID != nil {
		_, err = tx.Exec(ctx, `UPDATE finance_payment_intents SET state='VOID',updated_at=NOW() WHERE id=$1`, intentID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance_payment_events(invoice_id,intent_id,payment_id,action,actor,detail) VALUES ($1,$2,$3,'VOID_MANUAL',$4,$5)`, invoiceID, intentID, id, actor, reason)
	if err != nil {
		return err
	}
	if err = syncPaymentInvoiceStatus(ctx, tx, invoiceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func PaymentInvoiceDetails(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `WITH invoice AS (
        SELECT i.id,i.invoice_number,i.store_name,i.customer_code,i.order_number,i.invoice_date,i.due_date,i.payment_terms_days,
            i.total::text AS total,COALESCE(p.paid,0)::text AS paid_amount,
            GREATEST(i.total-COALESCE(p.paid,0),0)::text AS outstanding,
            COALESCE(r.reserved,0)::text AS reserved_amount,
            GREATEST(i.total-COALESCE(p.paid,0)-COALESCE(r.reserved,0),0)::text AS available_amount
        FROM invoices i
        LEFT JOIN LATERAL (SELECT SUM(amount-refunded_amount) AS paid FROM invoice_payments WHERE invoice_id=i.id AND status='CONFIRMED') p ON TRUE
        LEFT JOIN LATERAL (SELECT SUM(amount) AS reserved FROM finance_payment_intents WHERE invoice_id=i.id AND source='MIDTRANS' AND state IN ('CREATING','PENDING')) r ON TRUE
        WHERE i.id=$1 AND i.sales_order_id IS NOT NULL AND (UPPER(COALESCE(i.payment_settlement,''))='TERM' OR i.payment_terms_days>0)
            AND UPPER(COALESCE(i.status,'')) NOT IN ('VOID','CANCELLED','CANCELED')
        ) SELECT JSONB_BUILD_OBJECT('invoice',(SELECT TO_JSONB(invoice) FROM invoice),
        'payments',COALESCE((SELECT JSONB_AGG(TO_JSONB(x) ORDER BY x.paid_at DESC,x.id) FROM (
            SELECT id,payment_number,amount::text,refunded_amount::text,(amount-refunded_amount)::text AS net_amount,paid_at,status,source,method,
                reference_number,notes,created_by,COALESCE((SELECT name FROM users WHERE id::text=invoice_payments.created_by),NULLIF(created_by,''),'Data sebelumnya') AS created_by_name,voided_by,voided_at,void_reason FROM invoice_payments WHERE invoice_id=$1) x),'[]'::jsonb),
        'intents',COALESCE((SELECT JSONB_AGG(TO_JSONB(x) ORDER BY x.created_at DESC,x.id) FROM (
            SELECT id,amount::text,source,state,gateway_status,gateway_order_id,redirect_url,environment,created_at,expires_at FROM finance_payment_intents WHERE invoice_id=$1 AND source='MIDTRANS') x),'[]'::jsonb),
        'events',COALESCE((SELECT JSONB_AGG(TO_JSONB(x) ORDER BY x.created_at DESC,x.id DESC) FROM (
            SELECT id,action,COALESCE((SELECT name FROM users WHERE id::text=finance_payment_events.actor),actor) AS actor,detail,created_at FROM finance_payment_events WHERE invoice_id=$1) x),'[]'::jsonb))`, id).Scan(&payload)
	return json.RawMessage(payload), err
}

type PaymentListFilter struct {
	Query, DateFrom, DateTo, Status string
	Limit, Offset                   int
}

func ListInvoicePayments(ctx context.Context, pool *pgxpool.Pool, f PaymentListFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `WITH filtered AS (
        SELECT p.id,p.invoice_id,i.invoice_number,i.store_name,p.payment_number,p.amount::text AS amount,
            p.refunded_amount::text AS refunded_amount,(CASE WHEN p.status='CONFIRMED' THEN p.amount-p.refunded_amount ELSE 0 END)::text AS net_amount,
            p.paid_at,p.status,p.source,p.method,p.reference_number,p.notes,p.created_by,p.void_reason
        FROM invoice_payments p JOIN invoices i ON i.id=p.invoice_id
        WHERE ($1='' OR STRPOS(LOWER(CONCAT_WS(' ',i.invoice_number,i.store_name,p.payment_number,p.reference_number)),LOWER($1))>0)
            AND ($2='' OR (p.paid_at AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($2,'')::date)
            AND ($3='' OR (p.paid_at AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($3,'')::date)
            AND ($4='' OR $4='ALL' OR p.status=$4)
        ), page AS (SELECT * FROM filtered ORDER BY paid_at DESC,id LIMIT $5 OFFSET $6)
        SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page) ORDER BY paid_at DESC,id) FROM page),'[]'::jsonb),
            'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered),'limit',$5::int,'offset',$6::int),
            'summary',JSONB_BUILD_OBJECT('net_amount',COALESCE((SELECT SUM(net_amount::numeric) FROM filtered),0)::text))`,
		f.Query, f.DateFrom, f.DateTo, f.Status, f.Limit, f.Offset).Scan(&payload)
	return json.RawMessage(payload), err
}

// Serialize Snap creation per intent. Retrying the same request must not create competing tokens.
func EnsurePaymentSnap(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, create func(*PaymentIntent, string) (string, string, error)) (*PaymentIntent, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	item, err := scanPaymentIntent(tx.QueryRow(ctx, `SELECT `+intentColumns+` FROM finance_payment_intents WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return nil, err
	}
	if item.Token != "" || item.State != "CREATING" {
		return item, tx.Commit(ctx)
	}
	var customer string
	if err = tx.QueryRow(ctx, `SELECT store_name FROM invoices WHERE id=$1`, item.InvoiceID).Scan(&customer); err != nil {
		return nil, err
	}
	token, redirect, err := create(item, customer)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE finance_payment_intents SET snap_token=$2,redirect_url=$3,state='PENDING',updated_at=NOW() WHERE id=$1`, id, token, redirect)
	if err != nil {
		return nil, err
	}
	item.Token = token
	item.RedirectURL = redirect
	item.State = "PENDING"
	return item, tx.Commit(ctx)
}
