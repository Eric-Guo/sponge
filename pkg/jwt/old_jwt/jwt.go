// Package jwt is deprecated, old package path is "github.com/Eric-Guo/sponge/pkg/jwt/old_jwt"
// Please use new jwt package instead, new package path is "github.com/Eric-Guo/sponge/pkg/jwt"
package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrTokenExpired expired
var ErrTokenExpired = jwt.ErrTokenExpired

var opt *options

// Init initialize jwt
// Deprecated: build jwt.Init() before use
func Init(opts ...Option) {
	o := defaultOptions()
	o.apply(opts...)
	opt = o
}

// Claims standard claims, include uid, name, and RegisteredClaims
type Claims struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateToken generate token by uid and name, use universal Claims
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" GenerateToken instead
func GenerateToken(uid string, name ...string) (string, error) {
	if opt == nil {
		return "", errInit
	}

	nameVal := ""
	if len(name) > 0 {
		nameVal = name[0]
	}
	claims := Claims{
		uid,
		nameVal,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(opt.expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    opt.issuer,
		},
	}

	token := jwt.NewWithClaims(opt.signingMethod, claims)
	return token.SignedString(opt.signingKey)
}

// ParseToken parse token, return universal Claims
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" ValidateToken instead
func ParseToken(tokenString string) (*Claims, error) {
	if opt == nil {
		return nil, errInit
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return opt.signingKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errSignature
}

// RefreshToken refresh token
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" RefreshToken instead
func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(opt.expire))
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(opt.signingMethod, claims)
	return token.SignedString(opt.signingKey)
}

// -------------------------------------------------------------------------------------------

// KV map type
type KV = map[string]interface{}

// CustomClaims custom fields claims
type CustomClaims struct {
	Fields KV `json:"fields"`
	jwt.RegisteredClaims
}

// Get custom field value by key, if not found, return false
func (c *CustomClaims) Get(key string) (val interface{}, isExist bool) {
	if c.Fields == nil {
		return nil, false
	}
	val, isExist = c.Fields[key]
	return val, isExist
}

// GetString custom field value by key, if not found, return false
func (c *CustomClaims) GetString(key string) (string, bool) {
	val, isExist := c.Get(key)
	if isExist {
		str, ok := val.(string)
		return str, ok
	}
	return "", false
}

// GetInt custom field value by key, if not found, return false
func (c *CustomClaims) GetInt(key string) (int, bool) {
	val, isExist := c.Get(key)
	if isExist {
		if v, ok := val.(float64); ok {
			return int(v), true
		}
		if v, ok := val.(int); ok {
			return v, true
		}
	}
	return 0, false
}

// GetUint64 custom field value by key, if not found, return false
func (c *CustomClaims) GetUint64(key string) (uint64, bool) {
	val, isExist := c.Get(key)
	if isExist {
		if v, ok := val.(float64); ok {
			return uint64(v), true
		}
		if v, ok := val.(uint64); ok {
			return v, true
		}
	}
	return 0, false
}

// GenerateCustomToken generate token by custom fields, use CustomClaims
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" GenerateToken instead
func GenerateCustomToken(kv map[string]interface{}) (string, error) {
	if opt == nil {
		return "", errInit
	}

	claims := CustomClaims{
		kv,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(opt.expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    opt.issuer,
		},
	}

	token := jwt.NewWithClaims(opt.signingMethod, claims)
	return token.SignedString(opt.signingKey)
}

// ParseCustomToken parse token, return CustomClaims
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" ValidateToken instead
func ParseCustomToken(tokenString string) (*CustomClaims, error) {
	if opt == nil {
		return nil, errInit
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return opt.signingKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errSignature
}

// RefreshCustomToken refresh custom token
// Deprecated: use "github.com/Eric-Guo/sponge/pkg/jwt" RefreshToken instead
func RefreshCustomToken(tokenString string) (string, error) {
	claims, err := ParseCustomToken(tokenString)
	if err != nil {
		return "", err
	}
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(opt.expire))
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(opt.signingMethod, claims)
	return token.SignedString(opt.signingKey)
}
