package proxycache

import (
	"encoding/binary"
	"time"

	"github.com/dgraph-io/ristretto"

	spongecache "github.com/Eric-Guo/sponge/pkg/cache"
	"github.com/Eric-Guo/sponge/pkg/logger"
)

// GetCurrentTime allows overriding time in tests.
type GetCurrentTime func() time.Time

// MemoryCache provides a cache implementation backed by sponge's ristretto cache.
type MemoryCache struct {
	client         *ristretto.Cache
	capacity       int
	maxItemSize    int
	getCurrentTime GetCurrentTime
}

type memoryEntry struct {
	key   CacheKey
	value []byte
}

// storageKey encodes component boundaries for ristretto. The stored key is also
// compared on Get, so even an internal ristretto hash collision becomes a miss.
func storageKey(key CacheKey) string {
	var encoded []byte
	for _, part := range []string{key.Method, key.Host, key.Path, key.Query, key.Vary} {
		encoded = binary.AppendUvarint(encoded, uint64(len(part)))
		encoded = append(encoded, part...)
	}
	return string(encoded)
}

// NewMemoryCache constructs a memory cache bounded by capacity and per-item size.
func NewMemoryCache(capacity, maxItemSize int) *MemoryCache {
	opts := []spongecache.Option{}
	if capacity > 0 {
		opts = append(opts, spongecache.WithMaxCost(int64(capacity)))
		if numCounters := deriveNumCounters(capacity, maxItemSize); numCounters > 0 {
			opts = append(opts, spongecache.WithNumCounters(numCounters))
		}
	}

	client := spongecache.InitMemory(opts...)

	return &MemoryCache{
		client:         client,
		capacity:       capacity,
		maxItemSize:    maxItemSize,
		getCurrentTime: time.Now,
	}
}

// Set stores a value if it fits per-item limits, leveraging sponge's cache for eviction.
func (c *MemoryCache) Set(key CacheKey, value []byte, expiresAt time.Time) {
	if c.client == nil || !key.isCacheable() {
		return
	}

	itemSize := key.size() + len(value)
	if len(value) > c.maxItemSize || (c.capacity > 0 && itemSize > c.capacity) {
		logger.Debug(
			"proxy cache: item too large",
			logger.Int("item_size", itemSize),
			logger.Int("max_item_size", c.maxItemSize),
			logger.Int("capacity", c.capacity),
		)
		return
	}

	currentTime := c.getCurrentTime()
	ttl := expiresAt.Sub(currentTime)
	if ttl <= 0 {
		logger.Debug(
			"proxy cache: item already expired",
			logger.Time("expires_at", expiresAt),
		)
		return
	}

	valueCopy := append([]byte(nil), value...)
	if ok := c.client.SetWithTTL(storageKey(key), memoryEntry{key: key, value: valueCopy}, int64(itemSize), ttl); !ok {
		logger.Debug(
			"proxy cache: failed to store item",
			logger.Int("size", itemSize),
		)
		return
	}
	c.client.Wait()

	logger.Debug(
		"proxy cache: item stored",
		logger.Int("size", itemSize),
		logger.Time("expires_at", expiresAt),
	)
}

// Get retrieves a stored item when present and not expired.
func (c *MemoryCache) Get(key CacheKey) ([]byte, bool) {
	if c.client == nil || !key.isCacheable() {
		return nil, false
	}

	value, ok := c.client.Get(storageKey(key))
	if !ok {
		return nil, false
	}

	entry, ok := value.(memoryEntry)
	if !ok || entry.key != key {
		return nil, false
	}

	return append([]byte(nil), entry.value...), true
}

// deriveNumCounters sizes ristretto's frequency sketch so metadata overhead scales with the cache capacity.
func deriveNumCounters(capacity, maxItemSize int) int64 {
	if capacity <= 0 {
		return 0
	}

	const (
		minCounters     = 1_000
		maxCounters     = 10_000_000
		minAvgItemBytes = 1 << 10  // assume responses are at least 1KiB on average
		maxAvgItemBytes = 16 << 10 // cap assumed average at 16KiB to avoid undersizing
	)

	avgItemBytes := maxItemSize / 4
	if maxItemSize <= 0 {
		avgItemBytes = minAvgItemBytes
	}
	if avgItemBytes < minAvgItemBytes {
		avgItemBytes = minAvgItemBytes
	}
	if avgItemBytes > maxAvgItemBytes {
		avgItemBytes = maxAvgItemBytes
	}

	estimatedItems := capacity / avgItemBytes
	if estimatedItems <= 0 {
		estimatedItems = 1
	}

	numCounters := int64(estimatedItems * 10)
	if numCounters < minCounters {
		numCounters = minCounters
	}
	if numCounters > maxCounters {
		numCounters = maxCounters
	}

	return numCounters
}
