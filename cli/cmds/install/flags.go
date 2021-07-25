package install

import (
	"github.com/urfave/cli"
	"github.com/zhiyunliu/velocity/cli/cmds/service"
	"github.com/zhiyunliu/velocity/globals"
)

//getFlags 获取运行时的参数
func getFlags(cfg *globals.AppSetting) []cli.Flag {
	flags := service.GetBaseFlags(cfg)
	flags = append(flags, cli.BoolFlag{
		Name:        "debug,d",
		Destination: &cfg.IsDebug,
		Usage:       `-调试模式，打印更详细的系统运行日志，避免将详细的错误信息返回给调用方`,
	})
	flags = append(flags, cli.StringFlag{
		Name:        "trace,t",
		Destination: &cfg.TraceType,
		Usage:       `-性能分析。支持:cpu,mem,block,mutex,web`,
	})
	flags = append(flags, cli.StringFlag{
		Name:        "mask,msk",
		Destination: &cfg.IPMask,
		Usage:       `-子网掩码。多个网卡情况下根据mask获取本机IP`,
	})
	return flags
}
