package stytch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/redis"
	platformstytch "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/organizations/members"
)

// MemberDirectoryCacheKey is the Redis cache key for the member directory of
// an organization (conversation-row-scoping, task 1.4).
const memberDirectoryCacheKey = "auth:stytch:member-directory:org:%s"

// memberDirectoryCacheTTL matches the RBAC policy cache TTL (5 min): a member
// deactivated in Stytch may remain a valid reassignment target until the next
// refresh (residual S2 del VERDICT, ventana corta aceptada).
const memberDirectoryCacheTTL = 5 * time.Minute

// ErrMemberDirectoryUnavailable is returned when the directory cannot be
// resolved (circuit open or empty cache with Stytch unreachable). Handlers
// SHALL map it to 503 member_directory_unavailable.
var ErrMemberDirectoryUnavailable = errors.New("member_directory_unavailable")

// MemberDirectoryService resolves the list of active members of an
// organization from the Stytch B2B Members API, wrapped in the two-tier
// circuit breaker + Redis cache (5-min TTL, patrón de la política RBAC).
//
// Contrato validado en docs oficiales de Stytch:
//
//	POST /v1/b2b/organizations/members/search
//	  organization_ids = [org del solicitante], query vacía, paginación por
//	  next_cursor, filtro de status aplicado localmente (statuses: [active]).
//
// Solo se persiste/retorna stytch_member_id (FK lógico; nunca credenciales ni
// datos de sesión).
type MemberDirectoryService struct {
	api     *b2bstytchapi.API
	breaker *platformstytch.Client
	redis   redis.Client
	logger  logger.Logger
}

// NewMemberDirectoryService creates the directory service. breaker may be nil
// (no circuit guard) but production instances SHALL provide the consolidated
// breaker client via DI.
func NewMemberDirectoryService(api *b2bstytchapi.API, breaker *platformstytch.Client, redisClient redis.Client, log logger.Logger) *MemberDirectoryService {
	return &MemberDirectoryService{
		api:     api,
		breaker: breaker,
		redis:   redisClient,
		logger:  log,
	}
}

// ListActiveMemberIDs returns the stytch_member_ids of all active members of
// the organization (Stytch org UUID). Empty list + nil error = org sin
// miembros activos.
func (s *MemberDirectoryService) ListActiveMemberIDs(ctx context.Context, stytchOrgID string) ([]string, error) {
	cacheKey := fmt.Sprintf(memberDirectoryCacheKey, stytchOrgID)

	if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
		var ids []string
		unmarshalErr := json.Unmarshal([]byte(cached), &ids)
		if unmarshalErr == nil {
			s.logger.Debug("member directory fetched from cache", logger.Fields{
				"stytch_org_id": stytchOrgID,
				"members":       len(ids),
			})
			return ids, nil
		}
		s.logger.Warn("failed to unmarshal cached member directory", logger.Fields{
			"error": unmarshalErr.Error(),
		})
	}

	ids, err := s.fetchFromStytch(ctx, stytchOrgID)
	if err != nil {
		return nil, err
	}

	s.cache(ctx, cacheKey, ids)
	return ids, nil
}

// IsActiveMember reports whether the given member belongs to the org and is
// active. Used for server-side validation of reassignment targets (same-org).
func (s *MemberDirectoryService) IsActiveMember(ctx context.Context, stytchOrgID, stytchMemberID string) (bool, error) {
	ids, err := s.ListActiveMemberIDs(ctx, stytchOrgID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == stytchMemberID {
			return true, nil
		}
	}
	return false, nil
}

// fetchFromStytch paginates Members.Search until next_cursor is exhausted,
// keeping only members with status active.
func (s *MemberDirectoryService) fetchFromStytch(ctx context.Context, stytchOrgID string) ([]string, error) {
	var ids []string
	cursor := ""

	for {
		var pageIDs []string
		var nextCursor string

		fetch := func() error {
			params := &members.SearchParams{
				OrganizationIds: []string{stytchOrgID},
				Cursor:          cursor,
				Limit:           1000,
			}
			resp, err := s.api.Organizations.Members.Search(ctx, params)
			if err != nil {
				return fmt.Errorf("stytch members search failed: %w", err)
			}
			for _, m := range resp.Members {
				// statuses: [active] — filter localmente (el query de search no
				// soporta filtro de status; ver contrato validado).
				if m.Status == "active" {
					pageIDs = append(pageIDs, m.MemberID)
				}
			}
			nextCursor = resp.ResultsMetadata.NextCursor
			return nil
		}

		if s.breaker != nil {
			if err := s.breaker.Run(ctx, fetch); err != nil {
				s.logDirectoryFailure(err)
				return nil, ErrMemberDirectoryUnavailable
			}
		} else {
			if err := fetch(); err != nil {
				s.logDirectoryFailure(err)
				return nil, ErrMemberDirectoryUnavailable
			}
		}

		ids = append(ids, pageIDs...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	s.logger.Info("member directory resolved from Stytch", logger.Fields{
		"stytch_org_id": stytchOrgID,
		"active_members": len(ids),
	})
	return ids, nil
}

// logDirectoryFailure logs a directory fetch failure distinctly for
// circuit-open vs API errors.
func (s *MemberDirectoryService) logDirectoryFailure(err error) {
	if errors.Is(err, platformstytch.ErrCircuitOpen) {
		s.logger.Warn("member directory blocked by circuit breaker (directory unavailable)", logger.Fields{
			"error": err.Error(),
		})
		return
	}
	s.logger.Error("failed to fetch member directory from Stytch (directory unavailable)", logger.Fields{
		"error": err.Error(),
	})
}

func (s *MemberDirectoryService) cache(ctx context.Context, cacheKey string, ids []string) {
	data, err := json.Marshal(ids)
	if err != nil {
		s.logger.Warn("failed to marshal member directory for caching", logger.Fields{"error": err.Error()})
		return
	}
	if err := s.redis.Set(ctx, cacheKey, string(data), memberDirectoryCacheTTL); err != nil {
		s.logger.Warn("failed to cache member directory", logger.Fields{"error": err.Error()})
	}
}
