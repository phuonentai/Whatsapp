package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// scheduleService implements the handler-facing schedule CRUD façade with
// next-run computation in the org timezone and append-only audits.
type scheduleService struct {
	repo    domain.ScheduleRepository
	settings domain.FollowUpSettingsRepository
	catalog domain.CatalogReader
	tz      domain.OrgTimezoneReader
	kill    domain.KillSwitchReader
	audit   domain.AuditLogWriter
	clock   domain.Clock
	log     logger.Logger
}

// NewScheduleService builds the schedule service.
func NewScheduleService(
	repo domain.ScheduleRepository,
	settings domain.FollowUpSettingsRepository,
	catalog domain.CatalogReader,
	tz domain.OrgTimezoneReader,
	kill domain.KillSwitchReader,
	audit domain.AuditLogWriter,
	clock domain.Clock,
	log logger.Logger,
) InquiryscheduleService {
	return &scheduleService{repo: repo, settings: settings, catalog: catalog, tz: tz, kill: kill, audit: audit, clock: clock, log: log}
}

func (s *scheduleService) orgLocation(ctx context.Context, orgID int32) (*time.Location, error) {
	name, err := s.tz.Timezone(ctx, orgID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc = time.UTC
	}
	return loc, nil
}

func (s *scheduleService) CreateSchedule(ctx context.Context, orgID int32, memberID string, in CreateScheduleInput) (*domain.Schedule, error) {
	sch := &domain.Schedule{
		OrganizationID: orgID,
		Name:           in.Name,
		RunTime:        in.RunTime,
		DaysOfWeek:     in.DaysOfWeek,
		ProductIDs:     in.ProductIDs,
		SupplierIDs:    in.SupplierIDs,
		Note:           in.Note,
		IsActive:       true,
	}
	if err := sch.Validate(); err != nil {
		return nil, err
	}
	if err := sch.ValidateOrgScope(ctx, s.catalog); err != nil {
		return nil, err
	}
	loc, err := s.orgLocation(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := sch.NextOccurrenceAfter(s.clock.Now(), loc)
	if err != nil {
		return nil, err
	}
	sch.NextRunAt = next

	created, err := s.repo.Create(ctx, orgID, sch)
	if err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: orgID,
		EntityType:     "schedule",
		EntityID:       &created.ID,
		Action:         "schedule_created",
		MemberID:       strPtrOrNil(memberID),
		Metadata: map[string]any{
			"name":         created.Name,
			"run_time":     created.RunTime,
			"days_of_week": daysToStrings(created.DaysOfWeek),
			"next_run_at":  created.NextRunAt,
		},
	}); err != nil {
		s.log.Error("audit schedule_created failed", map[string]any{"error": err.Error()})
	}
	return created, nil
}

func (s *scheduleService) UpdateSchedule(ctx context.Context, orgID int32, memberID string, in UpdateScheduleInput) (*domain.Schedule, error) {
	existing, err := s.repo.Get(ctx, orgID, in.ID)
	if err != nil {
		return nil, err
	}
	existing.Name = in.Name
	existing.RunTime = in.RunTime
	existing.DaysOfWeek = in.DaysOfWeek
	existing.ProductIDs = in.ProductIDs
	existing.SupplierIDs = in.SupplierIDs
	existing.Note = in.Note
	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := existing.ValidateOrgScope(ctx, s.catalog); err != nil {
		return nil, err
	}
	loc, err := s.orgLocation(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := existing.NextOccurrenceAfter(s.clock.Now(), loc)
	if err != nil {
		return nil, err
	}
	existing.NextRunAt = next

	updated, err := s.repo.Update(ctx, orgID, existing)
	if err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: orgID,
		EntityType:     "schedule",
		EntityID:       &updated.ID,
		Action:         "schedule_updated",
		MemberID:       strPtrOrNil(memberID),
		Metadata:       map[string]any{"next_run_at": updated.NextRunAt},
	}); err != nil {
		s.log.Error("audit schedule_updated failed", map[string]any{"error": err.Error()})
	}
	return updated, nil
}

func (s *scheduleService) PauseSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error) {
	paused, err := s.repo.Pause(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: orgID,
		EntityType:     "schedule",
		EntityID:       &paused.ID,
		Action:         "schedule_paused",
		MemberID:       strPtrOrNil(memberID),
	}); err != nil {
		s.log.Error("audit schedule_paused failed", map[string]any{"error": err.Error()})
	}
	return paused, nil
}

func (s *scheduleService) ResumeSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error) {
	existing, err := s.repo.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	loc, err := s.orgLocation(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := existing.NextOccurrenceAfter(s.clock.Now(), loc)
	if err != nil {
		return nil, err
	}
	resumed, err := s.repo.Resume(ctx, orgID, id, next)
	if err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: orgID,
		EntityType:     "schedule",
		EntityID:       &resumed.ID,
		Action:         "schedule_resumed",
		MemberID:       strPtrOrNil(memberID),
		Metadata:       map[string]any{"next_run_at": resumed.NextRunAt},
	}); err != nil {
		s.log.Error("audit schedule_resumed failed", map[string]any{"error": err.Error()})
	}
	return resumed, nil
}

func (s *scheduleService) DeleteSchedule(ctx context.Context, orgID, id int32, memberID string) error {
	if err := s.repo.Delete(ctx, orgID, id); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: orgID,
		EntityType:     "schedule",
		EntityID:       &id,
		Action:         "schedule_deleted",
		MemberID:       strPtrOrNil(memberID),
	}); err != nil {
		s.log.Error("audit schedule_deleted failed", map[string]any{"error": err.Error()})
	}
	return nil
}

func (s *scheduleService) ListSchedules(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error) {
	return s.repo.ListWithStatus(ctx, orgID)
}

func (s *scheduleService) GetScheduleDetail(ctx context.Context, orgID, id int32) (*ScheduleDetail, error) {
	sch, err := s.repo.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings.GetByOrg(ctx, orgID)
	if errors.Is(err, domain.ErrFollowUpSettingsNotFound) {
		def := domain.DefaultFollowUpSettings(orgID)
		settings = &def
		err = nil
	}
	if err != nil {
		return nil, err
	}
	runs, err := s.repo.RecentRuns(ctx, orgID, id, 20)
	if err != nil {
		return nil, err
	}
	overdue, err := s.repo.CountOverdueRecipients(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	return &ScheduleDetail{Schedule: sch, FollowUp: settings, RecentRuns: runs, OverdueRecipients: overdue}, nil
}

func (s *scheduleService) GetFollowUpSettings(ctx context.Context, orgID int32) (*domain.FollowUpSettings, error) {
	settings, err := s.settings.GetByOrg(ctx, orgID)
	if errors.Is(err, domain.ErrFollowUpSettingsNotFound) {
		def := domain.DefaultFollowUpSettings(orgID)
		return &def, nil
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *scheduleService) UpdateFollowUpSettings(ctx context.Context, orgID int32, in UpdateFollowUpSettingsInput) (*domain.FollowUpSettings, error) {
	settings := &domain.FollowUpSettings{
		OrganizationID:  orgID,
		Enabled:         in.Enabled,
		DeadlineHours:   in.DeadlineHours,
		MaxNudges:       in.MaxNudges,
		MessageTemplate: in.MessageTemplate,
	}
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	return s.settings.Upsert(ctx, settings)
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func daysToStrings(days domain.DaysOfWeek) []string {
	out := make([]string, 0, len(days))
	for _, d := range days {
		out = append(out, fmt.Sprintf("%d", int(d)))
	}
	return out
}
