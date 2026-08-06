package domain

import (
	"errors"
	"time"
)

type DealStatus string

const (
	DealStatusAbierto    DealStatus = "abierto"
	DealStatusGanado     DealStatus = "ganado"
	DealStatusPerdido    DealStatus = "perdido"
	DealStatusAbandonado DealStatus = "abandonado"
)

type Deal struct {
	ID                  int32                  `json:"id"`
	OrganizationID      int32                  `json:"organization_id"`
	Nombre              string                 `json:"nombre"`
	ContactID           *int32                 `json:"contact_id,omitempty"`
	CompanyID           *int32                 `json:"company_id,omitempty"`
	PipelineID          int32                  `json:"pipeline_id"`
	StageID             *int32                 `json:"stage_id,omitempty"`
	Monto               *float64               `json:"monto,omitempty"`
	Moneda              string                 `json:"moneda"`
	FechaCierreEsperada *time.Time             `json:"fecha_cierre_esperada,omitempty"`
	Estado              DealStatus             `json:"estado"`
	Probabilidad        *int32                 `json:"probabilidad,omitempty"`
	Notas               string                 `json:"notas,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	AssignedTo          *int32                 `json:"assigned_to,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

func (d *Deal) Validate() error {
	if d.Nombre == "" {
		return ErrDealNameRequired
	}
	if d.PipelineID == 0 {
		return errors.New("el pipeline es requerido")
	}
	return nil
}

func (d *Deal) CanTransitionTo(target DealStatus) bool {
	if d.Estado == DealStatusGanado || d.Estado == DealStatusPerdido || d.Estado == DealStatusAbandonado {
		return false
	}
	switch target {
	case DealStatusAbierto, DealStatusGanado, DealStatusPerdido, DealStatusAbandonado:
		return true
	}
	return false
}

type DealWithRefs struct {
	Deal
	ContactName  string `json:"contact_name,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	CompanyName  string `json:"company_name,omitempty"`
}
