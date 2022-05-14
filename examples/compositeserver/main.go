package main

import (
	"github.com/zhiyunliu/gel"
	_ "github.com/zhiyunliu/gel/contrib/cache/redis"
	_ "github.com/zhiyunliu/gel/contrib/config/consul"
	_ "github.com/zhiyunliu/gel/contrib/config/nacos"
	_ "github.com/zhiyunliu/gel/contrib/queue/redis"
	_ "github.com/zhiyunliu/gel/contrib/registry/nacos"
	_ "github.com/zhiyunliu/gel/contrib/xdb/mysql"
	_ "github.com/zhiyunliu/gel/contrib/xdb/oracle"
	_ "github.com/zhiyunliu/gel/contrib/xdb/postgres"
	_ "github.com/zhiyunliu/gel/contrib/xdb/sqlite"
	_ "github.com/zhiyunliu/gel/contrib/xdb/sqlserver"

	_ "github.com/zhiyunliu/gel/contrib/dlocker/redis"
)

var (
	opts = []gel.Option{}
)

func main() {

	app := gel.NewApp(opts...)
	app.Start()
}