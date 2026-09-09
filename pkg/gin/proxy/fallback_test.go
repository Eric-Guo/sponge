package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func TestFallbackKeepsAPIRoutesAndCachesAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hits := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Set-Cookie", "session=origin")
		_, _ = w.Write([]byte(r.URL.RequestURI()))
	}))
	defer backend.Close()
	r := gin.New()
	r.GET("/api/v1/local", func(c *gin.Context) { c.String(200, "local") })
	cfg := FallbackConfig{Proxy: FallbackProxyConfig{Enabled: true, TargetURL: backend.URL,
		Cache: FallbackCacheConfig{Enabled: true, CapacityBytes: 1 << 20, MaxItemSizeBytes: 1 << 16, MaxResponseBodyBytes: 1 << 16},
	}}
	require.NoError(t, RegisterFallback(r, cfg))
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), "GET", path, nil))
		return w
	}
	require.Equal(t, "local", request("/api/v1/local").Body.String())
	first := request("/assets/app.js?v=1")
	require.Equal(t, "/assets/app.js?v=1", first.Body.String())
	require.Equal(t, "session=origin", first.Header().Get("Set-Cookie"))
	second := request("/assets/app.js?v=1")
	require.Equal(t, "hit", second.Header().Get("X-Cache"))
	require.Empty(t, second.Header().Get("Set-Cookie"))
	require.Equal(t, 1, hits)
}

func TestFallbackUpgradeThroughCompressionGuard(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nX-Request-ID: origin\r\n\r\n")
		_ = rw.Flush()
		line, err := rw.ReadString('\n')
		if err == nil {
			_, _ = rw.WriteString(line)
			_ = rw.Flush()
		}
	}))
	defer backend.Close()
	r := gin.New()
	require.NoError(t, RegisterFallback(r, FallbackConfig{Proxy: FallbackProxyConfig{
		Enabled: true, TargetURL: backend.URL, XSendfileEnabled: true,
		Cache: FallbackCacheConfig{Enabled: true, CapacityBytes: 1 << 20, MaxItemSizeBytes: 1 << 16},
	}}))
	front := httptest.NewServer(httpsrv.WrapHandler(r, httpsrv.MiddlewareOptions{
		AddRequestID: true, GzipEnabled: true, GzipDisableOnAuth: true, GzipJitter: 32, LogRequests: true,
	}))
	defer front.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", front.URL+"/cable", nil)
	require.NoError(t, err)
	req.Header.Set("Connection", "Upgrade, X-Request-ID")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Cookie", "session=secret")
	resp, err := front.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	require.Empty(t, resp.Header.Get("Content-Encoding"))
	require.Equal(t, "bypass", resp.Header.Get("X-Cache"))
	require.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	require.NotEqual(t, "origin", resp.Header.Get("X-Request-ID"))
	stream, ok := resp.Body.(io.ReadWriteCloser)
	require.True(t, ok)
	_, err = io.WriteString(stream, "ping\n")
	require.NoError(t, err)
	data := make([]byte, 5)
	_, err = io.ReadFull(stream, data)
	require.NoError(t, err)
	require.Equal(t, "ping\n", string(data))
}

func TestFallbackUnixSocketSurvivesHealthCheck(t *testing.T) {
	dir, err := os.MkdirTemp("", "sp-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "u.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("socket")) })}
	defer srv.Close()
	go func() { _ = srv.Serve(listener) }()
	r := gin.New()
	cfg := FallbackConfig{Proxy: FallbackProxyConfig{Enabled: true, HealthCheck: FallbackHealthCheck{IntervalSeconds: 1, TimeoutSeconds: 1}}}
	cfg.Upstream.TargetBindSocket = "unix://" + socket
	require.NoError(t, RegisterFallback(r, cfg))
	time.Sleep(1200 * time.Millisecond)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), "GET", "/socket", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, "socket", w.Body.String())
}

func TestFallbackSendfileHeaderCasing(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "asset.txt")
	require.NoError(t, os.WriteFile(filename, []byte("file contents"), 0600))
	for _, header := range []string{"X-Sendfile", "x-sendfile"} {
		t.Run(header, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "X-Sendfile", r.Header.Get("X-Sendfile-Type"))
				w.Header()[header] = []string{filename}
				w.WriteHeader(200)
			}))
			defer backend.Close()
			r := gin.New()
			require.NoError(t, RegisterFallback(r, FallbackConfig{Proxy: FallbackProxyConfig{Enabled: true, TargetURL: backend.URL, XSendfileEnabled: true}}))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), "GET", "/download", nil))
			require.Equal(t, 200, w.Code)
			require.Equal(t, "file contents", w.Body.String())
			require.Empty(t, w.Header().Get("X-Sendfile"))
		})
	}
}

func TestFallbackH2C(t *testing.T) {
	backend := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	}), &http2.Server{}))
	defer backend.Close()
	r := gin.New()
	require.NoError(t, RegisterFallback(r, FallbackConfig{Proxy: FallbackProxyConfig{Enabled: true, TargetURL: backend.URL, H2cEnabled: true}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), "GET", "/", nil))
	require.Equal(t, "HTTP/2.0", w.Body.String())
}
