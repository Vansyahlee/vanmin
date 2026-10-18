package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"furi-backend/services"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TaxHandler struct {
	service *services.TaxService
}

type saveStandardTaxRequest struct {
	Name                string  `json:"name"`
	NominalRate         float64 `json:"nominal_rate"`
	DPPFactorPercent    float64 `json:"dpp_factor_percent"`
	RegulationReference string  `json:"regulation_reference"`
	Notes               string  `json:"notes"`
}

func NewTaxHandler(
	pool *pgxpool.Pool,
) *TaxHandler {
	return &TaxHandler{
		service: services.NewTaxService(
			pool,
		),
	}
}

func (h *TaxHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	profiles, err :=
		h.service.GetAll(
			r.Context(),
		)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Aturan pajak tidak dapat dimuat. Silakan coba kembali.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		profiles,
	)
}

func (h *TaxHandler) GetCurrent(
	w http.ResponseWriter,
	r *http.Request,
) {
	profile, err :=
		h.service.GetCurrent(
			r.Context(),
		)
	if err != nil {
		if errors.Is(
			err,
			services.ErrTaxProfileNotFound,
		) {
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "Aturan PPN aktif belum tersedia.",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Aturan PPN aktif tidak dapat dimuat.",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		profile,
	)
}

func (h *TaxHandler) SaveStandard(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request saveStandardTaxRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "Data aturan pajak tidak valid.",
			},
		)
		return
	}

	profile, err :=
		h.service.SaveStandard(
			r.Context(),
			services.SaveStandardTaxInput{
				Name:                request.Name,
				NominalRate:         request.NominalRate,
				DPPFactorPercent:    request.DPPFactorPercent,
				RegulationReference: request.RegulationReference,
				Notes:               request.Notes,
			},
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrTaxProfileNameInvalid,
		),
			errors.Is(
				err,
				services.ErrTaxNominalRateInvalid,
			),
			errors.Is(
				err,
				services.ErrTaxDPPFactorInvalid,
			):
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)

		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "Aturan PPN tidak dapat disimpan. Silakan coba kembali.",
				},
			)
		}
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		profile,
	)
}
