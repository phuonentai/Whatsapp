package siigo

import (
	"sync"
	"time"
)

// tokenEntry is one organization's cached OAuth2 access token.
type tokenEntry struct {
	token    string
	expires  time.Time
	fetching bool
}

// tokenCache is a per-organization, TTL-bounded, single-flight in-memory cache
// for OAuth2 access tokens. Tokens MUST live only here (never in PostgreSQL or
// logs). Keys are organization ids; a nil refresh provider falls back to the
// adapter's env credentials under key 0.
type tokenCache struct {
	mu      sync.Mutex
	refresh func(orgID int32) (string, error)
	entries map[int32]*tokenEntry
}

func newTokenCache(refresh func(orgID int32) (string, error)) *tokenCache {
	return &tokenCache{refresh: refresh, entries: map[int32]*tokenEntry{}}
}

// Get returns a valid cached token for the org, refreshing (single-flight) if
// expired or missing.
func (c *tokenCache) Get(orgID int32) (string, error) {
	c.mu.Lock()
	entry, ok := c.entries[orgID]
	if !ok {
		entry = &tokenEntry{}
		c.entries[orgID] = entry
	}
	if entry.token != "" && time.Now().Before(entry.expires) {
		tok := entry.token
		c.mu.Unlock()
		return tok, nil
	}
	if entry.fetching {
		c.mu.Unlock()
		// Another caller is refreshing this org's token; wait for it.
		for {
			c.mu.Lock()
			if !entry.fetching {
				break
			}
			c.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
		if entry.token != "" {
			tok := entry.token
			c.mu.Unlock()
			return tok, nil
		}
		// Refresh failed; fall through and retry ourselves.
	}
	entry.fetching = true
	c.mu.Unlock()

	token, err := c.refresh(orgID)
	if err != nil {
		c.mu.Lock()
		entry.fetching = false
		c.mu.Unlock()
		return "", err
	}

	c.mu.Lock()
	entry.token = token
	entry.expires = time.Now().Add(TokenTTL)
	entry.fetching = false
	c.mu.Unlock()
	return token, nil
}

// invalidate clears the org's cached token so the next call refreshes (used on
// 401 responses).
func (c *tokenCache) invalidate(orgID int32) {
	c.mu.Lock()
	if entry, ok := c.entries[orgID]; ok {
		entry.token = ""
		entry.expires = time.Time{}
	}
	c.mu.Unlock()
}
