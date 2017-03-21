package intercept

import (
	"fmt"
	"strings"

	"cloud.google.com/go/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const key = "google_cloud_trace_header"

// EnableGRPCTracingDialOption enables tracing of requests that are sent over a gRPC connection.
// Modified version of: https://github.com/GoogleCloudPlatform/google-cloud-go/blob/master/trace/trace.go#L242-L265
var EnableGRPCTracingDialOption = grpc.WithUnaryInterceptor(grpc.UnaryClientInterceptor(clientInterceptor))

func clientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// trace request w/ child span
	span := trace.FromContext(ctx).NewChild(fmt.Sprintf("/grpc.Client%s", method))
	defer span.Finish()

	// new metadata, or copy of existing
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	// append google trace header to metadata
	// https://cloud.google.com/trace/docs/faq
	md[key] = append(md[key], fmt.Sprintf("%s/%d;o=1", span.TraceID(), 0))
	ctx = metadata.NewContext(ctx, md)

	return invoker(ctx, method, req, reply, cc, opts...)
}

// EnableGRPCTracingServerOption enables parsing google trace header from metadata
// and adds a new child span to the context.
func EnableGRPCTracingServerOption(traceClient *trace.Client) grpc.ServerOption {
	return grpc.UnaryInterceptor(serverInterceptor(traceClient))
}

func serverInterceptor(traceClient *trace.Client) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// fetch metadata from request context
		md, ok := metadata.FromContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		header := strings.Join(md[key], "")

		// create new child span from google trace header, add to
		// current request context
		span := traceClient.SpanFromHeader(info.FullMethod, header)
		defer span.Finish()
		ctx = trace.NewContext(ctx, span)

		return handler(ctx, req)
	}
}
