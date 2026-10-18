package services

import (
	"context"
	"errors"
	"strings"

	"furi-backend/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTaxProfileNotFound    = errors.New("aturan pajak tidak ditemukan")
	ErrTaxNominalRateInvalid = errors.New("tarif PPN tidak valid")
	ErrTaxDPPFactorInvalid   = errors.New("faktor DPP tidak valid")
	ErrTaxProfileNameInvalid = errors.New("nama aturan pajak tidak valid")
)

type TaxService struct {
	pool *pgxpool.Pool
}

type SaveStandardTaxInput struct {
	Name                string
	NominalRate         float64
	DPPFactorPercent    float64
	RegulationReference string
	Notes               string
}

func NewTaxService(
	pool *pgxpool.Pool,
) *TaxService {
	return &TaxService{
		pool: pool,
	}
}

func (s *TaxService) GetAll(
	ctx context.Context,
) ([]database.TaxProfile, error) {
	return database.GetTaxProfiles(
		ctx,
		s.pool,
	)
}

func (s *TaxService) GetCurrent(
	ctx context.Context,
) (*database.TaxProfile, error) {
	profile, err :=
		database.GetDefaultTaxProfile(
			ctx,
			s.pool,
		)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrTaxProfileNotFound
	}

	return profile, nil
}

func (s *TaxService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*database.TaxProfile, error) {
	profile, err :=
		database.GetTaxProfileByID(
			ctx,
			s.pool,
			id,
		)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrTaxProfileNotFound
	}

	return profile, nil
}

func (s *TaxService) SaveStandard(
	ctx context.Context,
	input SaveStandardTaxInput,
) (*database.TaxProfile, error) {
	name :=
		strings.TrimSpace(
			input.Name,
		)
	if len(name) < 3 {
		return nil, ErrTaxProfileNameInvalid
	}

	if input.NominalRate < 0 ||
		input.NominalRate > 100 {
		return nil, ErrTaxNominalRateInvalid
	}

	if input.DPPFactorPercent < 0 ||
		input.DPPFactorPercent > 200 {
		return nil, ErrTaxDPPFactorInvalid
	}

	return database.SaveStandardTaxProfile(
		ctx,
		s.pool,
		database.SaveTaxProfileInput{
			Name:        name,
			NominalRate: input.NominalRate,
			DPPFactor:   input.DPPFactorPercent / 100,
			RegulationReference: strings.TrimSpace(
				input.RegulationReference,
			),
			Notes: strings.TrimSpace(
				input.Notes,
			),
		},
	)
}
