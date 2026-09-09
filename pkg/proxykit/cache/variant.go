package proxycache

import (
	"net/http"
	"slices"
	"strings"
)

// Variant captures the request characteristics that affect cacheability.
type Variant struct {
	r           *http.Request
	headerNames []string
}

// NewVariant builds a variant tracker for the provided request.
func NewVariant(r *http.Request) *Variant {
	return &Variant{r: r}
}

// SetResponseHeader inspects the response headers to learn about the Vary configuration.
func (v *Variant) SetResponseHeader(header http.Header) {
	v.headerNames = v.parseVaryHeader(header)
}

// HeaderNames returns the list of headers that influence the variant cache key.
func (v *Variant) HeaderNames() []string {
	return append([]string(nil), v.headerNames...)
}

// ApplyHeaderNames seeds the variant with a set of cached header names.
func (v *Variant) ApplyHeaderNames(names []string) {
	if len(names) == 0 {
		v.headerNames = nil
		return
	}
	v.headerNames = append([]string(nil), names...)
}

// CacheKey computes the stable cache key for the request variant.
func (v *Variant) CacheKey() CacheKey {
	vary := make([]string, len(v.headerNames))
	for i, name := range v.headerNames {
		vary[i] = name + "=" + strings.Join(v.r.Header.Values(name), "\x00")
	}
	return CacheKey{
		Method: strings.Clone(v.r.Method), Host: strings.Clone(v.r.Host),
		Path: strings.Clone(v.r.URL.EscapedPath()), Query: v.r.URL.Query().Encode(),
		Vary: strings.Join(vary, "\n"),
	}
}

// Matches verifies whether the response headers align with the request variant.
func (v *Variant) Matches(responseHeader http.Header) bool {
	for _, name := range v.headerNames {
		if !slices.Equal(responseHeader.Values(name), v.r.Header.Values(name)) {
			return false
		}
	}
	return true
}

// VariantHeader returns the headers that make this variant unique.
func (v *Variant) VariantHeader() http.Header {
	requestHeader := http.Header{}
	for _, name := range v.headerNames {
		requestHeader[name] = append([]string(nil), v.r.Header.Values(name)...)
	}
	return requestHeader
}

// Private

func (v *Variant) parseVaryHeader(responseHeader http.Header) []string {
	list := strings.Join(responseHeader.Values("Vary"), ",")
	if list == "" {
		return []string{}
	}

	names := strings.Split(list, ",")
	for i, name := range names {
		names[i] = http.CanonicalHeaderKey(strings.TrimSpace(name))
	}
	slices.Sort(names)

	return slices.Compact(names)
}
