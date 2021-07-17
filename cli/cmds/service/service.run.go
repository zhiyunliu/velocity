package service

import (
	"time"
)

func (p *ServiceApp) run() (err error) {

	//3.创建trace性能跟踪
	p.trace, err = startTrace(p.config.TraceType, p.config.TracePort)
	if err != nil {
		return err
	}
	//5. 创建服务器
	errChan := make(chan error)
	go func() {
		err := p.server.Start(p.config)
		errChan <- err
	}()

	select {
	case err = <-errChan:
		return err
	case <-time.After(time.Second):
		return nil
	}
}
