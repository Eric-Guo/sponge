package proxycache

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/requestid"
)

// CacheKey keeps request components distinct, preventing ambiguous concatenation
// and hash collisions from sharing responses between different requests.
type CacheKey struct {
	Method string
	Host   string
	Path   string
	Query  string
	Vary   string
}

func (k CacheKey) size() int {
	return len(k.Method) + len(k.Host) + len(k.Path) + len(k.Query) + len(k.Vary)
}

func (k CacheKey) isCacheable() bool { return k.size() <= 8*1024 }

type variantIndexEntry struct {
	headers []string
	expires time.Time
}

// Cache describes the storage backend used by the handler.
type Cache interface {
	Get(key CacheKey) ([]byte, bool)
	Set(key CacheKey, value []byte, expiresAt time.Time)
}

// CacheHandler intercepts responses to add caching semantics around the next handler.
type CacheHandler struct {
	cache       Cache
	next        http.Handler
	maxBodySize int

	varyIndexMu sync.Mutex
	varyIndex   map[CacheKey]variantIndexEntry
}

// NewCacheHandler constructs a caching handler in front of the provided next handler.
func NewCacheHandler(cache Cache, maxBodySize int, next http.Handler) *CacheHandler {
	return &CacheHandler{
		cache:       cache,
		next:        next,
		maxBodySize: maxBodySize,
		varyIndex:   make(map[CacheKey]variantIndexEntry),
	}
}

// ServeHTTP attempts to serve a cached response, falling back to the next handler.
func (h *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.shouldCacheRequest(r) {
		h.bypass(w, r)
		return
	}

	variant := NewVariant(r)
	baseKey := variant.CacheKey()
	if !baseKey.isCacheable() {
		h.bypass(w, r)
		return
	}
	variant.ApplyHeaderNames(h.loadVariantHeaders(baseKey))
	if !variant.CacheKey().isCacheable() {
		h.bypass(w, r)
		return
	}
	response, found := h.lookupCacheEntry(r, variant.CacheKey())

	if found {
		variant.SetResponseHeader(response.HTTPHeader)
		if !variant.Matches(response.VariantHeader) {
			found = false
		}
	}

	if found {
		response.WriteCachedResponse(w, r)
		return
	}

	cr := NewCacheableResponse(w, h.maxBodySize)
	h.next.ServeHTTP(cr, r)

	cacheable, expires := cr.CacheStatus()
	if cacheable {
		variant.SetResponseHeader(cr.HTTPHeader)
		key := variant.CacheKey()
		h.rememberVariantHeaders(baseKey, variant.HeaderNames(), expires)
		if !key.isCacheable() {
			return
		}
		cr.VariantHeader = variant.VariantHeader()

		encoded, err := cr.ToBuffer()
		if err != nil {
			logger.Error("proxy cache: encode response failed", logger.String("request_id", requestid.LogValue(r)), logger.String("path", r.URL.Path), logger.Err(err))
		} else {
			h.cache.Set(key, encoded, expires)
			logger.Debug("proxy cache: stored response", logger.String("request_id", requestid.LogValue(r)), logger.String("path", r.URL.Path), logger.Int("size", len(encoded)))
		}
	}
}

// Private

func (h *CacheHandler) bypass(w http.ResponseWriter, r *http.Request) {
	logger.Debug("proxy cache: bypassing request", logger.String("request_id", requestid.LogValue(r)), logger.String("path", r.URL.Path), logger.String("method", r.Method))
	w.Header().Set("X-Cache", "bypass")
	h.next.ServeHTTP(w, r)
}

func (h *CacheHandler) lookupCacheEntry(r *http.Request, key CacheKey) (CacheableResponse, bool) {
	cached, found := h.cache.Get(key)
	if !found {
		return CacheableResponse{}, false
	}

	response, err := CacheableResponseFromBuffer(cached)
	if err != nil {
		logger.Error("proxy cache: decode cached response failed", logger.String("request_id", requestid.LogValue(r)), logger.String("path", r.URL.Path), logger.Err(err))
		return CacheableResponse{}, false
	}

	return response, true
}

func (h *CacheHandler) rememberVariantHeaders(baseKey CacheKey, headers []string, expires time.Time) {
	h.varyIndexMu.Lock()
	defer h.varyIndexMu.Unlock()

	indexSize := baseKey.size()
	for _, name := range headers {
		indexSize += len(name)
	}
	if len(headers) == 0 || indexSize > 8*1024 {
		delete(h.varyIndex, baseKey)
		return
	}

	copyHeaders := append([]string(nil), headers...)
	// This index is only a lookup hint. Bound it separately from response storage;
	// forgetting a hint causes a miss, never a response from another variant.
	if len(h.varyIndex) >= 4096 {
		for key := range h.varyIndex {
			delete(h.varyIndex, key)
			break
		}
	}
	h.varyIndex[baseKey] = variantIndexEntry{headers: copyHeaders, expires: expires}
}

func (h *CacheHandler) loadVariantHeaders(baseKey CacheKey) []string {
	h.varyIndexMu.Lock()
	defer h.varyIndexMu.Unlock()

	if entry, ok := h.varyIndex[baseKey]; ok {
		if time.Now().Before(entry.expires) {
			return append([]string(nil), entry.headers...)
		}
		delete(h.varyIndex, baseKey)
	}

	return nil
}

func (h *CacheHandler) shouldCacheRequest(r *http.Request) bool {
	allowedMethod := r.Method == http.MethodGet || r.Method == http.MethodHead
	isUpgrade := strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") || r.Header.Get("Upgrade") != ""
	isRange := r.Header.Get("Range") != ""

	return allowedMethod && !isUpgrade && !isRange
}
