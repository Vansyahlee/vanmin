package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	AuthRevision int64     `json:"-"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

func FindUserByEmail(
	ctx context.Context,
	pool *pgxpool.Pool,
	email string,
) (User, error) {
	query := `
		SELECT
			id::text,
			name,
			email,
			password_hash,
			role,
			is_active,
			created_at,
            auth_revision
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	var user User

	err := pool.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.AuthRevision,
	)

	return user, err
}

func UpdateUserPassword(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
	passwordHash string,
) error {
	query := `
		UPDATE users
		SET password_hash = $1, auth_revision = auth_revision + 1
		WHERE id = $2::uuid
	`

	_, err := pool.Exec(
		ctx,
		query,
		passwordHash,
		userID,
	)

	return err
}
