## jwt

> Note: This package is deprecated, please use `github.com/Eric-Guo/sponge/pkg/jwt` instead.

`jwt` is a library for generating and parsing token based on [jwt](https://github.com/golang-jwt/jwt).

<br>

## Example of use

### Default jwt

```go
    import "github.com/Eric-Guo/sponge/pkg/jwt/old_jwt"

    jwt.Init(
        // jwt.WithSigningKey("123456"),   // key
        // jwt.WithExpire(time.Hour), // expiry time
        // jwt.WithSigningMethod(jwt.HS512), // encryption method, default is HS256, can be set to HS384, HS512
    )

    uid := "123"
    name := "admin"

    // generate token
    token, err := jwt.GenerateToken(uid, name)
    // handle err

    // parse token
    claims, err := jwt.ParseToken(token)
    // handle err

    // verify
    if claims.Uid != uid || claims.Name != name {
        print("verify failed")
        return
    }
```

<br>

### Custom jwt

```go
    import "github.com/Eric-Guo/sponge/pkg/jwt/old_jwt"

    jwt.Init(
        // jwt.WithSigningKey("123456"),   // key
        // jwt.WithExpire(time.Hour), // expiry time
        // jwt.WithSigningMethod(jwt.HS512), // encryption method, default is HS256, can be set to HS384, HS512
    )

    fields := map[string]interface{}{"id": 123, "foo": "bar"}

    // generate token
    token, err := jwt.GenerateCustomToken(fields)
    // handle err

    // parse token
    claims, err := jwt.ParseCustomToken(token)
    // handle err

    // verify
    id, isExist1 := claims.GetInt("id")
    if !isExist1 || id != fields["id"].(int) {
        print("verify failed")
    }
    foo, isExist2 := claims.GetString("foo")
    if !isExist1 || foo != fields["foo"].(string) {
        print("verify failed")
        return
    }
```
