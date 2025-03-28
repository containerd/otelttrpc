/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

/*
   Copyright The OpenTelemetry Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package otelttrpc

import (
	"context"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/containerd/otelttrpc/internal"
	"github.com/containerd/ttrpc"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "testService"

// testingService is our prototype service definition for use in testing the full model.
//
// Typically, this is generated. We define it here to ensure that that package
// primitive has what is required for generated code.
type testingService interface {
	Test(ctx context.Context, req *internal.TestPayload) (*internal.TestPayload, error)
}

type testingClient struct {
	client *ttrpc.Client
}

func newTestingClient(client *ttrpc.Client) *testingClient {
	return &testingClient{
		client: client,
	}
}

func (tc *testingClient) Test(ctx context.Context, req *internal.TestPayload) (*internal.TestPayload, error) {
	var tp internal.TestPayload
	return &tp, tc.client.Call(ctx, serviceName, "Test", req, &tp)
}

// testingServer is what would be implemented by the user of this package.
type testingServer struct{}

func (s *testingServer) Test(ctx context.Context, req *internal.TestPayload) (*internal.TestPayload, error) {
	tp := &internal.TestPayload{Foo: strings.Repeat(req.Foo, 2)}
	if dl, ok := ctx.Deadline(); ok {
		tp.Deadline = dl.UnixNano()
	}

	if v, ok := ttrpc.GetMetadataValue(ctx, "foo"); ok {
		tp.Metadata = v
	}

	return tp, nil
}

func TestClientCallServerConcurrent(t *testing.T) {
	var (
		ctx            = ttrpc.WithMetadata(context.Background(), ttrpc.MD{"test-key": []string{"test-val"}})
		exp, tp        = newTracerProvider()
		server         = mustServer(t)(newServerWithTTRPCInterceptor(tp))
		testImpl       = &testingServer{}
		addr, listener = newTestListener(t)
		payload        = &internal.TestPayload{
			Foo: "bar",
		}
	)

	concurrency := 30
	testClients := make([]*testingClient, 0, concurrency)
	for i := 0; i < concurrency; i++ {
		client, cleanup := newTestClient(t, addr, tp)
		testClients = append(testClients, newTestingClient(client))
		defer cleanup()
	}
	defer listener.Close()
	defer func() { _ = tp.Shutdown(ctx) }()

	registerTestingService(server, testImpl)

	go func() {
		_ = server.Serve(ctx, listener)
	}()
	defer func() {
		_ = server.Shutdown(ctx)
	}()

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	for _, testClient := range testClients {
		wg.Add(1)
		go func(tc *testingClient) {
			defer wg.Done()
			if _, err := tc.Test(ctx, payload); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
		}(testClient)
	}

	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// get exported spans
	snapshots := exp.GetSpans().Snapshots()
	// we should capture `concurrency * 2` spans, one each from client and server side
	// TODO: validate individual spans and their attributes
	assert.Equal(t, concurrency*2, len(snapshots), "Number of spans mismatched")
}

func TestClientCallServer(t *testing.T) {
	var (
		ctx             = ttrpc.WithMetadata(context.Background(), ttrpc.MD{"test-key": []string{"test-val"}})
		exp, tp         = newTracerProvider()
		server          = mustServer(t)(newServerWithTTRPCInterceptor(tp))
		testImpl        = &testingServer{}
		addr, listener  = newTestListener(t)
		client, cleanup = newTestClient(t, addr, tp)
		testClient      = newTestingClient(client)
		payload         = &internal.TestPayload{
			Foo: "bar",
		}
	)
	defer listener.Close()
	defer cleanup()
	defer func() { _ = tp.Shutdown(ctx) }()

	registerTestingService(server, testImpl)

	go func() {
		_ = server.Serve(ctx, listener)
	}()
	defer func() {
		_ = server.Shutdown(ctx)
	}()

	_, err := testClient.Test(ctx, payload)

	if err != nil {
		t.Fatal(err)
	}

	// get exported spans
	snapshots := exp.GetSpans().Snapshots()
	// we should capture 2 spans, one each from client and server side
	// TODO: validate individual spans and their attributes
	assert.Equal(t, 2, len(snapshots), "Number of spans mismatched")
}

func newServerWithTTRPCInterceptor(tp trace.TracerProvider) (*ttrpc.Server, error) {
	serverOpt := ttrpc.WithUnaryServerInterceptor(UnaryServerInterceptor(WithTracerProvider(tp)))
	return ttrpc.NewServer(serverOpt)
}

func mustServer(t testing.TB) func(server *ttrpc.Server, err error) *ttrpc.Server {
	return func(server *ttrpc.Server, err error) *ttrpc.Server {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}

		return server
	}
}

// newTracerProvider creates in memory exporter and tracer provider to be
// used as tracing test
func newTracerProvider() (*tracetest.InMemoryExporter, *sdktrace.TracerProvider) {
	//create in memory exporter
	exp := tracetest.NewInMemoryExporter()

	//create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return exp, tp
}

func newTestListener(t testing.TB) (string, net.Listener) {
	var prefix string

	// Abstracts sockets are only available on Linux.
	if runtime.GOOS == "linux" {
		prefix = "\x00"
	}
	addr := prefix + t.Name()
	listener, err := net.Listen("unix", addr)
	if err != nil {
		t.Fatal(err)
	}

	return addr, listener
}

func newTestClient(t testing.TB, addr string, tp *sdktrace.TracerProvider) (*ttrpc.Client, func()) {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	client := ttrpc.NewClient(conn, ttrpc.WithUnaryClientInterceptor(UnaryClientInterceptor(WithTracerProvider(tp))))
	return client, func() {
		conn.Close()
		client.Close()
	}
}

// registerTestingService mocks more of what is generated code. Unlike grpc, we
// register with a closure so that the descriptor is allocated only on
// registration.
func registerTestingService(server *ttrpc.Server, service testingService) {
	server.Register(serviceName, map[string]ttrpc.Method{
		"Test": func(ctx context.Context, unmarshal func(interface{}) error) (interface{}, error) {
			var req internal.TestPayload
			if err := unmarshal(&req); err != nil {
				return nil, err
			}
			return service.Test(ctx, &req)
		},
	})
}
