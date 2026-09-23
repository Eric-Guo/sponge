package interceptor

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	zapLog "github.com/Eric-Guo/sponge/pkg/logger"
)

var contentMark = []byte(" ...... ")

// ---------------------------------- client interceptor ----------------------------------

// UnaryClientLog client log unary interceptor
func UnaryClientLog(logger *zap.Logger, opts ...LogOption) grpc.UnaryClientInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// ignore printing of the specified method
		if ignoreLogMethods[method] {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		startTime := time.Now()

		var reqIDField zap.Field
		if requestID := ClientCtxRequestID(ctx); requestID != "" {
			reqIDField = zap.String(ContextRequestIDKey, requestID)
		} else {
			reqIDField = zap.Skip()
		}

		err := invoker(ctx, method, req, reply, cc, opts...)

		statusCode := status.Code(err)
		fields := []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "unary"),
			zap.String("method", method),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
			reqIDField,
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error("invoker error", fields...)
		} else {
			logger.Info("invoker result", fields...)
		}

		return err
	}
}

// StreamClientLog client log stream interceptor
func StreamClientLog(logger *zap.Logger, opts ...LogOption) grpc.StreamClientInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
		streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// ignore printing of the specified method
		if ignoreLogMethods[method] {
			return streamer(ctx, desc, cc, method, opts...)
		}

		startTime := time.Now()

		var reqIDField zap.Field
		if requestID := ClientCtxRequestID(ctx); requestID != "" {
			reqIDField = zap.String(ContextRequestIDKey, requestID)
		} else {
			reqIDField = zap.Skip()
		}

		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		statusCode := status.Code(err)
		fields := []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "stream"),
			zap.String("method", method),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
			reqIDField,
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error("streamer invoker error", fields...)
		} else {
			logger.Info("streamer invoker result", fields...)
		}

		return clientStream, err
	}
}

// ---------------------------------- server interceptor ----------------------------------

var defaultMaxLength = 300 // max length of response data to print
var defaultMarshalFn = func(reply interface{}) []byte {
	data, _ := json.Marshal(reply)
	return data
}
var ignoreLogMethods = map[string]bool{ // ignore printing methods
	"/grpc.health.v1.Health/Check": true,
}
var printErrorBySpecifiedCodes = map[codes.Code]bool{
	codes.Internal:                           true,
	codes.Unavailable:                        true,
	errcode.StatusInternalServerError.Code(): true,
	errcode.StatusServiceUnavailable.Code():  true,
}

// LogOption log settings
type LogOption func(*logOptions)

type logOptions struct {
	maxLength           int
	isReplaceGRPCLogger bool
	marshalFn           func(reply interface{}) []byte // default json.Marshal
}

func defaultLogOptions() *logOptions {
	return &logOptions{
		maxLength: defaultMaxLength,
		marshalFn: defaultMarshalFn,
	}
}

func (o *logOptions) apply(opts ...LogOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithMaxLen logger content max length
func WithMaxLen(maxLen int) LogOption {
	return func(o *logOptions) {
		if maxLen > 0 {
			o.maxLength = maxLen
		}
	}
}

// WithReplaceGRPCLogger replace grpc logger v2
func WithReplaceGRPCLogger() LogOption {
	return func(o *logOptions) {
		o.isReplaceGRPCLogger = true
	}
}

// WithPrintErrorByCodes set print error by grpc codes
func WithPrintErrorByCodes(code ...codes.Code) LogOption {
	return func(o *logOptions) {
		for _, c := range code {
			printErrorBySpecifiedCodes[c] = true
		}
	}
}

// WithMarshalFn custom response data marshal function
func WithMarshalFn(fn func(reply interface{}) []byte) LogOption {
	return func(o *logOptions) {
		if fn != nil {
			o.marshalFn = fn
		}
	}
}

// WithLogIgnoreMethods ignore printing methods
// fullMethodName format: /packageName.serviceName/methodName,
// example /api.userExample.v1.userExampleService/GetByID
func WithLogIgnoreMethods(fullMethodNames ...string) LogOption {
	return func(o *logOptions) {
		for _, method := range fullMethodNames {
			ignoreLogMethods[method] = true
		}
	}
}

// UnaryServerLog server-side log unary interceptor
func UnaryServerLog(logger *zap.Logger, opts ...LogOption) grpc.UnaryServerInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)

	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// ignore printing of the specified method
		if ignoreLogMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		startTime := time.Now()
		requestID := ServerCtxRequestID(ctx)

		fields := []zap.Field{
			zap.String("type", "unary"),
			zap.String("method", info.FullMethod),
			zap.Any("request", req),
		}
		if requestID != "" {
			fields = append(fields, zap.String(ContextRequestIDKey, requestID))
		}
		logger.Info("<<<<", fields...)

		resp, err := handler(ctx, req)

		data := o.marshalFn(resp)
		if len(data) > o.maxLength {
			data = append(data[:o.maxLength], contentMark...)
		}

		statusCode := status.Code(err)
		fields = []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "unary"),
			zap.String("method", info.FullMethod),
			zap.ByteString("data", data),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
		}
		if requestID != "" {
			fields = append(fields, zap.String(ContextRequestIDKey, requestID))
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error(">>>>", fields...)
		} else {
			logger.Info(">>>>", fields...)
		}

		return resp, err
	}
}

// UnaryServerSimpleLog server-side log unary interceptor, only print response
func UnaryServerSimpleLog(logger *zap.Logger, opts ...LogOption) grpc.UnaryServerInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)

	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// ignore printing of the specified method
		if ignoreLogMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		startTime := time.Now()
		requestID := ServerCtxRequestID(ctx)

		resp, err := handler(ctx, req)

		requestIDField := zap.Skip()
		if requestID != "" {
			requestIDField = zap.String(ContextRequestIDKey, requestID)
		}
		statusCode := status.Code(err)
		fields := []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "unary"),
			zap.String("method", info.FullMethod),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
			requestIDField,
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error("gRPC response error", fields...)
		} else {
			logger.Info("gRPC response", fields...)
		}

		return resp, err
	}
}

// StreamServerLog Server-side log stream interceptor
func StreamServerLog(logger *zap.Logger, opts ...LogOption) grpc.StreamServerInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)

	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// ignore printing of the specified method
		if ignoreLogMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		startTime := time.Now()
		requestID := ServerCtxRequestID(stream.Context())

		requestIDField := zap.Skip()
		if requestID != "" {
			requestIDField = zap.String(ContextRequestIDKey, requestID)
		}
		fields := []zap.Field{
			zap.String("type", "stream"),
			zap.String("method", info.FullMethod),
			requestIDField,
		}
		logger.Info("<<<<", fields...)

		err := handler(srv, stream)

		statusCode := status.Code(err)
		fields = []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "stream"),
			zap.String("method", info.FullMethod),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
			requestIDField,
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error(">>>>", fields...)
		} else {
			logger.Info(">>>>", fields...)
		}

		return err
	}
}

// StreamServerSimpleLog Server-side log stream interceptor, only print response
func StreamServerSimpleLog(logger *zap.Logger, opts ...LogOption) grpc.StreamServerInterceptor {
	o := defaultLogOptions()
	o.apply(opts...)

	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if o.isReplaceGRPCLogger {
		zapLog.ReplaceGRPCLoggerV2(logger)
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// ignore printing of the specified method
		if ignoreLogMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		startTime := time.Now()
		requestID := ServerCtxRequestID(stream.Context())

		requestIDField := zap.Skip()
		if requestID != "" {
			requestIDField = zap.String(ContextRequestIDKey, requestID)
		}

		err := handler(srv, stream)

		statusCode := status.Code(err)
		fields := []zap.Field{
			zap.String("code", statusCode.String()),
			zap.Error(err),
			zap.String("type", "stream"),
			zap.String("method", info.FullMethod),
			zap.Int64("time_us", time.Since(startTime).Microseconds()),
			requestIDField,
		}
		if err != nil && printErrorBySpecifiedCodes[statusCode] {
			logger.WithOptions(zap.AddStacktrace(zapcore.PanicLevel)).Error("gRPC response error", fields...)
		} else {
			logger.Info("gRPC response", fields...)
		}

		return err
	}
}
