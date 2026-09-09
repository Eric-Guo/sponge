package proxycache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCacheSeparatesAmbiguousRequestComponents(t *testing.T) {
	h := NewCacheHandler(newRecordingCache(), 1024, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = fmt.Fprint(w, r.URL.RequestURI())
	}))
	// These previously hashed the same concatenation of path + query + host.
	for _, target := range []string{"http://example.com/foo?a=b", "http://example.com/fooa=b", "http://example.com/a%2Fb", "http://example.com/a/b"} {
		for _, status := range []string{"miss", "hit"} {
			r := httptest.NewRequest("GET", target, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, status, w.Header().Get("X-Cache"))
			require.Equal(t, r.URL.RequestURI(), w.Body.String())
		}
	}
}

func TestUncacheableRequestsBypassWarmCache(t *testing.T) {
	cache := newRecordingCache()
	var hits int
	h := NewCacheHandler(cache, 1024, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = fmt.Fprintf(w, "origin %d", hits)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/asset", nil))
	for _, r := range []*http.Request{
		httptest.NewRequest("POST", "/asset", nil),
		httptest.NewRequest("GET", "/asset?q="+strings.Repeat("a", 9*1024), nil),
		httptest.NewRequest("GET", "/asset", nil),
		httptest.NewRequest("GET", "/asset", nil),
	} {
		if hits == 3 {
			r.Header.Set("Range", "bytes=0-1")
		}
		if hits == 4 {
			r.Header.Set("Connection", "keep-alive, UpGrAdE")
			r.Header.Set("Upgrade", "websocket")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, "bypass", w.Header().Get("X-Cache"))
		require.NotEqual(t, "origin 1", w.Body.String())
	}
	require.Equal(t, 5, hits)
	require.Len(t, cache.entries, 1)
}

func TestOversizedVaryKeyAndMultipleVaryLines(t *testing.T) {
	cache := newRecordingCache()
	h := NewCacheHandler(cache, 1024, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Add("Vary", "Accept")
		w.Header().Add("Vary", "Accept-Language")
		_, _ = fmt.Fprint(w, r.Header.Get("Accept-Language"))
	}))
	for i, language := range []string{"en", "fr", "en", "fr"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", language)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, language, w.Body.String())
		if i >= 2 {
			require.Equal(t, "hit", w.Header().Get("X-Cache"))
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept", strings.Repeat("a", 9*1024))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, "bypass", w.Header().Get("X-Cache"))
	require.Len(t, cache.entries, 2)
}

func TestMemoryCacheKeyBoundsAndCollisionCheck(t *testing.T) {
	c := NewMemoryCache(1024, 1024)
	t.Cleanup(c.client.Close)
	key := CacheKey{Method: "GET", Host: "example.com", Path: "/"}
	c.Set(key, []byte("public"), time.Now().Add(time.Minute))
	value, ok := c.Get(key)
	require.True(t, ok)
	require.Equal(t, "public", string(value))
	largeKey := key
	largeKey.Query = strings.Repeat("x", 1000)
	c.Set(largeKey, []byte(strings.Repeat("a", 100)), time.Now().Add(time.Minute))
	_, ok = c.Get(largeKey)
	require.False(t, ok, "cache budget must include request key bytes")
	// Simulate the storage engine returning an entry from a colliding hash.
	otherKey := key
	otherKey.Host = "private.example"
	require.True(t, c.client.SetWithTTL(storageKey(key), memoryEntry{key: otherKey, value: []byte("secret")}, 100, time.Minute))
	c.client.Wait()
	_, ok = c.Get(key)
	require.False(t, ok)
}
