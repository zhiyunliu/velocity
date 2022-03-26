package transport

import (
	"context"
	"net/url"

	"github.com/zhiyunliu/velocity/config"
)

// Server is transport server.
type Server interface {
	Name() string
	Type() string
	Start(context.Context) error
	Stop(context.Context) error
	Config(cfg config.Config)
}

// Endpointer is registry endpoint.
type Endpointer interface {
	Endpoint() *url.URL
}

// Header is the storage medium used by a Header.
type Header interface {
	Get(key string) string
	Set(key string, value string)
	Keys() []string
}

// Transporter is transport context value interface.
type Transporter interface {
	// Kind transporter
	// grpc
	// http
	Kind() Kind
	// Endpoint return server or client endpoint
	// Server Transport: grpc://127.0.0.1:9000
	// Client Transport: discovery:///provider-demo
	Endpoint() string
}

// Kind defines the type of Transport
type Kind string

func (k Kind) String() string { return string(k) }

// type (
// 	serverTransportKey struct{}
// 	clientTransportKey struct{}
// )

// // NewServerContext returns a new Context that carries value.
// func NewServerContext(ctx context.Context, tr Transporter) context.Context {
// 	return context.WithValue(ctx, serverTransportKey{}, tr)
// }

// // FromServerContext returns the Transport value stored in ctx, if any.
// func FromServerContext(ctx context.Context) (tr Transporter, ok bool) {
// 	tr, ok = ctx.Value(serverTransportKey{}).(Transporter)
// 	return
// }

// // NewClientContext returns a new Context that carries value.
// func NewClientContext(ctx context.Context, tr Transporter) context.Context {
// 	return context.WithValue(ctx, clientTransportKey{}, tr)
// }

// // FromClientContext returns the Transport value stored in ctx, if any.
// func FromClientContext(ctx context.Context) (tr Transporter, ok bool) {
// 	tr, ok = ctx.Value(clientTransportKey{}).(Transporter)
// 	return
// }
