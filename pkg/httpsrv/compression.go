package httpsrv

import (
	"bufio"
	"net"
	"net/http"
	"strings"

	"github.com/klauspost/compress/gzhttp"
)

func newCompressionHandler(next http.Handler, jitter int, disableOnAuth bool) http.Handler {
	// The guard sits inside gzip so it can inspect response headers before gzip
	// chooses an encoding, including on explicit writes and streaming flushes.
	if disableOnAuth {
		next = compressionGuard(next)
	}
	wrapper, err := gzhttp.NewWrapper(
		gzhttp.MinSize(1024), gzhttp.CompressionLevel(6),
		gzhttp.ContentTypeFilter(compressibleContentType),
		gzhttp.RandomJitter(max(0, jitter), 0, false),
	)
	if err != nil {
		panic("create gzip middleware: " + err.Error())
	}
	return wrapper(next)
}

func compressibleContentType(contentType string) bool {
	if !gzhttp.DefaultContentTypeFilter(contentType) {
		return false
	}
	contentType = strings.ToLower(contentType)
	for _, prefix := range []string{
		"image/jpeg", "image/jpg", "image/png", "image/apng", "image/webp",
		"image/gif", "image/avif", "image/heic", "image/heif", "image/jxl",
	} {
		if strings.HasPrefix(contentType, prefix) {
			return false
		}
	}
	return true
}

func compressionGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Csrf-Token") != "" {
			w.Header().Set(gzhttp.HeaderNoCompression, "1")
		}
		guard := &compressionGuardWriter{ResponseWriter: w}
		next.ServeHTTP(guard, r)
		// Also cover responses that only set headers and never write a body.
		guard.checkHeaders()
	})
}

type compressionGuardWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *compressionGuardWriter) checkHeaders() {
	if w.wroteHeader {
		return
	}
	h := w.Header()
	sensitive := h.Get("Set-Cookie") != ""
	for _, value := range h.Values("Cache-Control") {
		for directive := range strings.SplitSeq(value, ",") {
			name, _, _ := strings.Cut(strings.TrimSpace(directive), "=")
			sensitive = sensitive || strings.EqualFold(name, "private") || strings.EqualFold(name, "no-store")
		}
	}
	for _, value := range h.Values("Vary") {
		for name := range strings.SplitSeq(value, ",") {
			sensitive = sensitive || strings.EqualFold(strings.TrimSpace(name), "Cookie")
		}
	}
	if sensitive {
		h.Set(gzhttp.HeaderNoCompression, "1")
	}
}

func (w *compressionGuardWriter) WriteHeader(code int) {
	w.checkHeaders()
	if code >= 200 || code == http.StatusSwitchingProtocols {
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *compressionGuardWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *compressionGuardWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *compressionGuardWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *compressionGuardWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
