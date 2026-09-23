## jwt

`jwt` is a library for generating and parsing token based on [jwt](https://github.com/golang-jwt/jwt).

encapsulated functions:

- Support custom fields.
- Support token refresh.
- Support generating token pairs (refresh token and access token).

<br>

## Example of use

### One Token

```go
package main

import (
    "github.com/Eric-Guo/sponge/pkg/jwt"
    "time"
)

func main() {
    uid := "123"

    // Case 1: default, signKey, signMethod(HS256), expiry time(24 hour)
    {
        // generate token
        jwtID, token, err := jwt.GenerateToken(uid)

        // validate token, get claims
        claims, err := jwt.ValidateToken(token)

        // refresh token
        //jwtID, newToken, err := jwt.RefreshToken(token)
    }

    // Case 2: custom signMethod, signKey, expiry time, fields, claims
    {
        now := time.Now()
        signMethod := jwt.HS384
        signKey := "your-secret-key"

        // generate token
        jwtID, token, err := jwt.GenerateToken(
            uid,
            jwt.WithGenerateTokenSignMethod(signMethod),
            jwt.WithGenerateTokenSignKey(signKey),
            jwt.WithGenerateTokenFields(map[string]interface{}{
                "name": "john",
                "role": "admin",
            }),
            jwt.WithGenerateTokenClaims([]jwt.RegisteredClaimsOption{
                jwt.WithExpires(time.Hour * 12),
                jwt.WithIssuedAt(now),
                // jwt.WithSubject("123"),
                // jwt.WithIssuer("https://auth.example.com"),
                // jwt.WithAudience("https://api.example.com"),
                // jwt.WithNotBefore(now),
                // jwt.WithJwtID("abc1234xxx"),
            }...),
        )

        // validate token, get claims
        claims, err := jwt.ValidateToken(token)

        // refresh token
        //jwtID, newToken, err := jwt.RefreshToken(
        //    token,
        //    jwt.WithRefreshTokenSignKey(signKey),
        //    jwt.WithRefreshTokenExpire(time.Hour*12),
        //)
    }
}
```

> Tip: jwtID is used to prevent replay attacks. If you need to kick the user offline, you can add it to the blacklist and reject it directly next time you request it.

<br>

### Two Tokens

```go
package main

import (
    "github.com/Eric-Guo/sponge/pkg/jwt"
    "time"
)

func main() {
    uid := "123"

    // Case 1: default, signKey, signMethod(HS256), expiry time(24 hour)
    {
        // generate token
        tokens, err := jwt.GenerateTwoTokens(uid)

        // validate token, get claims
        claims, err := jwt.ValidateToken(tokens.AccessToken)

        // refresh token, get new access token, if refresh token is expired time is less than 3 hours, will refresh token too.
        //newAccessTokens, err := jwt.RefreshTwoTokens(tokens.RefreshToken, tokens.AccessToken)
    }

    // Case 2: custom signMethod, signKey, expiry time, fields, claims
    {
        now := time.Now()
        signMethod := jwt.HS384
        signKey := "your-secret-key"

        // generate token
        tokens, err := jwt.GenerateTwoTokens(
            uid,
            jwt.WithGenerateTwoTokensSignMethod(signMethod),
            jwt.WithGenerateTwoTokensSignKey(signKey),
            jwt.WithGenerateTwoTokensFields(map[string]interface{}{
                "name": "john",
                "role": "admin",
            }),
            jwt.WithGenerateTwoTokensRefreshTokenClaims([]jwt.RegisteredClaimsOption{
                jwt.WithExpires(time.Hour * 24 * 15),
                jwt.WithIssuedAt(now),
                // jwt.WithSubject("123"),
                // jwt.WithIssuer("https://auth.example.com"),
                // jwt.WithAudience("https://api.example.com"),
                // jwt.WithNotBefore(now),
                // jwt.WithJwtID("abc1234xxx"),
            }...),
            jwt.WithGenerateTwoTokensAccessTokenClaims([]jwt.RegisteredClaimsOption{
                jwt.WithExpires(time.Minute * 15),
                jwt.WithIssuedAt(now),
                // jwt.WithSubject("123"),
                // jwt.WithIssuer("https://auth.example.com"),
                // jwt.WithAudience("https://api.example.com"),
                // jwt.WithNotBefore(now),
                // jwt.WithJwtID("abc1234xxx"),
            }...),
        )

        // validate token, get claims
        claims, err := jwt.ValidateToken(tokens.AccessToken)

        // refresh token
        newTokens, err := jwt.RefreshTwoTokens(
            tokens.RefreshToken,
            tokens.AccessToken,
            jwt.WithRefreshTwoTokensSignKey(signKey),
            jwt.WithRefreshTwoTokensRefreshTokenExpires(time.Hour*24*15),
            jwt.WithRefreshTwoTokensAccessTokenExpires(time.Minute*15),
        )
    }
}
```

---

> **Note**: If you used sponge<=v1.12.8 and referenced this library in your project code, 
> update to the latest version and cause compilation errors, replace the batch import path
> `github.com/Eric-Guo/sponge/pkg/jwt` with `github.com/Eric-Guo/sponge/pkg/jwt/old_jwt`.
> `old_jwt` will remove in the future.