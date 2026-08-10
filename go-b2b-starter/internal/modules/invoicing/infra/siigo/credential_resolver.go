package siigo

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/secrets"
)

// CredentialResolver implements domain.CredentialProvider: it reads the
// organization's stored ciphertext and decrypts it with the envelope master
// key at call time. Plaintext exists only in memory for the duration of the
// call; nothing is logged or returned.
type CredentialResolver struct {
	repo domain.ConnectionRepository
	env  *secrets.Envelope
}

func NewCredentialResolver(repo domain.ConnectionRepository, env *secrets.Envelope) domain.CredentialProvider {
	return &CredentialResolver{repo: repo, env: env}
}

func (r *CredentialResolver) ResolveCredentials(ctx context.Context, orgID int32) (string, string, error) {
	conn, err := r.repo.Get(ctx, orgID)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", domain.ErrCredentialResolution, err)
	}
	if !conn.HasCredentials() {
		return "", "", fmt.Errorf("%w: no credentials stored for organization %d", domain.ErrCredentialResolution, orgID)
	}
	clientID, err := r.env.Decrypt(conn.ClientIDEnc)
	if err != nil {
		return "", "", fmt.Errorf("%w: client id decrypt failed", domain.ErrCredentialResolution)
	}
	clientSecret, err := r.env.Decrypt(conn.ClientSecretEnc)
	if err != nil {
		return "", "", fmt.Errorf("%w: client secret decrypt failed", domain.ErrCredentialResolution)
	}
	return clientID, clientSecret, nil
}
