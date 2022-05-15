package grpc

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/zhiyunliu/gel/contrib/xhttp/default/balancer"
	"github.com/zhiyunliu/gel/registry"
	"github.com/zhiyunliu/gel/selector"
	"github.com/zhiyunliu/gel/selector/filter"
	"github.com/zhiyunliu/gel/xhttp"
	"github.com/zhiyunliu/golibs/httputil"
)

type Client struct {
	registrar registry.Registrar
	setting   *setting
	client    *http.Client
	selector  selector.Selector
	ctx       context.Context
	ctxCancel context.CancelFunc
}

//NewClientByConf 创建RPC客户端,地址是远程RPC服务器地址或注册中心地址
func NewClient(registrar registry.Registrar, setting *setting, serviceName string) (*Client, error) {
	client := &Client{
		registrar: registrar,
		setting:   setting,
		client:    &http.Client{},
	}
	tlsCfg, err := client.getTlsConfig()
	if err != nil {
		return nil, err
	}

	client.selector, err = balancer.NewSelector(client.ctx, registrar, serviceName, setting.Balancer)
	if err != nil {
		return nil, err
	}

	client.client.Transport = &http.Transport{
		TLSClientConfig: tlsCfg,
		Proxy:           client.getProxy(),
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(setting.ConnTimeout) * time.Second,
			KeepAlive: time.Duration(setting.KeepaliveTimeout) * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          setting.MaxIdleConns,
		IdleConnTimeout:       time.Duration(setting.IdleConnTimeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(setting.TLSHandshakeTimeout) * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client.ctx, client.ctxCancel = context.WithCancel(context.Background())
	return client, nil
}

//RequestByString 发送Request请求
func (c *Client) RequestByString(ctx context.Context, reqPath *url.URL, input []byte, opts ...xhttp.RequestOption) (res xhttp.Body, err error) {
	//处理可选参数
	o := &xhttp.Options{
		Method: http.MethodGet,
		Header: make(map[string]string),
	}
	for _, opt := range opts {
		opt(o)
	}

	response, err := c.clientRequest(ctx, reqPath, o, input)
	if err != nil {
		return newBodyByError(err), err
	}
	return response, err
}

//Close 关闭RPC客户端连接
func (c *Client) Close() {
	c.ctxCancel()
}

func (c *Client) clientRequest(ctx context.Context, reqPath *url.URL, o *xhttp.Options, input []byte) (response xhttp.Body, err error) {

	node, done, err := c.selector.Select(ctx, selector.WithFilter(filter.Version(o.Version)))
	if err != nil {
		return nil, err
	}
	defer func() {
		done(ctx, selector.DoneInfo{Err: err})
	}()

	httpOpts := make([]httputil.Option, 0)
	for k, v := range o.Header {
		httpOpts = append(httpOpts, httputil.WithHeader(k, v))
	}
	httpOpts = append(httpOpts, httputil.WithClient(c.client))

	return httputil.Request(o.Method, fmt.Sprintf("%s%s", node.Address(), reqPath.Path), input, httpOpts...)
}

func (c *Client) getTlsConfig() (*tls.Config, error) {
	ssl := &tls.Config{InsecureSkipVerify: true}
	if c.setting.CertFile != "" && c.setting.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.setting.CertFile, c.setting.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("LoadX509KeyPair(CertFile: %s, KeyFile: %s),error:%v", c.setting.CertFile, c.setting.KeyFile, err)
		}
		ssl.Certificates = []tls.Certificate{cert}
	}
	if c.setting.CaFile != "" {
		caData, err := ioutil.ReadFile(c.setting.CaFile)
		if err != nil {
			return nil, fmt.Errorf("CaFile(%s) error:%v", c.setting.CaFile, err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		ssl.RootCAs = pool
	}
	if len(ssl.Certificates) == 0 && ssl.RootCAs == nil {
		return ssl, nil
	}
	ssl.Rand = rand.Reader
	return ssl, nil
}

func (c *Client) getProxy() func(*http.Request) (*url.URL, error) {
	if c.setting.ProxyURL != "" {
		return func(_ *http.Request) (*url.URL, error) {
			return url.Parse(c.setting.ProxyURL)
		}
	}
	return nil
}
