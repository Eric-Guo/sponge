// Package middleware is gin middleware plugin.
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	"github.com/Eric-Guo/sponge/pkg/gin/response"
	"github.com/Eric-Guo/sponge/pkg/jwt"
)

// HeaderAuthorizationKey http header authorization key, value is "Bearer token"
const HeaderAuthorizationKey = "Authorization"

// ExtraVerifyFn extra verify function
type ExtraVerifyFn = func(claims *jwt.Claims, c *gin.Context) error

// AuthOption set the auth options.
type AuthOption func(*authOptions)

type authOptions struct {
	signKey           []byte // sign key for jwt
	isReturnErrReason bool
	extraVerifyFn     ExtraVerifyFn
}

func defaultAuthOptions() *authOptions {
	return &authOptions{}
}

func (o *authOptions) apply(opts ...AuthOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithSignKey set jwt sign key
func WithSignKey(key []byte) AuthOption {
	return func(o *authOptions) {
		o.signKey = key
	}
}

// WithReturnErrReason set return error reason
func WithReturnErrReason() AuthOption {
	return func(o *authOptions) {
		o.isReturnErrReason = true
	}
}

// WithExtraVerify set extra verify function
func WithExtraVerify(fn ExtraVerifyFn) AuthOption {
	return func(o *authOptions) {
		o.extraVerifyFn = fn
	}
}

// WithVerify alias of WithExtraVerify
var WithVerify = WithExtraVerify

func responseUnauthorized(isReturnErrReason bool, errMsg string) *errcode.Error {
	if isReturnErrReason {
		return errcode.Unauthorized.RewriteMsg("Unauthorized, " + errMsg)
	}
	return errcode.Unauthorized
}

// -------------------------------------------------------------------------------------------

// Auth authorization middleware, support custom extra verify.
func Auth(opts ...AuthOption) gin.HandlerFunc {
	o := defaultAuthOptions()
	o.apply(opts...)

	return func(c *gin.Context) {
		authorization := c.GetHeader(HeaderAuthorizationKey)
		if len(authorization) < 100 {
			response.Out(c, responseUnauthorized(o.isReturnErrReason, "token is illegal"))
			c.Abort()
			return
		}

		tokenString := authorization[7:] // remove Bearer prefix

		claims, err := jwt.ValidateToken(tokenString, jwt.WithValidateTokenSignKey(o.signKey))
		if err != nil {
			response.Out(c, responseUnauthorized(o.isReturnErrReason, err.Error()))
			c.Abort()
			return
		}
		// extra verify function
		if o.extraVerifyFn != nil {
			if err = o.extraVerifyFn(claims, c); err != nil {
				response.Out(c, responseUnauthorized(o.isReturnErrReason, err.Error()))
				c.Abort()
				return
			}
		}
		c.Set("claims", claims)
		c.Next()
	}
}

// GetClaims get jwt claims from gin context.
func GetClaims(c *gin.Context) (*jwt.Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*jwt.Claims)
	return jwtClaims, ok
}
