module github.com/containerd/otelttrpc/example

go 1.21

require (
	github.com/containerd/otelttrpc v0.0.0-20240305015340-ea5083fda723
	github.com/containerd/ttrpc v1.2.7
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.29.0
	go.opentelemetry.io/otel/sdk v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	google.golang.org/protobuf v1.36.2
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230731190214-cbb8c96f2d6d // indirect
	google.golang.org/grpc v1.57.1 // indirect
)

replace github.com/containerd/otelttrpc => ../
