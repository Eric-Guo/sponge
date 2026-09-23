package interceptor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"

	"github.com/Eric-Guo/sponge/pkg/jwt"
)

var (
	uid    = "100"
	fields = map[string]interface{}{
		"name":   "bob",
		"age":    10,
		"is_vip": true,
	}
	jwtSignKey = []byte("your-secret-key")
)

func extraVerifyFn(ctx context.Context, claims *jwt.Claims) error {
	// judge whether the user is disabled, query whether jwt id exists from the blacklist
	//if CheckBlackList(uid, claims.ID) {
	//	return errors.New("user is disabled")
	//}

	// check fields
	if claims.UID != uid {
		return fmt.Errorf("uid not match, expect %s, got %s", uid, claims.UID)
	}
	if name, _ := claims.GetString("name"); name != fields["name"] {
		return fmt.Errorf("name not match, expect %s, got %s", fields["name"], name)
	}
	if age, _ := claims.GetInt("age"); age != fields["age"] {
		return fmt.Errorf("age not match, expect %d, got %d", fields["age"], age)
	}
	if isVip, _ := claims.GetBool("is_vip"); isVip != fields["is_vip"] {
		return fmt.Errorf("is_vip not match, expect %v, got %v", fields["is_vip"], isVip)
	}

	return nil
}

func TestJwtAuth_Unary(t *testing.T) {
	t.Run("default jwt_sign_key", func(t *testing.T) {
		// run grpc server
		addr := newUnaryRPCServer(
			UnaryServerJwtAuth(),
		)
		time.Sleep(time.Millisecond * 200)

		// run grpc client and call rpc method
		cli := newUnaryRPCClient(addr)
		ctx := context.Background()
		_, token, _ := jwt.GenerateToken(uid)
		ctx = SetJwtTokenToCtx(ctx, token)

		err := sayHelloMethod(ctx, cli)
		assert.NoError(t, err)
	})

	t.Run("custom jwt_sign_key and claims", func(t *testing.T) {
		// run grpc server
		addr := newUnaryRPCServer(
			UnaryServerJwtAuth(
				WithSignKey(jwtSignKey),
				WithExtraVerify(extraVerifyFn),
				WithAuthIgnoreMethods(
					"/api.user.v1.User/Register",
					"/api.user.v1.User/Login",
				),
			),
		)
		time.Sleep(time.Millisecond * 200)

		// run grpc client and call rpc method
		cli := newUnaryRPCClient(addr)
		ctx := context.Background()
		_, token, _ := jwt.GenerateToken(
			uid,
			jwt.WithGenerateTokenSignKey(jwtSignKey),
			jwt.WithGenerateTokenSignMethod(jwt.HS384),
			jwt.WithGenerateTokenFields(fields),
			jwt.WithGenerateTokenClaims([]jwt.RegisteredClaimsOption{
				jwt.WithExpires(time.Hour * 12),
				//jwt.WithIssuedAt(now),
				// jwt.WithSubject("123"),
				// jwt.WithIssuer("https://auth.example.com"),
				// jwt.WithAudience("https://api.example.com"),
				// jwt.WithNotBefore(now),
				// jwt.WithJwtID("abc1234xxx"),
			}...),
		)
		ctx = SetJwtTokenToCtx(ctx, token)

		err := sayHelloMethod(ctx, cli)
		assert.NoError(t, err)
	})
}

func TestJwtAuth_Stream(t *testing.T) {
	t.Run("default jwt_sign_key", func(t *testing.T) {
		// run grpc server
		addr := newStreamRPCServer(
			StreamServerJwtAuth(),
		)
		time.Sleep(time.Millisecond * 200)

		// run grpc client and call rpc method
		cli := newStreamRPCClient(addr)
		ctx := context.Background()
		_, token, _ := jwt.GenerateToken(uid)
		ctx = SetJwtTokenToCtx(ctx, token)

		err := discussHelloMethod(ctx, cli)
		assert.NoError(t, err)
	})

	t.Run("custom jwt_sign_key and claims", func(t *testing.T) {
		// run grpc server
		addr := newStreamRPCServer(
			StreamServerJwtAuth(
				WithSignKey(jwtSignKey),
				WithExtraVerify(extraVerifyFn),
			),
		)
		time.Sleep(time.Millisecond * 200)

		// run grpc client and call rpc method
		cli := newStreamRPCClient(addr)
		ctx := context.Background()
		_, token, _ := jwt.GenerateToken(
			uid,
			jwt.WithGenerateTokenSignKey(jwtSignKey),
			jwt.WithGenerateTokenSignMethod(jwt.HS384),
			jwt.WithGenerateTokenFields(fields),
			jwt.WithGenerateTokenClaims([]jwt.RegisteredClaimsOption{
				jwt.WithExpires(time.Hour * 12),
				//jwt.WithIssuedAt(now),
				// jwt.WithSubject("123"),
				// jwt.WithIssuer("https://auth.example.com"),
				// jwt.WithAudience("https://api.example.com"),
				// jwt.WithNotBefore(now),
				// jwt.WithJwtID("abc1234xxx"),
			}...),
		)
		ctx = SetJwtTokenToCtx(ctx, token)

		err := discussHelloMethod(ctx, cli)
		assert.NoError(t, err)
	})
}

func TestJwtVerifyError(t *testing.T) {
	// token illegal
	ctx := context.Background()
	authorization := []string{GetAuthorization("error token......")}
	ctx = metadata.NewIncomingContext(ctx, metadata.MD{headerAuthorize: authorization})
	_, err := jwtVerify(ctx, nil)
	assert.Error(t, err)

	// authorization format error, missing Bearer
	ctx = context.WithValue(context.Background(), headerAuthorize, authorization)
	_, err = jwtVerify(ctx, nil)
	assert.Error(t, err)
}

func TestGetAuthCtxKey(t *testing.T) {
	key := GetAuthCtxKey()
	assert.Equal(t, authCtxClaimsName, key)
}

func TestGetAuthorization(t *testing.T) {
	testData := "token"
	authorization := GetAuthorization(testData)
	assert.Equal(t, authScheme+" "+testData, authorization)
}

func TestAuthOptions(t *testing.T) {
	o := defaultAuthOptions()

	o.apply(WithAuthScheme(authScheme))
	assert.Equal(t, authScheme, o.authScheme)

	o.apply(WithAuthClaimsName(authCtxClaimsName))
	assert.Equal(t, authCtxClaimsName, o.ctxClaimsName)

	o.apply(WithAuthIgnoreMethods("/metrics"))
	assert.Equal(t, struct{}{}, o.ignoreMethods["/metrics"])

	o.apply(WithSignKey(jwtSignKey))
	assert.Equal(t, jwtSignKey, o.signKey)
	o.apply(WithExtraVerify(extraVerifyFn))
	assert.NotNil(t, o.extraVerifyFn)
}

func TestSetJWTTokenToCtx(t *testing.T) {
	ctx := context.Background()
	_, token, _ := jwt.GenerateToken(uid)
	expected := []string{GetAuthorization(token)}

	ctx = SetJwtTokenToCtx(ctx, token)
	md, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(t, expected, md.Get(headerAuthorize))
}

func TestSetAuthToCtx(t *testing.T) {
	ctx := context.Background()
	_, token, _ := jwt.GenerateToken(uid)
	authorization := GetAuthorization(token)
	expected := []string{authorization}

	ctx = SetAuthToCtx(ctx, authorization)
	md, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(t, expected, md.Get(headerAuthorize))
}
