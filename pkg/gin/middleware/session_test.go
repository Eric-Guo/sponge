package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRailsCookieAuthMiddleware_MissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RailsCookieAuthMiddleware("secret", "session"))
	r.GET("/", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestRailsCookieAuthMiddleware_InvalidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RailsCookieAuthMiddleware("secret", "session"))
	r.GET("/", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "not--valid--cookie"})
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestVerifyRailsSessionUserID(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     any
		status int
	}{
		{"integer", int64(1), 200}, {"json number", float64(1), 200}, {"string", "1", 200},
		{"other user", 2, 403}, {"fractional", 1.5, 401}, {"invalid", "oops", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("rails_session", map[string]any{"warden.user.user.key": []any{[]any{tc.id}, "salt"}})
			})
			r.Use(VerifyRailsSessionUserIDIs(1))
			r.GET("/", func(c *gin.Context) { c.String(200, "ok") })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			assert.Equal(t, tc.status, w.Code)
		})
	}
}
