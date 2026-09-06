package middleware

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/gin/middleware/auth"
)

// 1. Universal session middleware example refer to https://github.com/gin-contrib/sessions?tab=readme-ov-file#basic-examples

// -------------------------------------------------------------------------------------------

// 2. Special session for rails

// RailsCookieAuthMiddleware validates and decrypts a Rails encrypted cookie,
// attaches the session payload to context under key "rails_session".
func RailsCookieAuthMiddleware(secretKeyBase string, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing cookie"})
			return
		}

		session, err := auth.DecodeSignedCookie(secretKeyBase, cookie, cookieName)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid cookie"})
			return
		}

		c.Set("rails_session", session)
		c.Next()
	}
}

const sessionErrorKey = "error"

// VerifyRailsSessionUserIDIs returns a middleware that verifies the rails session
// contains a warden user id.
func VerifyRailsSessionUserIDIs(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("rails_session")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "rails_session missing"})
			return
		}
		session, ok := v.(map[string]any)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid rails_session"})
			return
		}
		uidVal, ok := auth.UserIDFromSession(session)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "user id not found in session"})
			return
		}
		var uid int64
		switch v := uidVal.(type) {
		case int64:
			uid = v
		case int:
			uid = int64(v)
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || v < -9223372036854775808 || v >= 9223372036854775808 {
				c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid user id in session"})
				return
			}
			uid = int64(v)
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid user id in session"})
				return
			}
			uid = parsed
		default:
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid user id type in session"})
			return
		}
		if uid != userID {
			c.AbortWithStatusJSON(403, gin.H{sessionErrorKey: "forbidden"})
			return
		}
		c.Next()
	}
}
