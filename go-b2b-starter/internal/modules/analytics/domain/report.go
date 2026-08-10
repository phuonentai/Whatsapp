package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidRange    = errors.New("rango de fechas inválido")
	ErrInvalidDays     = errors.New("parámetro days inválido")
	ErrInvalidPeriod   = errors.New("periodo inválido")
	ErrInvalidLimit    = errors.New("parámetro limit inválido")
	ErrAnalyticsFailed = errors.New("error al generar el reporte")
)

// RevenuePoint is one time-bucketed revenue aggregate (COP, invoiced).
type RevenuePoint struct {
	Periodo    string
	MontoTotal float64
}

// TopCustomer is a customer aggregated by invoiced revenue (COP).
type TopCustomer struct {
	Nombre     string
	MontoTotal float64
}

// FunnelEntry is one pipeline funnel bucket (stage, estado or otras_pipelines).
type FunnelEntry struct {
	Etapa      string
	Cantidad   int32
	MontoTotal float64
}

// FunnelReport is the assembled funnel: stages + closed-deal aggregates.
type FunnelReport struct {
	Etapas         []FunnelEntry
	OtrasPipelines *FunnelEntry
	Ganado         *FunnelEntry
	Perdido        *FunnelEntry
	Abandonado     *FunnelEntry
}

// InactiveContact is a contact without recent WhatsApp activity.
type InactiveContact struct {
	Telefono        string
	Nombre          string
	UltimoMensajeAt *time.Time
	Clasificacion   string
}

// AnalyticsRepository provides org-scoped sales aggregations.
type AnalyticsRepository interface {
	RevenueByPeriod(ctx context.Context, orgID int32, from, to time.Time, period string) ([]RevenuePoint, error)
	TopCustomersByRevenue(ctx context.Context, orgID int32, limit int32) ([]TopCustomer, error)
	FunnelByStage(ctx context.Context, orgID int32) ([]FunnelEntry, error)
	DealStateCounts(ctx context.Context, orgID int32) ([]FunnelEntry, error)
	InactiveContacts(ctx context.Context, orgID int32, since time.Time) ([]InactiveContact, error)
	DefaultPipelineStages(ctx context.Context, orgID int32) ([]string, error)
}
