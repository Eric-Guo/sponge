package requestid

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestIDTrustAndLogLimit(t *testing.T) {
	for _, trust := range []bool{false, true} {
		for _, supplied := range []string{"", "client-id", strings.Repeat("x", 300)} {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set(Header, supplied)
			w := httptest.NewRecorder()
			Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, r.Header.Get(Header), w.Header().Get(Header))
				require.LessOrEqual(t, len(LogValue(r)), 255)
			}), trust).ServeHTTP(w, r)
			if trust && supplied != "" {
				require.Equal(t, supplied, w.Header().Get(Header))
			} else {
				require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, w.Header().Get(Header))
			}
		}
	}
}
