package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Warehouse struct {
	ID             uuid.UUID `json:"id"`
	WarehouseCode  string    `json:"warehouse_code"`
	Name           string    `json:"name"`
	Classification string    `json:"classification"`
	Address        string    `json:"address"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateWarehouseInput struct {
	Name           string
	Classification string
	Address        string
}

type UpdateWarehouseInput struct {
	Name           string
	Classification string
	Address        string
	IsActive       bool
}

func GetWarehouses(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]Warehouse, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			warehouse_code,
			name,
			classification,
			COALESCE(address, ''),
			is_active,
			created_at,
			updated_at
		FROM warehouses
		ORDER BY
			is_active DESC,
			CASE
				WHEN warehouse_code = 'GUDANG_1' THEN 1
				WHEN warehouse_code = 'GUDANG_2' THEN 2
				WHEN warehouse_code = 'GUDANG_3' THEN 3
				ELSE 4
			END,
			name ASC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	warehouses := make(
		[]Warehouse,
		0,
	)

	for rows.Next() {
		var warehouse Warehouse

		if err := rows.Scan(
			&warehouse.ID,
			&warehouse.WarehouseCode,
			&warehouse.Name,
			&warehouse.Classification,
			&warehouse.Address,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		); err != nil {
			return nil, err
		}

		warehouses = append(
			warehouses,
			warehouse,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return warehouses, nil
}

func GetWarehouseByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Warehouse, error) {
	var warehouse Warehouse

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			warehouse_code,
			name,
			classification,
			COALESCE(address, ''),
			is_active,
			created_at,
			updated_at
		FROM warehouses
		WHERE id = $1
		`,
		id,
	).Scan(
		&warehouse.ID,
		&warehouse.WarehouseCode,
		&warehouse.Name,
		&warehouse.Classification,
		&warehouse.Address,
		&warehouse.IsActive,
		&warehouse.CreatedAt,
		&warehouse.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &warehouse, nil
}

func CreateWarehouse(
	ctx context.Context,
	pool *pgxpool.Pool,
	input CreateWarehouseInput,
) (*Warehouse, error) {
	var id uuid.UUID

	err := pool.QueryRow(
		ctx,
		`
		WITH next_code AS (
			SELECT
				'WH-' ||
				LPAD(
					nextval(
						'warehouses_warehouse_code_seq'
					)::text,
					6,
					'0'
				) AS value
		)
		INSERT INTO warehouses (
			code,
			warehouse_code,
			name,
			classification,
			address,
			is_active,
			created_at,
			updated_at
		)
		SELECT
			value,
			value,
			$1,
			$2,
			NULLIF($3, ''),
			TRUE,
			NOW(),
			NOW()
		FROM next_code
		RETURNING id
		`,
		strings.TrimSpace(input.Name),
		input.Classification,
		strings.TrimSpace(input.Address),
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return GetWarehouseByID(
		ctx,
		pool,
		id,
	)
}

func UpdateWarehouse(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	input UpdateWarehouseInput,
) (*Warehouse, error) {
	result, err := pool.Exec(
		ctx,
		`
		UPDATE warehouses
		SET
			name = $2,
			classification = $3,
			address = NULLIF($4, ''),
			is_active = $5,
			updated_at = NOW()
		WHERE id = $1
		`,
		id,
		strings.TrimSpace(input.Name),
		input.Classification,
		strings.TrimSpace(input.Address),
		input.IsActive,
	)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected() != 1 {
		return nil, nil
	}

	return GetWarehouseByID(
		ctx,
		pool,
		id,
	)
}

func DeactivateWarehouse(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Warehouse, error) {
	result, err := pool.Exec(
		ctx,
		`
		UPDATE warehouses
		SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected() != 1 {
		return nil, nil
	}

	return GetWarehouseByID(
		ctx,
		pool,
		id,
	)
}
