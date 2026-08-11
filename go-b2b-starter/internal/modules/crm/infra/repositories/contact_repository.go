package repositories

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type contactRepository struct {
	store sqlc.Store
}

func NewContactRepository(store sqlc.Store) domain.ContactRepository {
	return &contactRepository{store: store}
}

func (r *contactRepository) UpsertByPhone(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	params := sqlc.UpsertContactParams{
		OrganizationID: contact.OrganizationID,
		PhoneNumber:    helpers.ToPgText(contact.PhoneNumber),
		DisplayName:    helpers.ToPgText(contact.DisplayName),
		AvatarUrl:      helpers.ToPgText(contact.AvatarURL),
		Metadata:       helpers.ToJSONB(contact.Metadata),
		LastMessageAt:  helpers.ToPgTimestampPtr(contact.LastMessageAt),
	}
	result, err := r.store.UpsertContact(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert contact: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) GetByID(ctx context.Context, orgID, contactID int32) (*domain.Contact, error) {
	params := sqlc.GetContactByIDParams{ID: contactID, OrganizationID: orgID}
	result, err := r.store.GetContactByID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) GetByPhone(ctx context.Context, orgID int32, phoneNumber string) (*domain.Contact, error) {
	params := sqlc.GetContactByPhoneParams{OrganizationID: orgID, PhoneNumber: helpers.ToPgText(phoneNumber)}
	result, err := r.store.GetContactByPhone(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact by phone: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) UpsertByIGUser(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	params := sqlc.UpsertContactByIGUserParams{
		OrganizationID:    contact.OrganizationID,
		InstagramUserID:   helpers.ToPgText(contact.InstagramUserID),
		InstagramUsername: helpers.ToPgText(contact.InstagramUsername),
		DisplayName:       helpers.ToPgText(contact.DisplayName),
		AvatarUrl:         helpers.ToPgText(contact.AvatarURL),
		Metadata:          helpers.ToJSONB(contact.Metadata),
		LastMessageAt:     helpers.ToPgTimestampPtr(contact.LastMessageAt),
	}
	result, err := r.store.UpsertContactByIGUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert instagram contact: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) GetByIGUser(ctx context.Context, orgID int32, igUserID string) (*domain.Contact, error) {
	params := sqlc.GetContactByIGUserParams{OrganizationID: orgID, InstagramUserID: helpers.ToPgText(igUserID)}
	result, err := r.store.GetContactByIGUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact by IG user: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) UpdateInstagramProfile(ctx context.Context, orgID, contactID int32, username, avatarURL, displayName string) (*domain.Contact, error) {
	result, err := r.store.UpdateContactInstagramProfile(ctx, sqlc.UpdateContactInstagramProfileParams{
		ID:                contactID,
		OrganizationID:    orgID,
		InstagramUsername: helpers.ToPgText(username),
		AvatarUrl:         helpers.ToPgText(avatarURL),
		DisplayName:       helpers.ToPgText(displayName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update instagram profile: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Contact, error) {
	params := sqlc.ListContactsByOrganizationParams{OrganizationID: orgID, Limit: limit, Offset: offset}
	results, err := r.store.ListContactsByOrganization(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	contacts := make([]*domain.Contact, len(results))
	for i := range results {
		contacts[i] = r.mapToDomain(&results[i])
	}
	return contacts, nil
}

func (r *contactRepository) ListFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*domain.Contact, error) {
	results, err := r.store.ListContactsByOrganizationFiltered(ctx, sqlc.ListContactsByOrganizationFilteredParams{
		OrganizationID: orgID,
		Column2:        interface{}(source),
		Column3:        interface{}(leadStatus),
		Column4:        companyID,
		Column5:        assignedTo,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list filtered contacts: %w", err)
	}
	contacts := make([]*domain.Contact, len(results))
	for i := range results {
		contacts[i] = r.mapToDomain(&results[i])
	}
	return contacts, nil
}

func (r *contactRepository) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.Contact, error) {
	results, err := r.store.SearchContacts(ctx, sqlc.SearchContactsParams{
		OrganizationID: orgID,
		Column2:        helpers.ToPgText(query),
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}
	contacts := make([]*domain.Contact, len(results))
	for i := range results {
		contacts[i] = r.mapToDomain(&results[i])
	}
	return contacts, nil
}

func (r *contactRepository) CountFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo int32) (int32, error) {
	count, err := r.store.CountContactsByOrganizationFiltered(ctx, sqlc.CountContactsByOrganizationFilteredParams{
		OrganizationID: orgID,
		Column2:        interface{}(source),
		Column3:        interface{}(leadStatus),
		Column4:        companyID,
		Column5:        assignedTo,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count filtered contacts: %w", err)
	}
	return int32(count), nil
}

func (r *contactRepository) CountSearch(ctx context.Context, orgID int32, query string) (int32, error) {
	count, err := r.store.CountSearchContacts(ctx, sqlc.CountSearchContactsParams{
		OrganizationID: orgID,
		Column2:        helpers.ToPgText(query),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count search contacts: %w", err)
	}
	return int32(count), nil
}

func (r *contactRepository) Update(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	result, err := r.store.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contact.ID,
		OrganizationID: contact.OrganizationID,
		Column3:        helpers.ToPgText(contact.Email),
		CompanyID:      helpers.ToPgInt4Ptr(contact.CompanyID),
		Column5:        helpers.ToPgText(contact.DisplayName),
		Column6:        helpers.ToPgText(string(contact.Source)),
		Column7:        helpers.ToPgText(string(contact.LeadStatus)),
		Column8:        helpers.ToPgText(contact.JobTitle),
		AssignedTo:     helpers.ToPgInt4Ptr(contact.AssignedTo),
		Column10:       helpers.ToPgText(string(contact.TipoDocumento)),
		Column11:       helpers.ToPgText(contact.NumeroDocumento),
		Column12:       helpers.ToPgText(contact.AvatarURL),
		Column13:       helpers.ToJSONB(contact.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *contactRepository) Delete(ctx context.Context, orgID, contactID int32) error {
	return r.store.DeleteContact(ctx, sqlc.DeleteContactParams{ID: contactID, OrganizationID: orgID})
}

func (r *contactRepository) mapToDomain(c *sqlc.CrmContact) *domain.Contact {
	return &domain.Contact{
		ID:                c.ID,
		OrganizationID:    c.OrganizationID,
		PhoneNumber:       helpers.FromPgText(c.PhoneNumber),
		InstagramUserID:   helpers.FromPgText(c.InstagramUserID),
		InstagramUsername: helpers.FromPgText(c.InstagramUsername),
		DisplayName:       helpers.FromPgText(c.DisplayName),
		Email:             helpers.FromPgText(c.Email),
		CompanyID:         helpers.FromPgInt4Ptr(c.CompanyID),
		Source:            domain.ContactSource(c.Source),
		LeadStatus:        domain.LeadStatus(c.LeadStatus),
		JobTitle:          helpers.FromPgText(c.JobTitle),
		AssignedTo:        helpers.FromPgInt4Ptr(c.AssignedTo),
		TipoDocumento:     domain.TipoDocumento(helpers.FromPgText(c.TipoDocumento)),
		NumeroDocumento:   helpers.FromPgText(c.NumeroDocumento),
		AvatarURL:         helpers.FromPgText(c.AvatarUrl),
		Metadata:          helpers.FromJSONB(c.Metadata),
		IsBlocked:         c.IsBlocked,
		ConsentStatus:     c.ConsentStatus,
		LastMessageAt:     helpers.FromPgTimestampPtr(c.LastMessageAt),
		CreatedAt:         c.CreatedAt.Time,
		UpdatedAt:         c.UpdatedAt.Time,
	}
}
