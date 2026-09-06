package httpsrv

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapHandlerRequestStartAndGzip(t *testing.T) {
	handler := WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Started", r.Header.Get("X-Request-Start"))
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, strings.Repeat("body", 1000))
	}), MiddlewareOptions{AddRequestStartHeader: true, GzipEnabled: true, LogRequests: true})
	for _, start := range []string{"", "t=123"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("X-Request-Start", start)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		require.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
		require.NotEmpty(t, w.Header().Get("X-Started"))
		if start != "" {
			require.Equal(t, start, w.Header().Get("X-Started"))
		}
		reader, err := gzip.NewReader(w.Body)
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		require.Equal(t, strings.Repeat("body", 1000), string(body))
	}
}

func TestWrapHandlerBodyLimit(t *testing.T) {
	handler := WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 413)
			return
		}
		w.WriteHeader(204)
	}), MiddlewareOptions{MaxRequestBodyBytes: 4})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader("12345")))
	require.Equal(t, 413, w.Code)
}
