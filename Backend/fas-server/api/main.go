package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"furi-backend/handlers"
	"furi-backend/internal/config"
	"furi-backend/internal/database"
	"furi-backend/middleware"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	envFile := flag.String("env", "", "Lokasi file environment")
	flag.Parse()

	if err := config.Load(*envFile); err != nil {
		return err
	}

	pool, err := database.Open(context.Background())
	if err != nil {
		return err
	}
	defer pool.Close()

	log.Printf(
		"Berhasil terhubung ke PostgreSQL (%s)",
		os.Getenv("PGDATABASE"),
	)

	authHandler := handlers.NewAuthHandler(pool)
	usersHandler := handlers.NewUsersHandler(pool)
	predictionHandler := handlers.NewPredictionHandler(pool)
	productHandler := handlers.NewProductHandler(pool)
	customerHandler := handlers.NewCustomerHandler(pool)
	salesOrderHandler := handlers.NewSalesOrderHandler(pool)
	invoiceHandler := handlers.NewInvoiceHandler(pool)
	warehouseHandler := handlers.NewWarehouseHandler(pool)
	taxHandler := handlers.NewTaxHandler(pool)
	financeSalesHandler := handlers.NewFinanceSalesHandler(pool)
	financeHandler := handlers.NewFinanceHandler(pool)
	paymentHandler := handlers.NewPaymentHandler(pool)
	reportingHandler := handlers.NewFinanceReportingHandler(pool)
	exchangeHandler := handlers.NewFinanceExchangeHandler()
	ledgerHandler := handlers.NewFinanceLedgerHandler(pool)
	companyHandler := handlers.NewCompanyHandler(pool)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "Server Backend PT. FAS berjalan",
		})
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			log.Printf("Pemeriksaan database gagal: %v", err)

			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "error",
				"database": "unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	})

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	// User management: khusus Administrator. Setiap permintaan divalidasi
	// terhadap status, role, dan revisi akun terbaru di database.
	mux.HandleFunc(
		"GET /api/v1/users",
		middleware.AuthWithUsers(pool, middleware.RequireRole(usersHandler.List, "admin")),
	)
	mux.HandleFunc(
		"POST /api/v1/users",
		middleware.AuthWithUsers(pool, middleware.RequireRole(usersHandler.Create, "admin")),
	)
	mux.HandleFunc(
		"PATCH /api/v1/users/{id}",
		middleware.AuthWithUsers(pool, middleware.RequireRole(usersHandler.Update, "admin")),
	)
	mux.HandleFunc(
		"POST /api/v1/users/{id}/password",
		middleware.AuthWithUsers(pool, middleware.RequireRole(usersHandler.ResetPassword, "admin")),
	)
	mux.HandleFunc(
		"GET /api/v1/users/{id}/history",
		middleware.AuthWithUsers(pool, middleware.RequireRole(usersHandler.History, "admin")),
	)

	mux.HandleFunc(
		"GET /api/v1/products",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				productHandler.GetAll,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/products/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				productHandler.GetByID,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/products",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				productHandler.Create,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/products/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				productHandler.Update,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"DELETE /api/v1/products/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				productHandler.Delete,
				"admin",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/customers",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				customerHandler.GetAll,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/customers/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				customerHandler.GetByID,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/customers",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				customerHandler.Create,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/customers/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				customerHandler.Update,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"DELETE /api/v1/customers/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				customerHandler.Delete,
				"admin",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/warehouses",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				warehouseHandler.GetAll,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/warehouses",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				warehouseHandler.Create,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/warehouses/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				warehouseHandler.Update,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"DELETE /api/v1/warehouses/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				warehouseHandler.Deactivate,
				"admin",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/tax-profiles",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				taxHandler.GetAll,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/tax-profiles/current",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				taxHandler.GetCurrent,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/tax-profiles/standard",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				taxHandler.SaveStandard,
				"admin",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/sales-orders",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.GetAll,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/sales-orders/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.GetByID,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/sales-orders",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.CreateDraft,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/sales-orders/{id}/confirm",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.Confirm,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/sales-orders/{id}/picking",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.StartPicking,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/sales-orders/{id}/packed",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.FinishPacking,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/sales-orders/{id}/shipped",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				salesOrderHandler.Ship,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/sales-orders/{id}/invoice",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				invoiceHandler.GetBySalesOrder,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/sales-orders/{id}/invoice",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				invoiceHandler.EnsureBySalesOrder,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"PATCH /api/v1/invoices/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				invoiceHandler.Update,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/invoices/{id}/print",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				invoiceHandler.MarkPrinted,
				"admin",
				"pegawai",
				"finance",
			),
		),
	)

	mux.HandleFunc("GET /api/v1/finance/reporting/periods", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.Periods, "admin", "finance")))
	mux.HandleFunc("PUT /api/v1/finance/reporting/periods/{period}", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.SaveProfit, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/reporting/history", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.History, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/tax/register", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.TaxRegister, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/tax/invoices/{id}", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.TaxSource, "admin", "finance")))
	mux.HandleFunc("PUT /api/v1/finance/tax/invoices/{id}", middleware.AuthWithUsers(pool, middleware.RequireRole(reportingHandler.SaveTax, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/exchange-rates", middleware.AuthWithUsers(pool, middleware.RequireRole(exchangeHandler.Rates, "admin", "finance")))

	mux.HandleFunc("GET /api/v1/company-profile", middleware.AuthWithUsers(pool, middleware.RequireRole(companyHandler.Get, "admin", "finance", "pegawai")))
	mux.HandleFunc("PUT /api/v1/company-profile", middleware.AuthWithUsers(pool, middleware.RequireRole(companyHandler.Update, "admin")))
	mux.HandleFunc("GET /api/v1/company-profile/history", middleware.AuthWithUsers(pool, middleware.RequireRole(companyHandler.History, "admin")))
	mux.HandleFunc("GET /api/v1/finance/expenses", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.Expenses, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/expenses", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.CreateExpense, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/expenses/{id}/void", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.VoidExpense, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/cash", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.Cash, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/notifications", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.Notifications, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/notifications/{id}/read", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.ReadNotification, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/notifications/read-all", middleware.AuthWithUsers(pool, middleware.RequireRole(ledgerHandler.ReadAllNotifications, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payments/connection-test", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.TestConnection, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/payments/config", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Config, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/payments", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.List, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payments", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Create, "admin", "finance")))
	mux.HandleFunc("GET /api/v1/finance/invoices/{id}/payments", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.InvoiceDetails, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payments/{id}/void", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Void, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payment-intents/{id}/sync", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Sync, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payment-intents/{id}/recover", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Recover, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/finance/payment-intents/{id}/cancel", middleware.AuthWithUsers(pool, middleware.RequireRole(paymentHandler.Cancel, "admin", "finance")))
	mux.HandleFunc("POST /api/v1/payments/midtrans/notification", paymentHandler.Notification)

	mux.HandleFunc(
		"GET /api/v1/finance/products/trend",
		middleware.AuthWithUsers(pool, middleware.RequireRole(financeHandler.GetProductTrend, "admin", "finance")),
	)

	mux.HandleFunc(
		"GET /api/v1/finance/receivables",
		middleware.AuthWithUsers(pool, middleware.RequireRole(financeHandler.GetReceivables, "admin", "finance")),
	)

	mux.HandleFunc(
		"GET /api/v1/finance/sales",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				financeSalesHandler.GetSales,
				"admin",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/finance/sales/summary",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				financeSalesHandler.GetSummary,
				"admin",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/finance/sales/trend",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				financeSalesHandler.GetTrend,
				"admin",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/finance/invoices/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				financeSalesHandler.GetInvoiceByID,
				"admin",
				"finance",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/predictions",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				predictionHandler.GetAll,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"GET /api/v1/predictions/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				predictionHandler.GetByID,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/predictions",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				predictionHandler.Create,
				"admin",
				"pegawai",
			),
		),
	)

	mux.HandleFunc(
		"DELETE /api/v1/predictions/{id}",
		middleware.AuthWithUsers(pool,
			middleware.RequireRole(
				predictionHandler.Delete,
				"admin",
			),
		),
	)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server Backend PT. FAS jalan di http://localhost:8080")
	log.Println("Authentication + RBAC + Product + Customer + Sales Order Lifecycle + Prediction + PostgreSQL siap")

	log.Println("POST   /api/v1/auth/login")
	log.Println("GET    /api/v1/users")
	log.Println("POST   /api/v1/users")
	log.Println("PATCH  /api/v1/users/{id}")
	log.Println("POST   /api/v1/users/{id}/password")
	log.Println("GET    /api/v1/users/{id}/history")

	log.Println("GET    /api/v1/products")
	log.Println("GET    /api/v1/products/{id}")
	log.Println("POST   /api/v1/products")
	log.Println("PATCH  /api/v1/products/{id}")
	log.Println("DELETE /api/v1/products/{id}")

	log.Println("GET    /api/v1/customers")
	log.Println("GET    /api/v1/customers/{id}")
	log.Println("POST   /api/v1/customers")
	log.Println("PATCH  /api/v1/customers/{id}")
	log.Println("DELETE /api/v1/customers/{id}")

	log.Println("GET    /api/v1/warehouses")
	log.Println("POST   /api/v1/warehouses")
	log.Println("PATCH  /api/v1/warehouses/{id}")
	log.Println("DELETE /api/v1/warehouses/{id}")
	log.Println("GET    /api/v1/tax-profiles")
	log.Println("GET    /api/v1/tax-profiles/current")
	log.Println("POST   /api/v1/tax-profiles/standard")

	log.Println("GET    /api/v1/sales-orders")
	log.Println("GET    /api/v1/sales-orders/{id}")
	log.Println("POST   /api/v1/sales-orders")
	log.Println("PATCH  /api/v1/sales-orders/{id}/confirm")
	log.Println("PATCH  /api/v1/sales-orders/{id}/picking")
	log.Println("PATCH  /api/v1/sales-orders/{id}/packed")
	log.Println("PATCH  /api/v1/sales-orders/{id}/shipped")
	log.Println("GET    /api/v1/sales-orders/{id}/invoice")
	log.Println("POST   /api/v1/sales-orders/{id}/invoice")
	log.Println("PATCH  /api/v1/invoices/{id}")
	log.Println("POST   /api/v1/invoices/{id}/print")
	log.Println("GET    /api/v1/finance/sales")
	log.Println("GET    /api/v1/finance/sales/summary")
	log.Println("GET    /api/v1/finance/sales/trend")
	log.Println("GET    /api/v1/finance/invoices/{id}")

	log.Println("POST   /api/v1/predictions")
	log.Println("GET    /api/v1/predictions")
	log.Println("GET    /api/v1/predictions/{id}")
	log.Println("DELETE /api/v1/predictions/{id}")

	return server.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Gagal menulis respons JSON: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, PATCH, DELETE, OPTIONS",
		)
		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
