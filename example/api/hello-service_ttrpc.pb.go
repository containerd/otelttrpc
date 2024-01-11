// Code generated by protoc-gen-go-ttrpc. DO NOT EDIT.
// source: api/hello-service.proto
package api

import (
	context "context"
	ttrpc "github.com/containerd/ttrpc"
)

type HelloServiceService interface {
	SayHello(context.Context, *HelloRequest) (*HelloResponse, error)
}

func RegisterHelloServiceService(srv *ttrpc.Server, svc HelloServiceService) {
	srv.RegisterService("api.HelloService", &ttrpc.ServiceDesc{
		Methods: map[string]ttrpc.Method{
			"SayHello": func(ctx context.Context, unmarshal func(interface{}) error) (interface{}, error) {
				var req HelloRequest
				if err := unmarshal(&req); err != nil {
					return nil, err
				}
				return svc.SayHello(ctx, &req)
			},
		},
	})
}

type helloServiceClient struct {
	client *ttrpc.Client
}

func NewHelloServiceClient(client *ttrpc.Client) HelloServiceService {
	return &helloServiceClient{
		client: client,
	}
}

func (c *helloServiceClient) SayHello(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	var resp HelloResponse
	if err := c.client.Call(ctx, "api.HelloService", "SayHello", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}