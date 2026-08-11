package domain

import (
	"context"
	"time"
)

type ContactRepository interface {
	UpsertByPhone(ctx context.Context, contact *Contact) (*Contact, error)
	UpsertByIGUser(ctx context.Context, contact *Contact) (*Contact, error)
	GetByID(ctx context.Context, orgID, contactID int32) (*Contact, error)
	GetByPhone(ctx context.Context, orgID int32, phoneNumber string) (*Contact, error)
	GetByIGUser(ctx context.Context, orgID int32, igUserID string) (*Contact, error)
	UpdateInstagramProfile(ctx context.Context, orgID, contactID int32, username, avatarURL, displayName string) (*Contact, error)
	List(ctx context.Context, orgID int32, limit, offset int32) ([]*Contact, error)
	ListFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*Contact, error)
	Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*Contact, error)
	CountFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo int32) (int32, error)
	CountSearch(ctx context.Context, orgID int32, query string) (int32, error)
	Update(ctx context.Context, contact *Contact) (*Contact, error)
	Delete(ctx context.Context, orgID, contactID int32) error
}

type ConversationRepository interface {
	GetByID(ctx context.Context, orgID, convID int32) (*Conversation, error)
	GetActiveByContact(ctx context.Context, orgID, contactID int32) (*Conversation, error)
	GetActiveByContactChannel(ctx context.Context, orgID, contactID int32, channel string) (*Conversation, error)
	Create(ctx context.Context, conv *Conversation) (*Conversation, error)
	EnsureActive(ctx context.Context, conv *Conversation) (*Conversation, error)
	UpdateLastMessageAt(ctx context.Context, orgID, convID int32, lastMessageAt *time.Time) (*Conversation, error)
	UpdateStatus(ctx context.Context, orgID, convID int32, status ConversationStatus) (*Conversation, error)
	ListByOrganization(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string) ([]*ConversationWithContact, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	InsertIdempotent(ctx context.Context, msg *Message) (*Message, bool, error)
	GetByProviderID(ctx context.Context, orgID int32, channel, providerMessageID string) (*Message, error)
	UpdateStatus(ctx context.Context, id int32, status, providerMessageID string) (*Message, error)
	ListByConversation(ctx context.Context, orgID, convID int32, limit, offset int32) ([]*Message, error)
}

type CompanyRepository interface {
	Create(ctx context.Context, company *Company) (*Company, error)
	GetByID(ctx context.Context, orgID, companyID int32) (*CompanyWithCounts, error)
	GetByNit(ctx context.Context, orgID int32, nit string) (*Company, error)
	List(ctx context.Context, orgID int32, limit, offset int32) ([]*CompanyWithCounts, error)
	Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*CompanyWithCounts, error)
	CountList(ctx context.Context, orgID int32) (int32, error)
	CountSearch(ctx context.Context, orgID int32, query string) (int32, error)
	Update(ctx context.Context, company *Company) (*Company, error)
	Delete(ctx context.Context, orgID, companyID int32) error
}

type DealRepository interface {
	Create(ctx context.Context, deal *Deal) (*Deal, error)
	GetByID(ctx context.Context, orgID, dealID int32) (*DealWithRefs, error)
	List(ctx context.Context, orgID int32, pipelineID, stageID int32, status string, contactID, limit, offset int32) ([]*DealWithRefs, error)
	Update(ctx context.Context, deal *Deal) (*Deal, error)
	UpdateStage(ctx context.Context, orgID, dealID, stageID int32) (*Deal, error)
	Delete(ctx context.Context, orgID, dealID int32) error
}

type PipelineRepository interface {
	Create(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	GetByID(ctx context.Context, orgID, pipelineID int32) (*Pipeline, error)
	List(ctx context.Context, orgID int32) ([]*PipelineWithStages, error)
	GetDefault(ctx context.Context, orgID int32) (*Pipeline, error)
	Update(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	Delete(ctx context.Context, orgID, pipelineID int32) error
}

type PipelineStageRepository interface {
	Create(ctx context.Context, stage *PipelineStage) (*PipelineStage, error)
	ListByPipeline(ctx context.Context, pipelineID int32) ([]*PipelineStage, error)
	GetByID(ctx context.Context, stageID int32) (*PipelineStage, error)
	Update(ctx context.Context, stage *PipelineStage) (*PipelineStage, error)
	Delete(ctx context.Context, stageID, pipelineID int32) error
}

type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) (*Activity, error)
	GetByID(ctx context.Context, orgID, activityID int32) (*Activity, error)
	ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) ([]*ActivityWithActor, error)
	ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*ActivityWithActor, error)
	ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*ActivityWithActor, error)
	ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*ActivityWithActor, error)
	CountByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID int32) (int32, error)
	CountByContact(ctx context.Context, contactID, orgID int32) (int32, error)
	CountByDeal(ctx context.Context, dealID, orgID int32) (int32, error)
	CountByCompany(ctx context.Context, companyID, orgID int32) (int32, error)
}

type TagRepository interface {
	Create(ctx context.Context, tag *Tag) (*Tag, error)
	GetByID(ctx context.Context, orgID, tagID int32) (*Tag, error)
	List(ctx context.Context, orgID int32) ([]*Tag, error)
	Update(ctx context.Context, tag *Tag) (*Tag, error)
	Delete(ctx context.Context, orgID, tagID int32) error
}

type EntityTagRepository interface {
	Attach(ctx context.Context, tagID int32, entityType EntityType, entityID int32) (*EntityTag, error)
	Detach(ctx context.Context, tagID int32, entityType EntityType, entityID int32) error
	ListByEntity(ctx context.Context, entityType EntityType, entityID int32) ([]*Tag, error)
	ListByTag(ctx context.Context, tagID int32) ([]*EntityTag, error)
}
