// Package services implements the inquiry-scheduling application layer:
// schedule CRUD with next-run computation, the scheduled-run outbox handler,
// follow-up automation (deadline, one nudge, escalation), and the schedule /
// follow-up status surfaces. All state transitions go through
// transaction-isolated repository guards so outbox redelivery is idempotent.
package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
)

// CreateScheduleInput is the POST /api/procurement/schedules payload.
type CreateScheduleInput struct {
	Name        string
	RunTime     string // "HH:MM" in the org timezone
	DaysOfWeek  []domain.DayOfWeek
	ProductIDs  []int32
	SupplierIDs []int32
	Note        string
}

// UpdateScheduleInput carries the editable schedule fields (full replace).
type UpdateScheduleInput struct {
	ID          int32
	Name        string
	RunTime     string
	DaysOfWeek  []domain.DayOfWeek
	ProductIDs  []int32
	SupplierIDs []int32
	Note        string
}

// UpdateFollowUpSettingsInput is the PUT /api/procurement/followup-settings
// payload.
type UpdateFollowUpSettingsInput struct {
	Enabled         bool
	DeadlineHours   int
	MaxNudges       int
	MessageTemplate string
}

// ScheduleDetail is GET /api/procurement/schedules/:id — schedule, joined
// products/suppliers, follow-up settings, and recent runs.
type ScheduleDetail struct {
	Schedule          *domain.Schedule
	FollowUp          *domain.FollowUpSettings
	RecentRuns        []*domain.ScheduledRun
	OverdueRecipients int32
}

// InquiryscheduleService is the handler-facing façade for the module.
type InquiryscheduleService interface {
	CreateSchedule(ctx context.Context, orgID int32, memberID string, in CreateScheduleInput) (*domain.Schedule, error)
	UpdateSchedule(ctx context.Context, orgID int32, memberID string, in UpdateScheduleInput) (*domain.Schedule, error)
	PauseSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error)
	ResumeSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error)
	DeleteSchedule(ctx context.Context, orgID, id int32, memberID string) error
	ListSchedules(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error)
	GetScheduleDetail(ctx context.Context, orgID, id int32) (*ScheduleDetail, error)

	GetFollowUpSettings(ctx context.Context, orgID int32) (*domain.FollowUpSettings, error)
	UpdateFollowUpSettings(ctx context.Context, orgID int32, in UpdateFollowUpSettingsInput) (*domain.FollowUpSettings, error)
}

// ScheduledRunHandler processes the durable inquiry_run.scheduled event.
type ScheduledRunHandler interface {
	HandleScheduledRun(ctx context.Context, e *inqEvents.InquiryRunScheduled) error
}

// FollowUpSendHandler processes the durable inquiry.followup_send event.
type FollowUpSendHandler interface {
	HandleFollowUpSend(ctx context.Context, e *inqEvents.FollowupSend) error
}
