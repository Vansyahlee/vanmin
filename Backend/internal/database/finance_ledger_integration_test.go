package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Explicit test DSN only; all fixtures and cleanup stay within a unique test schema.
func TestFinanceLedgerNotificationsAndIssuer(t *testing.T) {
	dsn := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PAYMENT_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "ledger_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `CREATE TABLE users(id UUID PRIMARY KEY,name TEXT,role TEXT,is_active BOOLEAN);
 CREATE TABLE invoices(id UUID PRIMARY KEY,invoice_number TEXT,store_name TEXT,is_printed BOOLEAN DEFAULT FALSE,sales_order_id UUID,payment_settlement TEXT,payment_terms_days INT,status TEXT,total NUMERIC(18,2),updated_at TIMESTAMPTZ);
 CREATE TABLE invoice_payments(id UUID PRIMARY KEY,invoice_id UUID REFERENCES invoices(id),payment_number TEXT UNIQUE,amount NUMERIC(18,2),paid_at TIMESTAMPTZ,status TEXT,notes TEXT,created_at TIMESTAMPTZ DEFAULT NOW());`)
	if err != nil {
		t.Fatal(err)
	}
	migrate := func(name string) {
		t.Helper()
		sql, e := os.ReadFile("../../migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, string(sql)); e != nil {
			t.Fatal(e)
		}
	}
	migrate("20261014_payments_midtrans.sql")
	migrate("20261015_expenses_notifications.sql")
	migrate("20261015_company_profile.sql")
	migrate("20261015_expense_payroll_supplier.sql")
	invoiceID := uuid.New()
	actor := uuid.NewString()
	_, err = pool.Exec(ctx, `INSERT INTO users VALUES($1,'Owner Uji','admin',TRUE);`, actor)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO invoices(id,invoice_number,store_name,sales_order_id,payment_settlement,payment_terms_days,status,total) VALUES($1,'INV-TEST','Toko Uji',$2,'TERM',30,'UNPAID',8000000)`, invoiceID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	input := PaymentInput{InvoiceID: invoiceID, RequestKey: uuid.New(), Amount: "2000000", Source: "MANUAL", Method: "CASH", Actor: actor, PaidAt: time.Now().UTC().Truncate(time.Microsecond), Environment: "sandbox"}
	receipt, err := CreatePaymentIntent(ctx, pool, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CreatePaymentIntent(ctx, pool, input); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM finance_notifications`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate notification", count, err)
	}
	var payload struct {
		Unread int `json:"unread_count"`
		Meta   struct {
			Through string `json:"through_id"`
		} `json:"meta"`
	}
	filter := LedgerFilter{Actor: actor, Read: "ALL", Status: "ALL", Limit: 50}
	raw, err := ListFinanceNotifications(ctx, pool, filter)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &payload); err != nil || payload.Unread != 1 {
		t.Fatal("unread", string(raw), err)
	}
	if err = ReadFinanceNotifications(ctx, pool, actor, 1, true); err != nil {
		t.Fatal(err)
	}
	filter.Actor = "other-user"
	raw, err = ListFinanceNotifications(ctx, pool, filter)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(raw, &payload)
	if payload.Unread != 1 {
		t.Fatal("read state leaked across users")
	}
	expense := ExpenseInput{ID: uuid.New(), RequestKey: uuid.New(), Date: time.Now().UTC().Format("2006-01-02"), Category: "Internet", Payee: "Provider", Description: "Internet kantor", Amount: "100000", Method: "CASH", Actor: actor}
	id, err := CreateExpense(ctx, pool, expense)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := CreateExpense(ctx, pool, expense)
	if err != nil || replay != id {
		t.Fatal("expense replay", err)
	}
	changed := expense
	changed.Amount = "200000"
	if _, err = CreateExpense(ctx, pool, changed); !errors.Is(err, ErrExpenseConflict) {
		t.Fatal("changed payload accepted", err)
	}
	for n := 0; n < 2; n++ {
		if err = VoidExpense(ctx, pool, id, actor, "Salah catat pengeluaran"); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 2; n++ {
		if err = VoidManualPayment(ctx, pool, receipt.ID, actor, "Salah catat pembayaran"); err != nil {
			t.Fatal(err)
		}
	}
	var net string
	err = pool.QueryRow(ctx, `SELECT SUM(CASE WHEN direction='IN' THEN amount ELSE -amount END)::text FROM finance_cash_movements`).Scan(&net)
	if err != nil || net != "0.00" {
		t.Fatal("reversal net", net, err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM finance_cash_movements`).Scan(&count); err != nil || count != 4 {
		t.Fatal("duplicate reversal", count, err)
	}
	migrate("20261015_expenses_notifications.sql")
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM finance_notifications`).Scan(&count); err != nil || count != 2 {
		t.Fatal("migration replay", count, err)
	}
	var before, after json.RawMessage
	if err = pool.QueryRow(ctx, `SELECT issuer_profile FROM invoices WHERE id=$1`, invoiceID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE company_profile SET address='Alamat Baru' WHERE id=TRUE`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE invoices SET issuer_profile='{}'::jsonb WHERE id=$1`, invoiceID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT issuer_profile FROM invoices WHERE id=$1`, invoiceID).Scan(&after); err != nil || string(before) != string(after) {
		t.Fatal("issuer history changed", err)
	}
}
