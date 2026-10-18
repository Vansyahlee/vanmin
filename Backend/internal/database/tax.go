package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaxProfile struct {
	ID                  uuid.UUID `json:"id"`
	ProfileCode         string    `json:"profile_code"`
	Name                string    `json:"name"`
	NominalRate         float64   `json:"nominal_rate"`
	DPPFactor           float64   `json:"dpp_factor"`
	EffectiveRate       float64   `json:"effective_rate"`
	EffectiveYear       int       `json:"effective_year"`
	EffectiveFrom       time.Time `json:"effective_from"`
	RegulationReference string    `json:"regulation_reference"`
	Notes               string    `json:"notes"`
	IsActive            bool      `json:"is_active"`
	IsDefault           bool      `json:"is_default"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SaveTaxProfileInput struct {
	Name                string
	NominalRate         float64
	DPPFactor           float64
	RegulationReference string
	Notes               string
}

func GetTaxProfiles(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]TaxProfile, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			profile_code,
			name,
			nominal_rate,
			dpp_factor,
			effective_rate,
			effective_year,
			effective_from,
			COALESCE(regulation_reference, ''),
			COALESCE(notes, ''),
			is_active,
			is_default,
			created_at,
			updated_at
		FROM tax_profiles
		ORDER BY
			is_default DESC,
			effective_year DESC,
			effective_from DESC,
			created_at DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make(
		[]TaxProfile,
		0,
	)

	for rows.Next() {
		var profile TaxProfile

		if err := rows.Scan(
			&profile.ID,
			&profile.ProfileCode,
			&profile.Name,
			&profile.NominalRate,
			&profile.DPPFactor,
			&profile.EffectiveRate,
			&profile.EffectiveYear,
			&profile.EffectiveFrom,
			&profile.RegulationReference,
			&profile.Notes,
			&profile.IsActive,
			&profile.IsDefault,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		); err != nil {
			return nil, err
		}

		profiles = append(
			profiles,
			profile,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func GetDefaultTaxProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*TaxProfile, error) {
	var profile TaxProfile

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			profile_code,
			name,
			nominal_rate,
			dpp_factor,
			effective_rate,
			effective_year,
			effective_from,
			COALESCE(regulation_reference, ''),
			COALESCE(notes, ''),
			is_active,
			is_default,
			created_at,
			updated_at
		FROM tax_profiles
		WHERE is_default = TRUE
		  AND is_active = TRUE
		ORDER BY effective_from DESC, created_at DESC
		LIMIT 1
		`,
	).Scan(
		&profile.ID,
		&profile.ProfileCode,
		&profile.Name,
		&profile.NominalRate,
		&profile.DPPFactor,
		&profile.EffectiveRate,
		&profile.EffectiveYear,
		&profile.EffectiveFrom,
		&profile.RegulationReference,
		&profile.Notes,
		&profile.IsActive,
		&profile.IsDefault,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func GetTaxProfileByID(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
) (*TaxProfile, error) {
	var profile TaxProfile

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			profile_code,
			name,
			nominal_rate,
			dpp_factor,
			effective_rate,
			effective_year,
			effective_from,
			COALESCE(regulation_reference, ''),
			COALESCE(notes, ''),
			is_active,
			is_default,
			created_at,
			updated_at
		FROM tax_profiles
		WHERE id = $1
		  AND is_active = TRUE
		`,
		id,
	).Scan(
		&profile.ID,
		&profile.ProfileCode,
		&profile.Name,
		&profile.NominalRate,
		&profile.DPPFactor,
		&profile.EffectiveRate,
		&profile.EffectiveYear,
		&profile.EffectiveFrom,
		&profile.RegulationReference,
		&profile.Notes,
		&profile.IsActive,
		&profile.IsDefault,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func SaveStandardTaxProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
	input SaveTaxProfileInput,
) (*TaxProfile, error) {
	now := time.Now()
	year := now.Year()

	effectiveRate :=
		math.Round(
			input.NominalRate*
				input.DPPFactor*
				10000,
		) / 10000

	tx, err := pool.BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`
		UPDATE tax_profiles
		SET
			is_default = FALSE,
			is_active = FALSE,
			updated_at = NOW()
		WHERE is_default = TRUE
		`,
	)
	if err != nil {
		return nil, err
	}

	profileCode := fmt.Sprintf(
		"PPN-STANDARD-%d-%s",
		year,
		now.Format("20060102-150405"),
	)

	var id uuid.UUID

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO tax_profiles (
			profile_code,
			name,
			nominal_rate,
			dpp_factor,
			effective_rate,
			effective_year,
			effective_from,
			regulation_reference,
			notes,
			is_active,
			is_default,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			CURRENT_DATE,
			NULLIF($7, ''),
			NULLIF($8, ''),
			TRUE,
			TRUE,
			NOW(),
			NOW()
		)
		RETURNING id
		`,
		profileCode,
		input.Name,
		input.NominalRate,
		input.DPPFactor,
		effectiveRate,
		year,
		input.RegulationReference,
		input.Notes,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return GetTaxProfileByID(
		ctx,
		pool,
		id,
	)
}
