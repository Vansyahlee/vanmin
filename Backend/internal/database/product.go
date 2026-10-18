package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	WarehouseID    *uuid.UUID `json:"warehouse_id"`
	ID             uuid.UUID  `json:"id"`
	ProductCode    string     `json:"product_code"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Classification string     `json:"classification"`
	Unit           string     `json:"unit"`
	PurchasePrice  float64    `json:"purchase_price"`
	Price          float64    `json:"price"`
	Stock          int        `json:"stock"`
	MinStock       int        `json:"min_stock"`
	Barcode        *string    `json:"barcode"`
	ImageURL       *string    `json:"image_url"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateProductInput struct {
	WarehouseID    *uuid.UUID `json:"warehouse_id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Classification string     `json:"classification"`
	Unit           string     `json:"unit"`
	PurchasePrice  float64    `json:"purchase_price"`
	Price          float64    `json:"price"`
	Stock          int        `json:"stock"`
	MinStock       int        `json:"min_stock"`
	Barcode        *string    `json:"barcode"`
}

type UpdateProductInput struct {
	WarehouseID    *uuid.UUID `json:"warehouse_id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Classification string     `json:"classification"`
	Unit           string     `json:"unit"`
	PurchasePrice  float64    `json:"purchase_price"`
	Price          float64    `json:"price"`
	Stock          int        `json:"stock"`
	MinStock       int        `json:"min_stock"`
	Barcode        *string    `json:"barcode"`
	IsActive       bool       `json:"is_active"`
}

func CreateProduct(
	ctx context.Context,
	pool *pgxpool.Pool,
	input CreateProductInput,
) (*Product, error) {
	var product Product

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO products (
			sku,
			name,
			classification,
			unit,
			purchase_price,
			price,
			stock,
			min_stock,
			barcode,
			warehouse_id
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10
		)
		RETURNING
			id,
			product_code,
			sku,
			name,
			classification,
			unit,
			purchase_price,
			price,
			stock,
			min_stock,
			barcode,
			image_url,
			is_active,
			created_at,
			updated_at,
			warehouse_id
		`,
		input.SKU,
		input.Name,
		input.Classification,
		input.Unit,
		input.PurchasePrice,
		input.Price,
		input.Stock,
		input.MinStock,
		input.Barcode,
		input.WarehouseID,
	).Scan(
		&product.ID,
		&product.ProductCode,
		&product.SKU,
		&product.Name,
		&product.Classification,
		&product.Unit,
		&product.PurchasePrice,
		&product.Price,
		&product.Stock,
		&product.MinStock,
		&product.Barcode,
		&product.ImageURL,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.WarehouseID,
	)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func GetProducts(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]Product, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			product_code,
			sku,
			name,
			classification,
			unit,
			purchase_price,
			price,
			stock,
			min_stock,
			barcode,
			image_url,
			is_active,
			created_at,
			updated_at,
			warehouse_id
		FROM products
		ORDER BY product_code ASC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)

	for rows.Next() {
		var product Product

		if err := rows.Scan(
			&product.ID,
			&product.ProductCode,
			&product.SKU,
			&product.Name,
			&product.Classification,
			&product.Unit,
			&product.PurchasePrice,
			&product.Price,
			&product.Stock,
			&product.MinStock,
			&product.Barcode,
			&product.ImageURL,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.WarehouseID,
		); err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func GetProductByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*Product, error) {
	var product Product

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			product_code,
			sku,
			name,
			classification,
			unit,
			purchase_price,
			price,
			stock,
			min_stock,
			barcode,
			image_url,
			is_active,
			created_at,
			updated_at,
			warehouse_id
		FROM products
		WHERE id = $1
		`,
		id,
	).Scan(
		&product.ID,
		&product.ProductCode,
		&product.SKU,
		&product.Name,
		&product.Classification,
		&product.Unit,
		&product.PurchasePrice,
		&product.Price,
		&product.Stock,
		&product.MinStock,
		&product.Barcode,
		&product.ImageURL,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.WarehouseID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func UpdateProductByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	input UpdateProductInput,
) (*Product, error) {
	var product Product

	err := pool.QueryRow(
		ctx,
		`
		UPDATE products
		SET
			sku = $2,
			name = $3,
			classification = $4,
			unit = $5,
			purchase_price = $6,
			price = $7,
			stock = $8,
			min_stock = $9,
			barcode = $10,
			is_active = $11,
			warehouse_id = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			product_code,
			sku,
			name,
			classification,
			unit,
			purchase_price,
			price,
			stock,
			min_stock,
			barcode,
			image_url,
			is_active,
			created_at,
			updated_at,
			warehouse_id
		`,
		id,
		input.SKU,
		input.Name,
		input.Classification,
		input.Unit,
		input.PurchasePrice,
		input.Price,
		input.Stock,
		input.MinStock,
		input.Barcode,
		input.IsActive,
		input.WarehouseID,
	).Scan(
		&product.ID,
		&product.ProductCode,
		&product.SKU,
		&product.Name,
		&product.Classification,
		&product.Unit,
		&product.PurchasePrice,
		&product.Price,
		&product.Stock,
		&product.MinStock,
		&product.Barcode,
		&product.ImageURL,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.WarehouseID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func DeleteProductByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (bool, error) {
	result, err := pool.Exec(
		ctx,
		`
		DELETE FROM products
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}
