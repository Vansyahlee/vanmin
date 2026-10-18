package database

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only against an explicitly supplied TEST database; creates and drops its own schema.
func TestInstallmentConcurrencyAndGatewayReplay(t *testing.T) {
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
	schema := "payment_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	_, err = pool.Exec(ctx, `CREATE TABLE invoices(id UUID PRIMARY KEY,sales_order_id UUID,payment_settlement TEXT,payment_terms_days INT,status TEXT,total NUMERIC(18,2),updated_at TIMESTAMPTZ,store_name TEXT);
        CREATE TABLE invoice_payments(id UUID PRIMARY KEY,invoice_id UUID REFERENCES invoices(id),payment_number TEXT UNIQUE,amount NUMERIC(18,2),paid_at TIMESTAMPTZ,status TEXT,notes TEXT,created_at TIMESTAMPTZ DEFAULT NOW());`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/20261014_payments_midtrans.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	invoiceID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO invoices VALUES ($1,$2,'TERM',30,'UNPAID',8000000,NOW(),'Toko Uji')`, invoiceID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	input := PaymentInput{InvoiceID: invoiceID, RequestKey: uuid.New(), Amount: "2000000", Source: "MANUAL", Method: "CASH", Actor: uuid.NewString(), PaidAt: time.Now().UTC().Truncate(time.Microsecond), Environment: "sandbox"}
	first, err := CreatePaymentIntent(ctx, pool, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := CreatePaymentIntent(ctx, pool, input)
	if err != nil || first.ID != replay.ID {
		t.Fatal("idempotent replay", err)
	}
	var sum string
	balance := func() string {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT SUM(amount-refunded_amount)::text FROM invoice_payments WHERE invoice_id=$1 AND status='CONFIRMED'`, invoiceID).Scan(&sum); err != nil {
			t.Fatal(err)
		}
		return sum
	}
	if balance() != "2000000.00" {
		t.Fatal("first installment", sum)
	}
	var wg sync.WaitGroup
	outcomes := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attempt := input
			attempt.RequestKey = uuid.New()
			attempt.Amount = "4000000"
			_, e := CreatePaymentIntent(ctx, pool, attempt)
			outcomes <- e
		}()
	}
	wg.Wait()
	close(outcomes)
	success, conflict := 0, 0
	for err := range outcomes {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPaymentConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 || balance() != "6000000.00" {
		t.Fatal("concurrency", success, conflict, sum)
	}
	gateway := input
	gateway.RequestKey = uuid.New()
	gateway.Source = "MIDTRANS"
	gateway.Method = "MIDTRANS"
	intent, err := CreatePaymentIntent(ctx, pool, gateway)
	if err != nil {
		t.Fatal(err)
	}
	var snapCalls int32
	var snapWG sync.WaitGroup
	snapErrors := make(chan error, 2)
	for n := 0; n < 2; n++ {
		snapWG.Add(1)
		go func() {
			defer snapWG.Done()
			created, e := EnsurePaymentSnap(ctx, pool, intent.ID, func(item *PaymentIntent, customer string) (string, string, error) {
				atomic.AddInt32(&snapCalls, 1)
				return "stable-token", "https://app.sandbox.midtrans.com/snap/v2/vtweb/stable-token", nil
			})
			if e == nil && created.Token != "stable-token" {
				e = errors.New("unstable Snap token")
			}
			snapErrors <- e
		}()
	}
	snapWG.Wait()
	close(snapErrors)
	for e := range snapErrors {
		if e != nil {
			t.Fatal(e)
		}
	}
	if snapCalls != 1 {
		t.Fatal("duplicate Snap creation", snapCalls)
	}
	blocked := input
	blocked.RequestKey = uuid.New()
	if _, err = CreatePaymentIntent(ctx, pool, blocked); !errors.Is(err, ErrPaymentConflict) {
		t.Fatal("reservation must block manual overpayment", err)
	}
	status := GatewayPaymentUpdate{OrderID: intent.OrderID, Amount: "2000000.00", Status: "settlement", Paid: true, Method: "bank_transfer", TransactionID: "test-midtrans", PaidAt: time.Now().UTC()}
	for n := 0; n < 2; n++ {
		if err = ApplyGatewayPayment(ctx, pool, status); err != nil {
			t.Fatal(err)
		}
	}
	if balance() != "8000000.00" {
		t.Fatal("gateway duplicated", sum)
	}
	if replay, e := CreatePaymentIntent(ctx, pool, gateway); e != nil || replay.ID != intent.ID {
		t.Fatal("replay after provider method update", e)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM invoice_payments WHERE invoice_id=$1`, invoiceID).Scan(&count); err != nil || count != 3 {
		t.Fatal("receipt count", count, err)
	}
	status.Status = "partial_refund"
	status.Paid = false
	status.Refund = "1000000.00"
	if err = ApplyGatewayPayment(ctx, pool, status); err != nil {
		t.Fatal(err)
	}
	status.Status = "settlement"
	status.Paid = true
	status.Refund = ""
	if err = ApplyGatewayPayment(ctx, pool, status); err != nil {
		t.Fatal(err)
	}
	if balance() != "7000000.00" {
		t.Fatal("old callback reversed refund", sum)
	}
	if err = VoidManualPayment(ctx, pool, first.ID, input.Actor, "Catatan uji dibatalkan"); err != nil {
		t.Fatal(err)
	}
	if err = VoidManualPayment(ctx, pool, first.ID, input.Actor, "Retry pembatalan"); err != nil {
		t.Fatal(err)
	}
	if balance() != "5000000.00" {
		t.Fatal("void replay balance", sum)
	}
}
