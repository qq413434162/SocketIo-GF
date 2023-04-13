// Package core 面向开发者高度封装配置
//
// 避免频繁使用框架提供的方法读取配置,造成大量的代码冗余
package core

import (
	"os"
	"socketio-gf/internal/tool"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
)

// Config 统一配置全局变量
var Config = &config{}

type config struct {
	AppName string        // 程序应用名
	Env     *environment // 通用配置
	Server  *serverConfig // web配置
	Tracer  *tracer       // 链路追踪配置
}

// 链路追踪配置
type tracer struct {
	Enable  bool
	Address string
}

// 程序基本配置
type environment struct {
	// 调试模式 测试环境=true 正式环境=false
	Debug bool
	// 环境命名空间
	EnterpriseId string
	// 是否是私有环境
	IsPrivate bool
	// 环境模式
	Profile *profileOptional
	// 对外显示的ip
	Host string
	// 对外显示的协议
	Scheme string

	// current working directory
	CWD              string
}

// 环境模式
type profileOptional struct {
	// 本地
	Local bool
	// 开发
	Dev bool
	// 测试
	Test bool
	// 预发布
	Rgs bool
	// 生产=正式
	Prod bool
}

// web服务配置
type serverConfig struct {
	// 对外显示的地址 可以是host可以是ip
	Host string
	// 启动监听的ip
	Ip string
	// 启动监听的端口
	Port int
	// 客户端终止后还能执行多久
	RequestFinishWaitTimeout time.Duration
}

func init() {
	Config.Init()
}

func (c *config) Init() *config {
	ctx := gctx.New()
	configObj := gcfg.Instance()

	env := environment{}
	server := serverConfig{}
	profile := profileOptional{}
	trace := tracer{}

	c.AppName = configObj.MustGet(ctx, "server.ServerAgent", "Unknown").String()

	// 环境模式
	switch configObj.MustGet(ctx, "env.Profile", "local").String() {
	case "local":
		profile.Local = true
	case "dev":
		profile.Dev = true
	case "test":
		profile.Test = true
	case "rgs":
		profile.Rgs = true
	case "prod":
		profile.Prod = true
	}

	// 启动配置
	env.Debug = configObj.MustGet(ctx, "env.Debug", false).Bool()
	env.EnterpriseId = configObj.MustGet(ctx, "env.EnterpriseId", "").String()
	if env.EnterpriseId == "" {
		env.EnterpriseId = "local"
	}
	env.IsPrivate = configObj.MustGet(ctx, "env.Private", false).Bool()
	env.CWD, _ = os.Getwd()

	// web服务配置
	env.Host = configObj.MustGet(ctx, "env.Host", "").String()
	if env.Host == "" {
		env.Host = "127.0.0.1"
	}
	env.Scheme = configObj.MustGet(ctx, "env.Scheme", "").String()
	if env.Scheme == "" {
		env.Scheme = "http"
	}
	ip := gstr.Split(configObj.MustGet(ctx, "server.Address", ":80").String(), ":")[0]
	if ip == "" {
		if i, err := tool.Ip.GetClientIp(); err != nil {
			ip = "127.0.0.1"
		} else {
			ip = i
		}
	}
	server.Ip = ip
	var port = 80
	if addrSplit := gstr.Split(configObj.MustGet(ctx, "server.Address", ":80").String(), ":"); len(addrSplit) > 0 {
		port = gconv.Int(addrSplit[1])
	}
	server.Port = port
	server.RequestFinishWaitTimeout = configObj.MustGet(ctx, "server.RequestFinishWaitTimeout", "30s").Duration()

	// 链路配置
	trace.Enable = configObj.MustGet(ctx, "tracer.Enable", false).Bool()
	trace.Address = configObj.MustGet(ctx, "tracer.Address", "localhost:6831").String()

	env.Profile = &profile
	Config.Tracer = &trace
	Config.Server = &server
	Config.Env = &env

	return c
}
