package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Prediction struct {
	ID          int64     `json:"id"`
	Jumlah      float64   `json:"jumlah"`
	ClQty       float64   `json:"cl_qty"`
	Cluster     string    `json:"cluster"`
	Rekomendasi string    `json:"rekomendasi"`
	CreatedAt   time.Time `json:"created_at"`
}

func SavePrediction(
	ctx context.Context,
	pool *pgxpool.Pool,
	jumlah float64,
	clQty float64,
	cluster string,
	rekomendasi string,
) error {
	query := `
		INSERT INTO predictions (
			jumlah,
			cl_qty,
			cluster,
			rekomendasi
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := pool.Exec(
		ctx,
		query,
		jumlah,
		clQty,
		cluster,
		rekomendasi,
	)

	return err
}

func GetPredictions(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]Prediction, error) {
	query := `
		SELECT
			id,
			jumlah,
			cl_qty,
			cluster,
			rekomendasi,
			created_at
		FROM predictions
		ORDER BY created_at DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	predictions := make([]Prediction, 0)

	for rows.Next() {
		var p Prediction

		err := rows.Scan(
			&p.ID,
			&p.Jumlah,
			&p.ClQty,
			&p.Cluster,
			&p.Rekomendasi,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		predictions = append(predictions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return predictions, nil
}

func GetPredictionByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id int64,
) (Prediction, error) {
	query := `
		SELECT
			id,
			jumlah,
			cl_qty,
			cluster,
			rekomendasi,
			created_at
		FROM predictions
		WHERE id = $1
	`

	var p Prediction

	err := pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&p.ID,
		&p.Jumlah,
		&p.ClQty,
		&p.Cluster,
		&p.Rekomendasi,
		&p.CreatedAt,
	)

	return p, err
}

func DeletePredictionByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id int64,
) (bool, error) {
	result, err := pool.Exec(
		ctx,
		`DELETE FROM predictions WHERE id = $1`,
		id,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}
