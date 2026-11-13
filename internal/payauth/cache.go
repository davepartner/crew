package payauth

import (
	"sync"
	"time"
)

type cacheEntry struct {
	user      User
	expiresAt time.Time
}

// Cache keeps validated sessions for a short window so a page load does not
// make a dozen round trips to PayTrackify. Revocation therefore takes at most
// TTL seconds to take effect everywhere — short by design.
type Cache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]cacheEntry
	max     int
}

func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &Cache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
		max:     5000,
	}
}

func (c *Cache) Get(token string) (User, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[token]
	if !ok {
		return User{}, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.entries, token)
		return User{}, false
	}

	return entry.user, true
}

func (c *Cache) Set(token string, user User) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		// Evict an arbitrary entry; the cache is an optimisation, not a store.
		for key := range c.entries {
			delete(c.entries, key)
			break
		}
	}

	c.entries[token] = cacheEntry{user: user, expiresAt: time.Now().Add(c.ttl)}
}

func (c *Cache) Delete(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, token)
}
