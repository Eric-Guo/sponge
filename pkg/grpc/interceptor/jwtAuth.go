package interceptor

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Eric-Guo/sponge/pkg/jwt"
)

// ---------------------------------- client ----------------------------------

// SetJwtTokenToCtx set the token (excluding prefix Bearer) to the context in grpc client side
// Example:
//
// authorization := "Bearer jwt-token"
//
//	ctx := SetJwtTokenToCtx(ctx, jwt-token)
//	cli.GetByID(ctx, req)
func SetJwtTokenToCtx(ctx context.Context, token string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md.Set(headerAuthorize, authScheme+" "+token)
	} else {
		md = metadata.Pairs(headerAuthorize, authScheme+" "+token)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// SetAuthToCtx set the authorization (including prefix Bearer) to the context in grpc client side
// Example:
//
//	ctx := SetAuthToCtx(ctx, authorization)
//	cli.GetByID(ctx, req)
func SetAuthToCtx(ctx context.Context, authorization string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md.Set(headerAuthorize, authorization)
	} else {
		md = metadata.Pairs(headerAuthorize, authorization)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// ---------------------------------- server interceptor ----------------------------------

var (
	headerAuthorize = "authorization"

	// auth Scheme
	authScheme = "Bearer"

	// authentication information in ctx key name
	authCtxClaimsName = "tokenInfo"

	// collection of skip authentication methods
	authIgnoreMethods = map[string]struct{}{}
)

// GetAuthorization combining tokens into authentication information
func GetAuthorization(token string) string {
	return authScheme + " " + token
}

// GetAuthCtxKey get the name of Claims
func GetAuthCtxKey() string {
	return authCtxClaimsName
}

// ExtraVerifyFn extra verify function
type ExtraVerifyFn = func(ctx context.Context, claims *jwt.Claims) error

// AuthOption setting the Authentication Field
type AuthOption func(*authOptions)

// authOptions settings
type authOptions struct {
	authScheme    string
	ctxClaimsName string
	ignoreMethods map[string]struct{}

	signKey       []byte // sign key for jwt
	extraVerifyFn ExtraVerifyFn
}

func defaultAuthOptions() *authOptions {
	return &authOptions{
		authScheme:    authScheme,
		ctxClaimsName: authCtxClaimsName,
		ignoreMethods: make(map[string]struct{}), // ways to ignore forensics
	}
}

func (o *authOptions) apply(opts ...AuthOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithAuthScheme set the message prefix for authentication
func WithAuthScheme(scheme string) AuthOption {
	return func(o *authOptions) {
		o.authScheme = scheme
	}
}

// WithAuthClaimsName set the key name of the information in ctx for authentication
func WithAuthClaimsName(claimsName string) AuthOption {
	return func(o *authOptions) {
		o.ctxClaimsName = claimsName
	}
}

// WithAuthIgnoreMethods ways to ignore forensics
// fullMethodName format: /packageName.serviceName/methodName,
// example /api.userExample.v1.userExampleService/GetByID
func WithAuthIgnoreMethods(fullMethodNames ...string) AuthOption {
	return func(o *authOptions) {
		for _, method := range fullMethodNames {
			o.ignoreMethods[method] = struct{}{}
		}
	}
}

// WithSignKey set jwt sign key
func WithSignKey(key []byte) AuthOption {
	return func(o *authOptions) {
		o.signKey = key
	}
}

// WithExtraVerify set extra verify function
func WithExtraVerify(fn ExtraVerifyFn) AuthOption {
	return func(o *authOptions) {
		o.extraVerifyFn = fn
	}
}

// -------------------------------------------------------------------------------------------

// verify authorization from context, support default and custom verify processing
func jwtVerify(ctx context.Context, opt *authOptions) (context.Context, error) {
	if opt == nil {
		opt = defaultAuthOptions()
	}

	tokenString, err := grpc_auth.AuthFromMD(ctx, authScheme) // key is authScheme
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "AuthFromMD error: %v", err)
	}

	if len(tokenString) <= 100 {
		return ctx, status.Errorf(codes.Unauthenticated, "token is illegal")
	}

	claims, err := jwt.ValidateToken(tokenString, jwt.WithValidateTokenSignKey(opt.signKey))
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "%v", err)
	}
	if opt.extraVerifyFn != nil {
		err = opt.extraVerifyFn(ctx, claims)
		if err != nil {
			return ctx, status.Errorf(codes.Unauthenticated, "extra verification fails: %v", err)
		}
	}
	newCtx := context.WithValue(ctx, authCtxClaimsName, claims) //nolint
	return newCtx, nil
}

// GetJwtClaims get the jwt default claims from context, contains fixed fields uid and name
func GetJwtClaims(ctx context.Context) (*jwt.Claims, bool) {
	v, ok := ctx.Value(authCtxClaimsName).(*jwt.Claims)
	return v, ok
}

// UnaryServerJwtAuth jwt unary interceptor
func UnaryServerJwtAuth(opts ...AuthOption) grpc.UnaryServerInterceptor {
	o := defaultAuthOptions()
	o.apply(opts...)
	authScheme = o.authScheme
	authCtxClaimsName = o.ctxClaimsName
	authIgnoreMethods = o.ignoreMethods

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var newCtx context.Context
		var err error

		if _, ok := authIgnoreMethods[info.FullMethod]; ok {
			newCtx = ctx
		} else {
			newCtx, err = jwtVerify(ctx, o)
			if err != nil {
				return nil, err
			}
		}

		return handler(newCtx, req)
	}
}

// StreamServerJwtAuth jwt stream interceptor
func StreamServerJwtAuth(opts ...AuthOption) grpc.StreamServerInterceptor {
	o := defaultAuthOptions()
	o.apply(opts...)
	authScheme = o.authScheme
	authCtxClaimsName = o.ctxClaimsName
	authIgnoreMethods = o.ignoreMethods

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error

		if _, ok := authIgnoreMethods[info.FullMethod]; ok {
			newCtx = stream.Context()
		} else {
			newCtx, err = jwtVerify(stream.Context(), o)
			if err != nil {
				return err
			}
		}

		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}
