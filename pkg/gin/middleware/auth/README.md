## auth

`auth` middleware for gin framework.

### Example of use

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/go-dev-frame/sponge/pkg/gin/middleware/auth"
    "github.com/go-dev-frame/sponge/pkg/gin/response"
)

func main() {
    r := gin.Default()

    // initialize jwt first
    auth.InitAuth([]byte("your-sign-key"), time.Hour*24) // default signing method is HS256
    // auth.InitAuth(
    //     []byte("your-sign-key"),
    //     time.Minute*24,
    //     auth.WithInitAuthSigningMethod(auth.HS512),
    //     auth.WithInitAuthIssuer("foobar.com"),
    // )

    r.POST("/auth/login", Login)

    g := r.Group("/api/v1")
    g.Use(auth.Auth())
    //g.Use(auth.Auth(auth.WithExtraVerify(extraVerifyFn))) // add extra verify function

    g.GET("/user/:id", GetByID)
    //g.PUT("/user/:id", Create)
    //g.DELETE("/user/:id", DeleteByID)

    r.Run(":8080")
}

func Login(c *gin.Context) {
    // Get uid from User.id or other place
    uid := "100"

    // Case 1: only uid for token
    {
        token, err := auth.GenerateToken(uid)
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
    }

    // Case 2: uid and custom fields for token
    {
        fields := map[string]interface{}{
            "name":   "bob",
            "age":    10,
            "is_vip": true,
        }
        token, err = auth.GenerateToken(uid, auth.WithGenerateTokenFields(fields))
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
    }

    response.Success(c, token)
}

func GetByID(c *gin.Context) {
    uid := c.Param("id")

    // if necessary, claims can be got from gin context
    claims, ok := auth.GetClaims(c)
    //uid := claims.UID
    //name, _ := claims.GetString("name")
    //age, _ := claims.GetInt("age")
    //isVip, _ := claims.GetBool("is_vip")

    response.Success(c, gin.H{"id": uid})
}

func extraVerifyFn(claims *auth.Claims, c *gin.Context) error {
    // check if token is about to expire (less than 10 minutes remaining)
    if time.Now().Unix()-claims.ExpiresAt.Unix() < int64(time.Minute*10) {
        token, err := auth.RefreshToken(claims)
        if err != nil {
            return err
        }
        c.Header("X-Renewed-Token", token)
    }

    // judge whether the user is disabled, query whether jwt id exists from the blacklist
    //if CheckBlackList(uid, claims.ID) {
    //    return errors.New("user is disabled")
    //}

    return nil
}
```

<br>
