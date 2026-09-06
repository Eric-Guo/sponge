package proxykit

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"go.uber.org/zap"

	spongelog "github.com/go-dev-frame/sponge/pkg/logger"
)

// Middleware is a function that takes a http.Handler and returns a http.Handler,
// used to build a middleware chain.
type Middleware func(http.Handler) http.Handler

// Chain links multiple middlewares together to form a single http.Handler.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Start from the last middleware and wrap backwards,
	// so that the first middleware is the outermost layer.
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

type logger struct {
	*zap.Logger
}

func newLogger() *logger {
	var l, _ = zap.NewProduction()
	return &logger{l}
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.Sugar().Infof(format, v...)
}

func (l *logger) Println(v ...interface{}) {
	l.Sugar().Info(v...)
}

var log = newLogger()

var doOnce sync.Once

func SetLogger(l *zap.Logger) {
	doOnce.Do(func() {
		log = &logger{l}
	})
}

// SendfileHandler converts X-Sendfile headers into direct file responses when enabled.
type SendfileHandler struct {
	enabled bool
	next    http.Handler
}

// NewSendfileHandler wraps the provided handler with X-Sendfile support.
func NewSendfileHandler(enabled bool, next http.Handler) *SendfileHandler {
	return &SendfileHandler{enabled: enabled, next: next}
}

// ServeHTTP sets up X-Sendfile translation when enabled before delegating to the next handler.
func (h *SendfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.enabled {
		r.Header.Set("X-Sendfile-Type", "X-Sendfile")
		w = &sendfileWriter{ResponseWriter: w, request: r}
	} else {
		r.Header.Del("X-Sendfile-Type")
	}

	h.next.ServeHTTP(w, r)
}

type sendfileWriter struct {
	http.ResponseWriter
	request       *http.Request
	headerWritten bool
	sendingFile   bool
}

func (w *sendfileWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	if w.sendingFile {
		return 0, http.ErrBodyNotAllowed
	}

	return w.ResponseWriter.Write(b)
}

func (w *sendfileWriter) WriteHeader(statusCode int) {
	filename := w.ResponseWriter.Header().Get("X-Sendfile")
	w.ResponseWriter.Header().Del("X-Sendfile")

	w.sendingFile = filename != ""
	w.headerWritten = true

	if w.sendingFile {
		w.serveFile(filename)
	} else {
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *sendfileWriter) serveFile(filename string) {
	spongelog.Debug("x-sendfile sending file", spongelog.String("path", filename))

	w.setContentLength(filename)
	http.ServeFile(w.ResponseWriter, w.request, filename)
}

func (w *sendfileWriter) setContentLength(filename string) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		w.ResponseWriter.Header().Del("Content-Length")
		return
	}

	w.ResponseWriter.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
}

func (w *sendfileWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter does not implement http.Hijacker")
	}

	return hijacker.Hijack()
}

func (w *sendfileWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sendfileWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}
