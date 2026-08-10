package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
)

type stubAnalyticsRepo struct {
	revenue  []domain.RevenuePoint
	top      []domain.TopCustomer
	funnel   []domain.FunnelEntry
	states   []domain.FunnelEntry
	inactive []domain.InactiveContact
	stages   []string
}

func (s *stubAnalyticsRepo) RevenueByPeriod(ctx context.Context, orgID int32, from, to time.Time, period string) ([]domain.RevenuePoint, error) {
	if s.revenue == nil {
		return []domain.RevenuePoint{}, nil
	}
	return s.revenue, nil
}
func (s *stubAnalyticsRepo) TopCustomersByRevenue(ctx context.Context, orgID int32, limit int32) ([]domain.TopCustomer, error) {
	return s.top, nil
}
func (s *stubAnalyticsRepo) FunnelByStage(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return s.funnel, nil
}
func (s *stubAnalyticsRepo) DealStateCounts(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return s.states, nil
}
func (s *stubAnalyticsRepo) InactiveContacts(ctx context.Context, orgID int32, since time.Time) ([]domain.InactiveContact, error) {
	if s.inactive == nil {
		return []domain.InactiveContact{}, nil
	}
	return s.inactive, nil
}
func (s *stubAnalyticsRepo) DefaultPipelineStages(ctx context.Context, orgID int32) ([]string, error) {
	return s.stages, nil
}

func TestRevenueByPeriod_ValidatesPeriod(t *testing.T) {
	svc := NewSalesReportService(&stubAnalyticsRepo{})
	_, err := svc.RevenueByPeriod(context.Background(), 1, "year", nil, nil)
	require.ErrorIs(t, err, domain.ErrInvalidPeriod)

	points, err := svc.RevenueByPeriod(context.Background(), 1, "month", nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, points)
}

func TestRevenueByPeriod_ValidatesRange(t *testing.T) {
	svc := NewSalesReportService(&stubAnalyticsRepo{})
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.RevenueByPeriod(context.Background(), 1, "week", &from, &to)
	require.ErrorIs(t, err, domain.ErrInvalidRange)

	longAgo := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.RevenueByPeriod(context.Background(), 1, "week", &longAgo, &to)
	require.ErrorIs(t, err, domain.ErrInvalidRange)
}

func TestTopCustomers_ClampsLimit(t *testing.T) {
	repo := &stubAnalyticsRepo{top: []domain.TopCustomer{{Nombre: "Cliente A", MontoTotal: 1000}}}
	svc := NewSalesReportService(repo)

	customers, err := svc.TopCustomers(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Len(t, customers, 1)

	_, err = svc.TopCustomers(context.Background(), 1, 999)
	require.NoError(t, err)
}

func TestInactiveContacts_ValidatesDays(t *testing.T) {
	svc := NewSalesReportService(&stubAnalyticsRepo{})
	_, err := svc.InactiveContacts(context.Background(), 1, 0)
	require.ErrorIs(t, err, domain.ErrInvalidDays)
	_, err = svc.InactiveContacts(context.Background(), 1, 366)
	require.ErrorIs(t, err, domain.ErrInvalidDays)

	contacts, err := svc.InactiveContacts(context.Background(), 1, 30)
	require.NoError(t, err)
	assert.NotNil(t, contacts)
}

func TestFunnel_ZeroFillsDefaultStagesAndGroupsOthers(t *testing.T) {
	repo := &stubAnalyticsRepo{
		funnel: []domain.FunnelEntry{
			{Etapa: "Prospección", Cantidad: 2, MontoTotal: 5000},
		},
		states: []domain.FunnelEntry{
			{Etapa: "ganado", Cantidad: 1, MontoTotal: 10000},
			{Etapa: "perdido", Cantidad: 3, MontoTotal: 0},
		},
		stages: []string{"Prospección", "Calificado"},
	}
	svc := NewSalesReportService(repo)

	report, err := svc.Funnel(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, report.Etapas, 2)
	assert.Equal(t, "Prospección", report.Etapas[0].Etapa)
	assert.Equal(t, int32(2), report.Etapas[0].Cantidad)
	assert.Equal(t, "Calificado", report.Etapas[1].Etapa)
	assert.Equal(t, int32(0), report.Etapas[1].Cantidad)
	require.NotNil(t, report.Ganado)
	assert.Equal(t, int32(1), report.Ganado.Cantidad)
	require.NotNil(t, report.Perdido)
	assert.Nil(t, report.Abandonado)
}

func TestFunnel_IncludesOtrasPipelines(t *testing.T) {
	repo := &stubAnalyticsRepo{
		funnel: []domain.FunnelEntry{
			{Etapa: "otras_pipelines", Cantidad: 4, MontoTotal: 2000},
		},
		stages: []string{},
	}
	svc := NewSalesReportService(repo)

	report, err := svc.Funnel(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, report.OtrasPipelines)
	assert.Equal(t, int32(4), report.OtrasPipelines.Cantidad)
}

func TestRevenueByPeriod_PropagatesRepoErrors(t *testing.T) {
	svc := NewSalesReportService(&errRepo{})
	_, err := svc.RevenueByPeriod(context.Background(), 1, "month", nil, nil)
	require.Error(t, err)
}

type errRepo struct{}

func (s *errRepo) RevenueByPeriod(ctx context.Context, orgID int32, from, to time.Time, period string) ([]domain.RevenuePoint, error) {
	return nil, errors.New("boom")
}
func (s *errRepo) TopCustomersByRevenue(ctx context.Context, orgID int32, limit int32) ([]domain.TopCustomer, error) {
	return nil, errors.New("boom")
}
func (s *errRepo) FunnelByStage(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return nil, errors.New("boom")
}
func (s *errRepo) DealStateCounts(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return nil, errors.New("boom")
}
func (s *errRepo) InactiveContacts(ctx context.Context, orgID int32, since time.Time) ([]domain.InactiveContact, error) {
	return nil, errors.New("boom")
}
func (s *errRepo) DefaultPipelineStages(ctx context.Context, orgID int32) ([]string, error) {
	return nil, errors.New("boom")
}
