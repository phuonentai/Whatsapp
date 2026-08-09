package domain

import "context"

type SignupFlowRepository interface {
	Upsert(ctx context.Context, flow *SignupFlow) (*SignupFlow, error)
	GetByOrganization(ctx context.Context, orgID int32) (*SignupFlow, error)
	UpdateStatus(ctx context.Context, orgID int32, status SignupStatus, step, errorCode string, retryCount int, metadata map[string]interface{}) (*SignupFlow, error)
}
