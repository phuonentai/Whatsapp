package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger"
)

// fakeTrialSeeder records calls for assertion.
type fakeTrialSeeder struct {
	calls []fakeTrialSeederCall
}

type fakeTrialSeederCall struct {
	organizationID int32
	trialEnd       time.Time
}

func (f *fakeTrialSeeder) SeedTrial(ctx context.Context, organizationID int32, trialEnd time.Time) error {
	f.calls = append(f.calls, fakeTrialSeederCall{organizationID: organizationID, trialEnd: trialEnd})
	return nil
}

// stubLogger is a no-op logger for unit tests.
type stubLogger struct{}

func (stubLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (stubLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (stubLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (stubLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (stubLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (stubLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return stubLogger{}
}

// assert compiler checks
var _ loggerDomain.Logger = stubLogger{}

func TestSeedTrial_DisabledByDefault(t *testing.T) {
	t.Setenv("TRIAL_ENABLED", "false")
	seeder := &fakeTrialSeeder{}
	svc := &memberService{trialSeeder: seeder, logger: stubLogger{}}

	err := svc.seedTrial(context.Background(), 42)
	require.NoError(t, err)
	assert.Empty(t, seeder.calls, "no seed call when TRIAL_ENABLED=false")
}

func TestSeedTrial_EnabledCreatesCall(t *testing.T) {
	t.Setenv("TRIAL_ENABLED", "true")
	t.Setenv("TRIAL_DAYS", "14")
	seeder := &fakeTrialSeeder{}
	svc := &memberService{trialSeeder: seeder, logger: stubLogger{}}

	err := svc.seedTrial(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, seeder.calls, 1)
	assert.Equal(t, int32(42), seeder.calls[0].organizationID)

	expectedEnd := time.Now().Add(14 * 24 * time.Hour)
	diff := seeder.calls[0].trialEnd.Sub(expectedEnd).Abs()
	assert.Less(t, diff, 2*time.Second, "trial end should be ~14 days from now")
}

func TestSeedTrial_CustomDays(t *testing.T) {
	t.Setenv("TRIAL_ENABLED", "true")
	t.Setenv("TRIAL_DAYS", "30")
	seeder := &fakeTrialSeeder{}
	svc := &memberService{trialSeeder: seeder, logger: stubLogger{}}

	err := svc.seedTrial(context.Background(), 99)
	require.NoError(t, err)
	require.Len(t, seeder.calls, 1)

	expectedEnd := time.Now().Add(30 * 24 * time.Hour)
	diff := seeder.calls[0].trialEnd.Sub(expectedEnd).Abs()
	assert.Less(t, diff, 2*time.Second, "trial end should be ~30 days from now")
}

func TestSeedTrial_NilSeederNoOp(t *testing.T) {
	t.Setenv("TRIAL_ENABLED", "true")
	svc := &memberService{trialSeeder: nil, logger: stubLogger{}}

	err := svc.seedTrial(context.Background(), 42)
	require.NoError(t, err) // nil seeder is a silent no-op
}

func TestSeedTrial_EmptyTrialDaysUsesDefault(t *testing.T) {
	t.Setenv("TRIAL_ENABLED", "true")
	t.Setenv("TRIAL_DAYS", "") // empty → default 14
	seeder := &fakeTrialSeeder{}
	svc := &memberService{trialSeeder: seeder, logger: stubLogger{}}

	err := svc.seedTrial(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, seeder.calls, 1)

	expectedEnd := time.Now().Add(14 * 24 * time.Hour)
	diff := seeder.calls[0].trialEnd.Sub(expectedEnd).Abs()
	assert.Less(t, diff, 2*time.Second)
}

func TestNewMemberService_HasTrialSeeder(t *testing.T) {
	seeder := &fakeTrialSeeder{}
	svc := NewMemberService(
		nil, nil, nil, nil, nil,
		domain.TrialSeeder(seeder),
		stubLogger{},
	)
	assert.NotNil(t, svc)
}
