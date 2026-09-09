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

func TestCompressionSafeguards(t *testing.T) {
	body := strings.Repeat("response with a secret token ", 200)
	tests := []struct {
		name, requestHeader, responseHeader, responseValue string
		guard, flush, implicit, compressed                 bool
	}{
		{name: "public", guard: true, compressed: true},
		{name: "guard off", requestHeader: "Cookie", compressed: true},
		{name: "cookie", guard: true, requestHeader: "Cookie"},
		{name: "authorization", guard: true, requestHeader: "Authorization"},
		{name: "csrf", guard: true, requestHeader: "X-Csrf-Token"},
		{name: "set cookie", guard: true, responseHeader: "Set-Cookie", responseValue: "session=secret"},
		{name: "implicit cookie", guard: true, implicit: true, responseHeader: "Set-Cookie", responseValue: "session=secret"},
		{name: "private", guard: true, responseHeader: "Cache-Control", responseValue: `public, Private="Set-Cookie"`},
		{name: "no store", guard: true, responseHeader: "Cache-Control", responseValue: "No-Store"},
		{name: "vary cookie", guard: true, responseHeader: "Vary", responseValue: "Origin, cOOkie"},
		{name: "flush cookie", guard: true, flush: true, responseHeader: "Set-Cookie", responseValue: "session=secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, encoding := range []string{"gzip", "identity"} {
				h := WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					if tt.responseHeader != "" {
						w.Header().Add(tt.responseHeader, tt.responseValue)
					}
					if tt.flush {
						w.(http.Flusher).Flush()
					} else if !tt.implicit {
						w.WriteHeader(http.StatusOK)
					}
					_, _ = io.WriteString(w, body)
				}), MiddlewareOptions{GzipEnabled: true, GzipJitter: 32, GzipDisableOnAuth: tt.guard})
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Accept-Encoding", encoding)
				if tt.requestHeader != "" {
					r.Header.Set(tt.requestHeader, "secret")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				resp := w.Result()
				require.Empty(t, resp.Header.Get("No-Gzip-Compression"))
				if tt.compressed && encoding == "gzip" {
					require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
					require.NotZero(t, w.Body.Bytes()[3]&0x10, "jitter must set the gzip comment flag")
					reader, err := gzip.NewReader(resp.Body)
					require.NoError(t, err)
					decoded, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, body, string(decoded))
					require.NoError(t, reader.Close())
				} else {
					require.Empty(t, resp.Header.Get("Content-Encoding"))
					require.Equal(t, body, w.Body.String())
				}
				require.NoError(t, resp.Body.Close())
			}
		})
	}
}

func TestCompressionContentTypesAndJitterOff(t *testing.T) {
	for _, contentType := range []string{
		"IMAGE/PNG", "image/apng", "image/avif-sequence", "image/gif", "image/heic-sequence",
		"image/heif", "image/jpeg", "image/jpg", "image/jxl", "image/png; charset=utf-8", "image/webp",
		"text/plain", "image/svg+xml", "image/bmp", "image/tiff",
	} {
		t.Run(contentType, func(t *testing.T) {
			h := newCompressionHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", contentType)
				_, _ = io.WriteString(w, strings.Repeat("body", 1000))
			}), 0, false)
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			switch contentType {
			case "text/plain", "image/svg+xml", "image/bmp", "image/tiff":
				require.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
				require.Zero(t, w.Body.Bytes()[3]&0x10)
			default:
				require.Empty(t, w.Header().Get("Content-Encoding"))
			}
		})
	}
}
