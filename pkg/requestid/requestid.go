// Package requestid assigns HTTP request identifiers and bounds their log representation.
package requestid

import (
	"net/http"
	"uuid"
)

// Header is shared by the client, application, and upstream proxy.
const Header = "X-Request-ID"

// Handler assigns one ID before downstream middleware runs. Client IDs are accepted
// only when the caller explicitly trusts the incoming proxy headers.
func Handler(next http.Handler, trustClientHeader bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !trustClientHeader || r.Header.Get(Header) == "" {
			r.Header.Set(Header, uuid.New().String())
		}
		w.Header().Set(Header, r.Header.Get(Header))
		next.ServeHTTP(w, r)
	})
}

// LogValue limits untrusted request IDs to 255 bytes in structured logs.
func LogValue(r *http.Request) string {
	id := r.Header.Get(Header)
	if len(id) > 255 {
		return id[:255]
	}
	return id
}
