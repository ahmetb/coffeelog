# Google Cloud Trace intercept for gRPC

Pass google trace context in remote procedure calls. This allows parent-child tracing across multiple services.

> Note: There is work in progress on https://github.com/GoogleCloudPlatform/google-cloud-go/issues/548 which will make this repo obsolete.

## Client

Use the `intercept.EnableGRPCTracingDialOption` option to add google trace context to outgoing RPC calls
made by the gRPC client.

```go
import "github.com/harlow/grpc-google-cloud-trace/intercept"

func main() {
	// add tracing option to dial
	conn, err := grpc.Dial(
		address,
		intercept.EnableGRPCTracingDialOption,
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// ...
}
```

## Server

Use the `intercept.EnableGRPCTracingServerOption` function to parse the google cloud context from the request
metadata. The interceptor will set up a new child span of the requesting party.

```go
import "github.com/harlow/grpc-google-cloud-trace/intercept"

func main() {
	// ...

	grpcServer := grpc.NewServer(
	  intercept.EnableGRPCTracingServerOption(traceClient),
  	)
 	pb.RegisterRouteGuideServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
```

Create new child spans from the request context:

```go
func getNearbyPoints(ctx context.Context, lat, lon float64) []geo.Point {
	span := trace.FromContext(ctx).NewChild("getNearbyPoints")
	defer span.Finish()

	// ...
}
```

## Credits

This codebase was heavily inspired by the following issues and repositories:

* [Google cloud dial option](https://github.com/GoogleCloudPlatform/google-cloud-go/blob/master/trace/trace.go#L242-L265)
* [OpenTracing support for gRPC in Go](https://github.com/grpc-ecosystem/grpc-opentracing/tree/master/go/otgrpc)
* [Support client side interceptor](https://github.com/grpc/grpc-go/pull/867)
