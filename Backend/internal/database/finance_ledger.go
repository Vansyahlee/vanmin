package database

import (
	"context"
	"encoding/json"
	"errors"
	"furi-backend/internal/expenselogic"
	"furi-backend/internal/paymentlogic"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrExpenseConflict = errors.New("kunci pengeluaran sudah dipakai untuk data berbeda atau status berubah")

type LedgerFilter struct {
	Query, DateFrom, DateTo, Status, Category, Method, Account, Direction, Read, Actor, Kind string
	Limit, Offset                                                                            int
}
type ExpenseInput struct {
	Kind                                                                           string
	Details                                                                        expenselogic.Details
	ID, RequestKey                                                                 uuid.UUID
	Date, Category, Payee, Description, Amount, Method, Reference, ProofURL, Actor string
}

func CreateExpense(ctx context.Context, pool *pgxpool.Pool, in ExpenseInput) (uuid.UUID, error) {
	if in.Kind == "" {
		in.Kind = "GENERAL"
	}
	details, detailErr := expenselogic.Normalize(in.Kind, in.Details, in.Amount)
	if detailErr != nil {
		return uuid.Nil, detailErr
	}
	in.Category = expenselogic.Category(in.Kind, in.Category)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	amount, err := paymentlogic.Cents(in.Amount)
	if err != nil {
		return uuid.Nil, err
	}
	in.Amount = paymentlogic.Decimal(amount)
	_, err = tx.Exec(ctx, `INSERT INTO finance_expenses(id,request_key,expense_number,expense_date,category,payee,description,amount,method,reference_number,proof_url,created_by,expense_kind,details)
 VALUES($1,$2,$3,$4::date,$5,$6,$7,$8::numeric,$9,$10,$11,$12,$13,$14::jsonb) ON CONFLICT(request_key) DO NOTHING`,
		in.ID, in.RequestKey, "EXP-"+in.ID.String(), in.Date, in.Category, in.Payee, in.Description, in.Amount, in.Method, in.Reference, in.ProofURL, in.Actor, in.Kind, string(details))
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	var same bool
	err = tx.QueryRow(ctx, `SELECT id,expense_date=$2::date AND category=$3 AND payee=$4 AND description=$5 AND amount=$6::numeric AND method=$7 AND reference_number=$8 AND proof_url=$9 AND created_by=$10 AND expense_kind=$11 AND details=$12::jsonb
 FROM finance_expenses WHERE request_key=$1`, in.RequestKey, in.Date, in.Category, in.Payee, in.Description, in.Amount, in.Method, in.Reference, in.ProofURL, in.Actor, in.Kind, string(details)).Scan(&id, &same)
	if err != nil {
		return uuid.Nil, err
	}
	if !same {
		return uuid.Nil, ErrExpenseConflict
	}
	return id, tx.Commit(ctx)
}
func VoidExpense(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actor, reason string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM finance_expenses WHERE id=$1 FOR UPDATE`, id).Scan(&status)
	if err != nil {
		return err
	}
	if status == "VOID" {
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `UPDATE finance_expenses SET status='VOID',voided_at=NOW(),voided_by=$2,void_reason=$3 WHERE id=$1`, id, actor, reason)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func ListExpenses(ctx context.Context, pool *pgxpool.Pool, f LedgerFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `WITH movements AS (
 SELECT m.*,CASE WHEN direction='IN' THEN amount ELSE -amount END AS signed,
 SUM(CASE WHEN direction='IN' THEN amount ELSE -amount END) OVER(PARTITION BY account ORDER BY occurred_at,id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS balance_after
 FROM finance_cash_movements m
 ), cash_scope AS (
 SELECT * FROM movements WHERE account IN ('CASH','BANK') AND ($6='' OR account=CASE WHEN $6='CASH' THEN 'CASH' ELSE 'BANK' END)
 ), cash_period AS (
 SELECT * FROM cash_scope WHERE ($2='' OR (occurred_at AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($2,'')::date)
 AND ($3='' OR (occurred_at AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($3,'')::date)
 ), cash_opening AS (
 SELECT COALESCE(SUM(signed),0) AS amount FROM cash_scope WHERE $2<>'' AND (occurred_at AT TIME ZONE 'Asia/Jakarta')::date<NULLIF($2,'')::date
 ), filtered AS (
 SELECT e.*,e.amount::text AS amount_text,COALESCE(u.name,e.created_by) AS created_by_name,
 m.account AS cash_account,m.occurred_at AS movement_at,(m.balance_after-m.signed)::text AS balance_before,m.balance_after::text AS balance_after
 FROM finance_expenses e LEFT JOIN users u ON u.id::text=e.created_by
 LEFT JOIN movements m ON m.expense_id=e.id AND m.kind='EXPENSE' 
 WHERE ($1='' OR STRPOS(LOWER(CONCAT_WS(' ',e.expense_number,e.payee,e.description,e.reference_number,e.category)),LOWER($1))>0)
 AND ($2='' OR expense_date>=NULLIF($2,'')::date) AND ($3='' OR expense_date<=NULLIF($3,'')::date)
 AND ($4='ALL' OR e.status=$4) AND ($5='' OR category=$5) AND ($6='' OR method=$6) AND ($9='' OR expense_kind=$9)
 ), page AS(SELECT * FROM filtered ORDER BY expense_date DESC,created_at DESC,id LIMIT $7 OFFSET $8)
 SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page)-'amount_text'||JSONB_BUILD_OBJECT('amount',amount_text) ORDER BY expense_date DESC,created_at DESC,id) FROM page),'[]'::jsonb),
 'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered)),
 'summary',JSONB_BUILD_OBJECT('posted_amount',COALESCE((SELECT SUM(amount) FROM filtered WHERE status='POSTED'),0)::text,
 'void_amount',COALESCE((SELECT SUM(amount) FROM filtered WHERE status='VOID'),0)::text,
 'cash_opening',(SELECT amount::text FROM cash_opening),
 'cash_in',COALESCE((SELECT SUM(amount) FROM cash_period WHERE direction='IN'),0)::text,
 'cash_out',COALESCE((SELECT SUM(amount) FROM cash_period WHERE direction='OUT'),0)::text,
 'cash_closing',((SELECT amount FROM cash_opening)+COALESCE((SELECT SUM(signed) FROM cash_period),0))::text))`, f.Query, f.DateFrom, f.DateTo, f.Status, f.Category, f.Method, f.Limit, f.Offset, f.Kind).Scan(&payload)
	return json.RawMessage(payload), err
}
func CashLedger(ctx context.Context, pool *pgxpool.Pool, f LedgerFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `WITH scope AS(
 SELECT m.*, (CASE WHEN direction='IN' THEN amount ELSE -amount END) AS signed FROM finance_cash_movements m
 WHERE ($1='' OR account=$1)
 ), opening AS(SELECT COALESCE(SUM(signed),0) AS amount FROM scope WHERE $2<>'' AND (occurred_at AT TIME ZONE 'Asia/Jakarta')::date<NULLIF($2,'')::date),
 period AS(SELECT * FROM scope WHERE ($2='' OR (occurred_at AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($2,'')::date)
 AND ($3='' OR (occurred_at AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($3,'')::date)),
 running AS(SELECT *,((SELECT amount FROM opening)+SUM(signed) OVER(ORDER BY occurred_at,id))::text AS running_balance FROM period),
 filtered AS(SELECT * FROM running WHERE ($4='' OR STRPOS(LOWER(CONCAT_WS(' ',reference_number,party,description,kind)),LOWER($4))>0) AND ($5='' OR direction=$5)),
 page AS(SELECT * FROM filtered ORDER BY occurred_at DESC,id DESC LIMIT $6 OFFSET $7)
 SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page)-'signed'||JSONB_BUILD_OBJECT('id',id::text,'amount',amount::text) ORDER BY occurred_at DESC,id DESC) FROM page),'[]'::jsonb),
 'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered)),
 'summary',JSONB_BUILD_OBJECT('opening',(SELECT amount::text FROM opening),
 'in_amount',COALESCE((SELECT SUM(amount) FROM period WHERE direction='IN'),0)::text,
 'out_amount',COALESCE((SELECT SUM(amount) FROM period WHERE direction='OUT'),0)::text,
 'closing',((SELECT amount FROM opening)+COALESCE((SELECT SUM(signed) FROM period),0))::text))`, f.Account, f.DateFrom, f.DateTo, f.Query, f.Direction, f.Limit, f.Offset).Scan(&payload)
	return json.RawMessage(payload), err
}
func ListFinanceNotifications(ctx context.Context, pool *pgxpool.Pool, f LedgerFilter) (json.RawMessage, error) {
	var payload []byte
	err := pool.QueryRow(ctx, `WITH base AS(
 SELECT n.*,n.id::text AS key_id,(n.is_historical OR nr.notification_id IS NOT NULL) AS is_read,p.status AS payment_status,
 (CASE WHEN p.status='CONFIRMED' THEN p.amount-p.refunded_amount ELSE 0 END)::text AS current_payment_net
 FROM finance_notifications n LEFT JOIN finance_notification_reads nr ON nr.notification_id=n.id AND nr.user_id=$1
 JOIN invoice_payments p ON p.id=n.payment_id
 ), filtered AS(SELECT * FROM base WHERE ($2='' OR STRPOS(LOWER(CONCAT_WS(' ',customer_name,invoice_number,reference_number)),LOWER($2))>0)
 AND ($3='' OR (created_at AT TIME ZONE 'Asia/Jakarta')::date>=NULLIF($3,'')::date)
 AND ($4='' OR (created_at AT TIME ZONE 'Asia/Jakarta')::date<=NULLIF($4,'')::date)
 AND ($5='' OR source=$5) AND ($6='ALL' OR ($6='UNREAD' AND NOT is_read) OR ($6='READ' AND is_read))),
 page AS(SELECT * FROM filtered ORDER BY created_at DESC,id DESC LIMIT $7 OFFSET $8)
 SELECT JSONB_BUILD_OBJECT('data',COALESCE((SELECT JSONB_AGG(TO_JSONB(page)-'key_id'||JSONB_BUILD_OBJECT('id',key_id,'amount',amount::text) ORDER BY created_at DESC,id DESC) FROM page),'[]'::jsonb),
 'meta',JSONB_BUILD_OBJECT('total',(SELECT COUNT(*) FROM filtered),'through_id',COALESCE((SELECT MAX(id)::text FROM base),'0')),
 'unread_count',(SELECT COUNT(*) FROM base WHERE NOT is_read))`, f.Actor, f.Query, f.DateFrom, f.DateTo, f.Method, f.Read, f.Limit, f.Offset).Scan(&payload)
	return json.RawMessage(payload), err
}
func ReadFinanceNotifications(ctx context.Context, pool *pgxpool.Pool, actor string, id int64, all bool) error {
	if !all {
		var found bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_notifications WHERE id=$1)`, id).Scan(&found)
		if err != nil {
			return err
		}
		if !found {
			return pgx.ErrNoRows
		}
	}
	_, err := pool.Exec(ctx, `INSERT INTO finance_notification_reads(notification_id,user_id)
 SELECT id,$1 FROM finance_notifications WHERE ($3 AND id<=$2) OR (NOT $3 AND id=$2)
 ON CONFLICT(notification_id,user_id) DO NOTHING`, actor, id, all)
	return err
}
