package mqc

import (
	"context"
	"fmt"

	"github.com/zhiyunliu/golibs/xnet"
	"github.com/zhiyunliu/velocity/config"
	"github.com/zhiyunliu/velocity/log"
	"github.com/zhiyunliu/velocity/middleware"
	"github.com/zhiyunliu/velocity/server"
	"github.com/zhiyunliu/velocity/transport"
)

type Server struct {
	name      string
	processor *processor
	ctx       context.Context
	opts      options
	started   bool
}

var _ transport.Server = (*Server)(nil)

// New 实例化
func New(name string, opts ...Option) *Server {
	s := &Server{
		name: name,
		opts: setDefaultOption(),
	}
	s.Options(opts...)

	return s
}

// Options 设置参数
func (e *Server) Options(opts ...Option) {
	for _, o := range opts {
		o(&e.opts)
	}
}

func (e *Server) Name() string {
	if e.name == "" {
		e.name = e.Type()
	}
	return e.name
}

func (e *Server) Type() string {
	return Type
}

func (e *Server) Config(cfg config.Config) {
	e.Options(WithConfig(cfg))
	setting := &Setting{}
	cfg.Get(fmt.Sprintf("servers.%s", e.Name())).Scan(setting)
	e.opts.setting = setting
}

// Start 开始
func (e *Server) Start(ctx context.Context) error {
	e.ctx = ctx
	e.newProcessor()
	if len(e.opts.startedHooks) > 0 {
		for _, fn := range e.opts.startedHooks {
			err := fn(ctx)
			if err != nil {
				log.Error("mqc.StartedHooks:", err)
				return err
			}
		}
	}
	go func() {

		e.started = true
		err := e.processor.Start()
		if err != nil {
			log.Errorf("[%s] Server start error: %s", e.name, err.Error())
		}
		<-ctx.Done()

		if len(e.opts.endHooks) > 0 {
			for _, fn := range e.opts.endHooks {
				err := fn(ctx)
				if err != nil {
					log.Error("mqc.endHooks:", err)
				}
			}
		}
		err = e.Stop(ctx)
		if err != nil {
			log.Errorf("[%s] Server shutdown error: %s", e.name, err.Error())
		}
	}()

	return nil
}

// Attempt 判断是否可以启动
func (e *Server) Attempt() bool {
	return !e.started
}

// Shutdown 停止
func (e *Server) Stop(ctx context.Context) error {
	return e.processor.Close()
}

func (e *Server) newProcessor() {
	var err error
	config := e.opts.setting.Config
	//queue://default
	protoType, configName, err := xnet.Parse(config.Addr)
	if err != nil {
		panic(err)
	}
	cfg := e.opts.config.Get(protoType)
	queueVal := cfg.Value(configName)
	ptotoCfg := queueVal.String()

	//redis://localhost
	protoType, configName, err = xnet.Parse(ptotoCfg)
	if err != nil {
		panic(err)
	}

	cfg = e.opts.config.Get(fmt.Sprintf("%s.%s", protoType, configName))
	e.processor, err = newProcessor(protoType, cfg)
	if err != nil {
		panic(err)
	}

	err = e.processor.Add(e.opts.setting.Tasks...)
	if err != nil {
		panic(err)
	}
	e.registryEngineRoute()
}

func (e *Server) Use(middlewares ...middleware.Middleware) {
	e.opts.router.Use(middlewares...)
}

func (e *Server) Group(group string, middlewares ...middleware.Middleware) *server.RouterGroup {
	return e.opts.router.Group(group, middlewares...)
}

func (e *Server) Handle(queue string, obj interface{}, methods ...server.Method) {
	e.opts.router.Handle(getService(queue), obj, methods...)
}
