package main

import (
	"time"

	"github.com/zhiyunliu/velocity/context"
)

type demo struct{}

func (d *demo) Handle(ctx context.Context) interface{} {
	ctx.Log().Infof("mqc.demo:%s", time.Now().Format("2006-01-02 15:04:05"))

	ctx.Log().Infof("header.a:%+v", ctx.Request().GetHeader("a"))
	ctx.Log().Infof("header.b:%+v", ctx.Request().GetHeader("b"))
	ctx.Log().Infof("header.c:%+v", ctx.Request().GetHeader("c"))

	ctx.Log().Infof("body-1:%s", ctx.Request().Body().Bytes())

	mapData := map[string]string{}
	ctx.Request().Body().Scan(&mapData)
	ctx.Log().Infof("body-2:%+v", mapData)

	return "success"
}
