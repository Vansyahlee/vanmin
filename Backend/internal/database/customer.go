package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Customer struct {
	ID               uuid.UUID `json:"id"`
	CustomerCode     string    `json:"customer_code"`
	Name             string    `json:"name"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	ShippingAddress  string    `json:"shipping_address"`
	TaxIDType        string    `json:"tax_id_type"`
	TaxIDNumber      string    `json:"tax_id_number"`
	TaxName          string    `json:"tax_name"`
	TaxAddress       string    `json:"tax_address"`
	IsPKP            bool      `json:"is_pkp"`
	TKUID            string    `json:"tku_id"`
	PaymentTermsDays int       `json:"payment_terms_days"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateCustomerInput struct {
	Name             string `json:"name"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	ShippingAddress  string `json:"shipping_address"`
	TaxIDType        string `json:"tax_id_type"`
	TaxIDNumber      string `json:"tax_id_number"`
	TaxName          string `json:"tax_name"`
	TaxAddress       string `json:"tax_address"`
	IsPKP            bool   `json:"is_pkp"`
	TKUID            string `json:"tku_id"`
	PaymentTermsDays int    `json:"payment_terms_days"`
}

type UpdateCustomerInput struct {
	Name             string
	Phone            string
	Email            string
	ShippingAddress  string
	TaxIDType        string
	TaxIDNumber      string
	TaxName          string
	TaxAddress       string
	IsPKP            bool
	TKUID            string
	PaymentTermsDays int
	IsActive         bool
}

func CreateCustomer(
	ctx context.Context,
	pool *pgxpool.Pool,
	input CreateCustomerInput,
) (*Customer, error) {
	var customer Customer

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO customers (
			name,
			phone,
			email,
			shipping_address,
			tax_id_type,
			tax_id_number,
			tax_name,
			tax_address,
			is_pkp,
			tku_id,
			payment_terms_days
		)
		VALUES (
			$1,
			NULLIF($2, ''),
			NULLIF($3, ''),
			NULLIF($4, ''),
			NULLIF($5, ''),
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			$9,
			NULLIF($10, ''),
			$11
		)
		RETURNING
			id,
			customer_code,
			name,
			COALESCE(phone, ''),
			COALESCE(email, ''),
			COALESCE(shipping_address, ''),
			COALESCE(tax_id_type, ''),
			COALESCE(tax_id_number, ''),
			COALESCE(tax_name, ''),
			COALESCE(tax_address, ''),
			is_pkp,
			COALESCE(tku_id, ''),
			payment_terms_days,
			is_active,
			created_at,
			updated_at
		`,
		input.Name,
		input.Phone,
		input.Email,
		input.ShippingAddress,
		input.TaxIDType,
		input.TaxIDNumber,
		input.TaxName,
		input.TaxAddress,
		input.IsPKP,
		input.TKUID,
		input.PaymentTermsDays,
	).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.ShippingAddress,
		&customer.TaxIDType,
		&customer.TaxIDNumber,
		&customer.TaxName,
		&customer.TaxAddress,
		&customer.IsPKP,
		&customer.TKUID,
		&customer.PaymentTermsDays,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func GetCustomers(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]Customer, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			customer_code,
			name,
			COALESCE(phone, ''),
			COALESCE(email, ''),
			COALESCE(shipping_address, ''),
			COALESCE(tax_id_type, ''),
			COALESCE(tax_id_number, ''),
			COALESCE(tax_name, ''),
			COALESCE(tax_address, ''),
			is_pkp,
			COALESCE(tku_id, ''),
			payment_terms_days,
			is_active,
			created_at,
			updated_at
		FROM customers
		ORDER BY name ASC, customer_code ASC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	customers := make([]Customer, 0)

	for rows.Next() {
		var customer Customer

		if err := rows.Scan(
			&customer.ID,
			&customer.CustomerCode,
			&customer.Name,
			&customer.Phone,
			&customer.Email,
			&customer.ShippingAddress,
			&customer.TaxIDType,
			&customer.TaxIDNumber,
			&customer.TaxName,
			&customer.TaxAddress,
			&customer.IsPKP,
			&customer.TKUID,
			&customer.PaymentTermsDays,
			&customer.IsActive,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		); err != nil {
			return nil, err
		}

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return customers, nil
}

func GetCustomerByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Customer, error) {
	var customer Customer

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			customer_code,
			name,
			COALESCE(phone, ''),
			COALESCE(email, ''),
			COALESCE(shipping_address, ''),
			COALESCE(tax_id_type, ''),
			COALESCE(tax_id_number, ''),
			COALESCE(tax_name, ''),
			COALESCE(tax_address, ''),
			is_pkp,
			COALESCE(tku_id, ''),
			payment_terms_days,
			is_active,
			created_at,
			updated_at
		FROM customers
		WHERE id = $1
		`,
		id,
	).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.ShippingAddress,
		&customer.TaxIDType,
		&customer.TaxIDNumber,
		&customer.TaxName,
		&customer.TaxAddress,
		&customer.IsPKP,
		&customer.TKUID,
		&customer.PaymentTermsDays,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func UpdateCustomerByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	input UpdateCustomerInput,
) (*Customer, error) {
	var customer Customer

	err := pool.QueryRow(
		ctx,
		`
		UPDATE customers
		SET
			name = $2,
			phone = NULLIF($3, ''),
			email = NULLIF($4, ''),
			shipping_address = NULLIF($5, ''),
			tax_id_type = NULLIF($6, ''),
			tax_id_number = NULLIF($7, ''),
			tax_name = NULLIF($8, ''),
			tax_address = NULLIF($9, ''),
			is_pkp = $10,
			tku_id = NULLIF($11, ''),
			payment_terms_days = $12,
			is_active = $13,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			customer_code,
			name,
			COALESCE(phone, ''),
			COALESCE(email, ''),
			COALESCE(shipping_address, ''),
			COALESCE(tax_id_type, ''),
			COALESCE(tax_id_number, ''),
			COALESCE(tax_name, ''),
			COALESCE(tax_address, ''),
			is_pkp,
			COALESCE(tku_id, ''),
			payment_terms_days,
			is_active,
			created_at,
			updated_at
		`,
		id,
		input.Name,
		input.Phone,
		input.Email,
		input.ShippingAddress,
		input.TaxIDType,
		input.TaxIDNumber,
		input.TaxName,
		input.TaxAddress,
		input.IsPKP,
		input.TKUID,
		input.PaymentTermsDays,
		input.IsActive,
	).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.ShippingAddress,
		&customer.TaxIDType,
		&customer.TaxIDNumber,
		&customer.TaxName,
		&customer.TaxAddress,
		&customer.IsPKP,
		&customer.TKUID,
		&customer.PaymentTermsDays,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func DeleteCustomerByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (bool, error) {
	result, err := pool.Exec(
		ctx,
		`
		DELETE FROM customers
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}
