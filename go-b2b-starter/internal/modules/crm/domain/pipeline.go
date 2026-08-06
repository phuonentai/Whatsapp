package domain

import "time"

type Pipeline struct {
	ID               int32     `json:"id"`
	OrganizationID   int32     `json:"organization_id"`
	Nombre           string    `json:"nombre"`
	EsPredeterminado bool      `json:"es_predeterminado"`
	Orden            int32     `json:"orden"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PipelineWithStages struct {
	Pipeline
	Etapas []PipelineStage `json:"etapas"`
}

type PipelineStage struct {
	ID           int32      `json:"id"`
	PipelineID   int32      `json:"pipeline_id"`
	Nombre       string     `json:"nombre"`
	Orden        int32      `json:"orden"`
	Color        string     `json:"color,omitempty"`
	Probabilidad *int32     `json:"probabilidad,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
