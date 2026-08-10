package domain

import "errors"

var (
	// ErrSegmentNotFound is returned when a segment does not exist or does
	// not belong to the organization.
	ErrSegmentNotFound = errors.New("segmento no encontrado")

	// ErrCampaignNotFound is returned when a campaign does not exist or does
	// not belong to the organization.
	ErrCampaignNotFound = errors.New("campaña no encontrada")

	// ErrCampaignNotDraft is returned when launching a campaign that is not
	// in draft state (idempotent launch guard).
	ErrCampaignNotDraft = errors.New("la campaña ya fue lanzada")

	// ErrInvalidFilterSpec is returned when a filter spec violates the
	// whitelist (unknown field/op, invalid value).
	ErrInvalidFilterSpec = errors.New("especificación de filtros inválida")

	// ErrAiCreditsExhausted is returned when the organization's AI credits
	// are exhausted before an audience build call.
	ErrAiCreditsExhausted = errors.New("créditos de IA agotados")
)
