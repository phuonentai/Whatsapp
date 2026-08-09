package siigo

import (
	"sync"
	"time"
)

// tokenCache is a TTL-bounded, single-flight in-memory cache for the OAuth2
// access token. Tokens MUST live only here (never in PostgreSQL or logs).
type tokenCache struct {
	mu       sync.Mutex
	refresh  func() (string, error)
	token    string
	expires  time.Time
	fetching bool
}

func newTokenCache(refresh func() (string, error)) *tokenCache {
	return &tokenCache{refresh: refresh}
}

// Get returns a valid cached token, refreshing (single-flight) if expired.
func (c *tokenCache) Get() (string, error) {
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.expires) {
		tok := c.token
		c.mu.Unlock()
		return tok, nil
	}
	if c.fetching {
		c.mu.Unlock()
		// Another caller is refreshing; wait for it to finish.
		for {
			c.mu.Lock()
			if !c.fetching {
				break
			}
			c.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
		if c.token != "" {
			tok := c.token
			c.mu.Unlock()
			return tok, nil
		}
		// Refresh failed; fall through and retry ourselves.
	}
	c.fetching = true
	c.mu.Unlock()

	token, err := c.refresh()
	if err != nil {
		c.mu.Lock()
		c.fetching = false
		c.mu.Unlock()
		return "", err
	}

	c.mu.Lock()
	c.token = token
	c.expires = time.Now().Add(TokenTTL)
	c.fetching = false
	c.mu.Unlock()
	return token, nil
}

// invalidate clears a cached token so the next call refreshes (used on 401).
func (c *tokenCache) invalidate() {
	c.mu.Lock()
	c.token = ""
	c.expires = time.Time{}
	c.mu.Unlock()
}
