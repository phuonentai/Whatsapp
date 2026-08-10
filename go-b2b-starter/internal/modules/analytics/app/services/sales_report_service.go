package services

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
)

const (
	maxReportSpan       = 13 * 30 * 24 * time.Hour
	defaultWindow       = 30 * 24 * time.Hour
	defaultLimit        = 10
	maxLimit            = 50
	defaultInactiveDays = 30
	minInactiveDays     = 1
	maxInactiveDays     = 365
)

// SalesReportService validates report parameters and orchestrates the
// org-scoped aggregation queries.
type SalesReportService struct {
	repo domain.AnalyticsRepository
}

func NewSalesReportService(repo domain.AnalyticsRepository) *SalesReportService {
	return &SalesReportService{repo: repo}
}

// RevenueByPeriod returns invoiced revenue bucketed by week or month.
func (s *SalesReportService) RevenueByPeriod(ctx context.Context, orgID int32, period string, from, to *time.Time) ([]domain.RevenuePoint, error) {
	if period != "week" && period != "month" {
		return nil, domain.ErrInvalidPeriod
	}
	fromTime, toTime, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.RevenueByPeriod(ctx, orgID, fromTime, toTime, period)
}

// TopCustomers returns the top N customers by invoiced revenue.
func (s *SalesReportService) TopCustomers(ctx context.Context, orgID int32, limit int32) ([]domain.TopCustomer, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return s.repo.TopCustomersByRevenue(ctx, orgID, limit)
}

// Funnel returns the assembled pipeline funnel for the org.
func (s *SalesReportService) Funnel(ctx context.Context, orgID int32) (*domain.FunnelReport, error) {
	aggregates, err := s.repo.FunnelByStage(ctx, orgID)
	if err != nil {
		return nil, err
	}
	stateCounts, err := s.repo.DealStateCounts(ctx, orgID)
	if err != nil {
		return nil, err
	}
	stages, err := s.repo.DefaultPipelineStages(ctx, orgID)
	if err != nil {
		return nil, err
	}

	report := &domain.FunnelReport{
		Etapas: make([]domain.FunnelEntry, 0, len(stages)),
	}
	byStage := make(map[string]*domain.FunnelEntry, len(aggregates))
	for i := range aggregates {
		entry := aggregates[i]
		byStage[entry.Etapa] = &entry
	}

	// Zero-fill every stage of the default pipeline.
	for _, stageName := range stages {
		entry, ok := byStage[stageName]
		if !ok {
			report.Etapas = append(report.Etapas, domain.FunnelEntry{Etapa: stageName})
			continue
		}
		report.Etapas = append(report.Etapas, *entry)
	}

	if other, ok := byStage["otras_pipelines"]; ok {
		report.OtrasPipelines = other
	}
	for i := range stateCounts {
		entry := stateCounts[i]
		switch entry.Etapa {
		case "ganado":
			report.Ganado = &entry
		case "perdido":
			report.Perdido = &entry
		case "abandonado":
			report.Abandonado = &entry
		}
	}
	return report, nil
}

// InactiveContacts returns contacts without WhatsApp activity for `days` days.
func (s *SalesReportService) InactiveContacts(ctx context.Context, orgID int32, days int32) ([]domain.InactiveContact, error) {
	if days < minInactiveDays || days > maxInactiveDays {
		return nil, domain.ErrInvalidDays
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	return s.repo.InactiveContacts(ctx, orgID, since)
}

// resolveRange validates and defaults the reporting window (default: last 30 days).
func resolveRange(from, to *time.Time) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	fromTime := now.Add(-defaultWindow)
	toTime := now
	if from != nil {
		fromTime = from.UTC()
	}
	if to != nil {
		toTime = to.UTC()
	}
	if fromTime.After(toTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: fecha inicial posterior a la final", domain.ErrInvalidRange)
	}
	if toTime.Sub(fromTime) > maxReportSpan {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: el rango no puede superar 13 meses", domain.ErrInvalidRange)
	}
	return fromTime, toTime, nil
}
