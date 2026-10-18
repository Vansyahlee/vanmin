package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFinanceInvalidDateRange = errors.New("rentang tanggal finance tidak valid")
	ErrFinanceInvoiceNotFound  = errors.New("faktur finance tidak ditemukan")
)

type FinanceSalesService struct {
	pool *pgxpool.Pool
}

type FinanceSalesFilter struct {
	DateFrom   time.Time
	DateTo     time.Time
	Query      string
	Status     string
	PrintState string
	Limit      int
	Offset     int
}

func NewFinanceSalesService(
	pool *pgxpool.Pool,
) *FinanceSalesService {
	return &FinanceSalesService{
		pool: pool,
	}
}

func (s *FinanceSalesService) List(
	ctx context.Context,
	filter FinanceSalesFilter,
) ([]database.FinanceSale, int64, error) {
	if filter.DateFrom.After(filter.DateTo) {
		return nil, 0, ErrFinanceInvalidDateRange
	}

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return database.ListFinanceSales(
		ctx,
		s.pool,
		database.FinanceSalesFilter{
			DateFrom:   filter.DateFrom,
			DateTo:     filter.DateTo,
			Query:      strings.TrimSpace(filter.Query),
			Status:     strings.ToUpper(strings.TrimSpace(filter.Status)),
			PrintState: strings.ToLower(strings.TrimSpace(filter.PrintState)),
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	)
}

func (s *FinanceSalesService) Summary(
	ctx context.Context,
	filter FinanceSalesFilter,
) (*database.FinanceSalesSummary, error) {
	if filter.DateFrom.After(filter.DateTo) {
		return nil, ErrFinanceInvalidDateRange
	}

	return database.GetFinanceSalesSummary(
		ctx,
		s.pool,
		database.FinanceSalesFilter{
			DateFrom:   filter.DateFrom,
			DateTo:     filter.DateTo,
			Query:      strings.TrimSpace(filter.Query),
			Status:     strings.ToUpper(strings.TrimSpace(filter.Status)),
			PrintState: strings.ToLower(strings.TrimSpace(filter.PrintState)),
		},
	)
}

func (s *FinanceSalesService) Trend(
	ctx context.Context,
	filter FinanceSalesFilter,
) (*database.FinanceSalesTrend, error) {
	if filter.DateFrom.After(filter.DateTo) {
		return nil, ErrFinanceInvalidDateRange
	}

	return database.GetFinanceSalesTrend(
		ctx,
		s.pool,
		database.FinanceSalesFilter{
			DateFrom:   filter.DateFrom,
			DateTo:     filter.DateTo,
			Query:      strings.TrimSpace(filter.Query),
			Status:     strings.ToUpper(strings.TrimSpace(filter.Status)),
			PrintState: strings.ToLower(strings.TrimSpace(filter.PrintState)),
		},
	)
}

func (s *FinanceSalesService) GetInvoiceByID(
	ctx context.Context,
	id uuid.UUID,
) (*database.Invoice, error) {
	invoice, err := database.GetInvoiceByID(
		ctx,
		s.pool,
		id,
	)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrFinanceInvoiceNotFound
	}

	return invoice, nil
}
