package postgres

import (
	"time"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// systemClock is the production domain.Clock.
type systemClock struct{}

// Now returns wall-clock time.
func (systemClock) Now() time.Time { return time.Now() }

// NewSystemClock builds the production clock.
func NewSystemClock() domain.Clock { return systemClock{} }

// Repositories bundles every inquiry-scheduling infra adapter over the SQLC
// store so the DI container can share one instance across interface bindings.
type Repositories struct {
	Schedules        domain.ScheduleRepository
	FollowUpSettings domain.FollowUpSettingsRepository
	Catalog          domain.CatalogReader
	KillSwitch       domain.KillSwitchReader
	Timezone         domain.OrgTimezoneReader
	Recipients       domain.RecipientStateReader
	Nudges           domain.NudgeIncrementer
	Enqueuer         domain.FollowUpEnqueuer
	Orgs             domain.FollowUpEnabledOrgLister
	Audit            domain.AuditLogWriter
	Creator          domain.InquiryRunCreator
}

// NewRepositories constructs all repository adapters over one store.
func NewRepositories(store sqlc.Store, clock domain.Clock, draft domain.DraftFunc, log logger.Logger) *Repositories {
	sched := NewScheduleRepository(store, clock)
	readers := NewRecipientReader(store, clock)
	governance := NewGovernanceReader(store)
	return &Repositories{
		Schedules:        sched,
		FollowUpSettings: sched,
		Catalog:          NewCatalogReader(store),
		KillSwitch:       governance,
		Timezone:         governance,
		Recipients:       readers,
		Nudges:           readers,
		Enqueuer:         readers,
		Orgs:             readers,
		Audit:            NewAuditWriter(store),
		Creator:          NewRunCreator(store, draft, log),
	}
}
