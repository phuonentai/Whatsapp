// Package auth provides a unified authentication and authorization layer.
//
// This package abstracts away the authentication provider (Stytch, Auth0, etc.)
// and provides a clean interface for the rest of the application to use.
//
// # Architecture
//
// The auth package follows the adapter pattern:
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│                        Application Layer                        │
//	│  (handlers, services - use auth.GetRequestContext, auth.RequirePermission) │
//	└─────────────────────────────────────────────────────────────────┘
//	                              │
//	                              ▼
//	┌─────────────────────────────────────────────────────────────────┐
//	│                         auth package                            │
//	│  • AuthProvider interface                                       │
//	│  • Identity (provider-agnostic user representation)            │
//	│  • RequestContext (resolved database IDs)                      │
//	│  • Middleware (RequireAuth, RequireOrganization, RequirePermission) │
//	│  • Type-safe context helpers                                   │
//	└─────────────────────────────────────────────────────────────────┘
//	                              │
//	                              ▼
//	┌─────────────────────────────────────────────────────────────────┐
//	│                    auth/adapters/stytch                         │
//	│  (Stytch-specific implementation - hidden from app layer)      │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Usage
//
// In routes:
//
//	router.Use(
//	    auth.RequireAuth(authProvider),
//	    auth.RequireOrganization(orgRepo, accountRepo, logger),
//	)
//	router.GET("/resource", auth.RequirePermission("resource", "view"), handler)
//
// In handlers:
//
//	func Handler(c *gin.Context) {
//	    reqCtx := auth.GetRequestContext(c)
//	    orgID := reqCtx.OrganizationID  // int32, type-safe
//	    accountID := reqCtx.AccountID   // int32, type-safe
//	}
//
// # Adding a New Auth Provider
//
// To add a new authentication provider (e.g., Auth0, Firebase):
//
//  1. Create a new adapter in auth/adapters/<provider>/
//  2. Implement the AuthProvider interface
//  3. Map provider-specific claims to auth.Identity
//  4. Register the adapter in the DI container
//
// See auth/adapters/stytch/ for a reference implementation.
package auth

import (
	"context"
)

// AuthProvider abstracts the authentication provider (Stytch, Auth0, Firebase, etc.).
//
// Implementations must:
//   - Verify the token signature and validity
//   - Extract user identity information
//   - Derive permissions from roles (if applicable)
//   - Return appropriate errors for invalid/expired tokens
//
// The application layer should only depend on this interface, never on
// provider-specific implementations.
type AuthProvider interface {
	// VerifyToken validates the provided token and returns the user's identity.
	//
	// The token is typically a JWT from the Authorization header.
	// Returns ErrInvalidToken, ErrTokenExpired, or other auth errors on failure.
	VerifyToken(ctx context.Context, token string) (*Identity, error)
}

// OrganizationRepository defines the interface for looking up organizations.
//
// This is used by the RequireOrganization middleware to resolve
// the auth provider's organization ID to a database ID.
type OrganizationRepository interface {
	// GetByProviderID looks up an organization by the auth provider's organization ID.
	// Returns the organization with its database ID, or an error if not found.
	GetByProviderID(ctx context.Context, providerOrgID string) (*Organization, error)
}

// AccountRepository defines the interface for looking up accounts.
//
// This is used by the RequireOrganization middleware to resolve
// the user's email to a database account ID within an organization.
type AccountRepository interface {
	// GetByEmail looks up an account by email within an organization.
	// Returns the account with its database ID, or an error if not found.
	GetByEmail(ctx context.Context, orgID int32, email string) (*Account, error)
}

// Organization represents the minimal organization data needed by the auth package.
type Organization struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Account represents the minimal account data needed by the auth package.
type Account struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}
