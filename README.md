##go-seeker
GO TCP服务开发框架，方便快速搭建一个TCP服务。

##安装方法
```
go get github.com/fbbin/go-seeker
```

##Examples
```
package main

import (
	"flag"
	"go-proxy/packet"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/fbbin/seeker"
	seelog "github.com/cihub/seelog"
)

// 定义系统信号通道
var signalChan = make(chan os.Signal, 1)

var (
	serverPort = flag.String("p", "0.0.0.0:20003", "服务监听的地址和端口号")
	logFileDir = flag.String("l", "./logs/", "服务器日志文件")
)

func main() {
	// 服务配置项
	config := &seeker.Config{
		Debug:                  false,
		ProtocolType:           "tcp4",
		BindAddress:            *serverPort,
		AcceptTimeout:          time.Second * 5,
		AcceptErrorLimit:       uint32(10),
		PacketSendChanLimit:    uint32(10),
		PacketReceiveChanLimit: uint32(10),
	}
	// 初始化日志
	initlog(config)
	// 事件回调
	dispatch := &ConnCallback{}
	dispatch.initialize()
	// 协议处理
	protocol := &packet.SelfProtocol{}
	// 初始化一个服务
	server := seeker.NewServer(config, dispatch, protocol)
	seelog.Info("------ Server " + *serverPort + " Has Started ------- ")
	// 启动服务
	go server.Start()
	// 监听系统服务退出信号
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP, os.Interrupt, os.Kill)
	seelog.Info("Get Signal:%v\r\n", <-signalChan)

	// 服务退出
	server.Stop()
	seelog.Info("------ Server " + *serverPort + " Has Stopped ------- ")
}

func initlog(config *seeker.Config) {
	logger, err := seelog.LoggerFromConfigAsFile("./conf/seelog.xml")
	if err != nil {
		seelog.Critical("日志配置文件解析错误", err)
		return
	}
	seelog.ReplaceLogger(logger)
}
```