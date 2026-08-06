package domain

import "time"

type Tag struct {
	ID             int32     `json:"id"`
	OrganizationID int32     `json:"organization_id"`
	Nombre         string    `json:"nombre"`
	Color          string    `json:"color,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type EntityType string

const (
	EntityTypeContact EntityType = "contact"
	EntityTypeCompany EntityType = "company"
	EntityTypeDeal    EntityType = "deal"
)

type EntityTag struct {
	ID         int32      `json:"id"`
	TagID      int32      `json:"tag_id"`
	EntityType EntityType `json:"entity_type"`
	EntityID   int32      `json:"entity_id"`
	CreatedAt  time.Time  `json:"created_at"`
}
