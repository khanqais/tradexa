package middleware

import (
	"sync"
	"time"
)

type tokenCacheEntry struct {
	blacklisted	bool
	expiresAt	time.Time
}

type tokenCache struct {
	mu	sync.RWMutex
	entries	map[string]tokenCacheEntry
}

var blacklistCache = &tokenCache{
	entries: make(map[string]tokenCacheEntry),
}

func (c *tokenCache) get(token string) (blacklisted bool, found bool) {
	c.mu.RLock()
	entry, ok := c.entries[token]
	c.mu.RUnlock()

	if !ok {
		return false, false
	}
	if time.Now().After(entry.expiresAt) {

		c.mu.Lock()
		delete(c.entries, token)
		c.mu.Unlock()
		return false, false
	}
	return entry.blacklisted, true
}

func (c *tokenCache) set(token string, blacklisted bool, ttl time.Duration) {
	c.mu.Lock()
	c.entries[token] = tokenCacheEntry{
		blacklisted:	blacklisted,
		expiresAt:	time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

func (c *tokenCache) invalidate(token string) {
	c.mu.Lock()
	delete(c.entries, token)
	c.mu.Unlock()
}

func startCacheJanitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			blacklistCache.mu.Lock()
			for token, entry := range blacklistCache.entries {
				if now.After(entry.expiresAt) {
					delete(blacklistCache.entries, token)
				}
			}
			blacklistCache.mu.Unlock()
		}
	}()
}

const notBlacklistedTTL = 60 * time.Second
