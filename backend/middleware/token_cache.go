package middleware

import (
	"sync"
	"time"
)

// tokenCacheEntry holds whether a token is blacklisted and when this cache entry expires.
type tokenCacheEntry struct {
	blacklisted bool
	expiresAt   time.Time
}

// tokenCache is a simple in-memory TTL cache for token blacklist lookups.
// It avoids a Redis round-trip on every authenticated request.
type tokenCache struct {
	mu      sync.RWMutex
	entries map[string]tokenCacheEntry
}

var blacklistCache = &tokenCache{
	entries: make(map[string]tokenCacheEntry),
}

// get returns (blacklisted, found). If the entry has expired it is evicted.
func (c *tokenCache) get(token string) (blacklisted bool, found bool) {
	c.mu.RLock()
	entry, ok := c.entries[token]
	c.mu.RUnlock()

	if !ok {
		return false, false
	}
	if time.Now().After(entry.expiresAt) {
		// Expired — remove and treat as cache miss so Redis is consulted again.
		c.mu.Lock()
		delete(c.entries, token)
		c.mu.Unlock()
		return false, false
	}
	return entry.blacklisted, true
}

// set stores a result with the given TTL.
// - blacklisted=true  → cache for the full remaining token lifetime (long-lived).
// - blacklisted=false → cache for notBlacklistedTTL so a logout takes effect quickly.
func (c *tokenCache) set(token string, blacklisted bool, ttl time.Duration) {
	c.mu.Lock()
	c.entries[token] = tokenCacheEntry{
		blacklisted: blacklisted,
		expiresAt:   time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// invalidate removes a token from the cache immediately (called on logout).
func (c *tokenCache) invalidate(token string) {
	c.mu.Lock()
	delete(c.entries, token)
	c.mu.Unlock()
}

// startCacheJanitor runs a background goroutine that periodically evicts
// expired entries to prevent unbounded memory growth.
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

// How long a "not blacklisted" result is trusted before we re-check Redis.
// 60 s means a logout propagates to all in-flight requests within 1 minute.
const notBlacklistedTTL = 60 * time.Second
