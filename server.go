package seeker

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

type Config struct {
	Debug bool
	// 定义服务类型（tcp）
	ProtocolType string
	// 服务端监听的ip和端口号
	BindAddress string
	// 接收超时时间
	AcceptTimeout time.Duration
	// 连接连续错误多少次退出
	AcceptErrorLimit uint32
	// 设置发送消息缓存区大小
	PacketSendChanLimit uint32
	// 设置发送消息缓存区大小
	PacketReceiveChanLimit uint32
}

type Server struct {
	// 服务配置选项
	config *Config
	// 服务消息事件回调处理
	callback ConnCallback
	// 自定义的数据包协议
	protocol Protocol
	// 通知所有协成退出的channel
	exitChan chan struct{}
	// 等待协成处理
	waitGroup *sync.WaitGroup
}

// 初始化一个服务
func NewServer(config *Config, callback ConnCallback, protocol Protocol) *Server {
	return &Server{
		config:    config,
		callback:  callback,
		protocol:  protocol,
		exitChan:  make(chan struct{}),
		waitGroup: &sync.WaitGroup{},
	}
}

// 启动服务
func (svr *Server) Start() {
	// 启用多核处理
	runtime.GOMAXPROCS(runtime.NumCPU())
	tcpAddr, err := net.ResolveTCPAddr(svr.config.ProtocolType, svr.config.BindAddress)
	if err != nil {
		fmt.Println("TCP地址解析失败 : " + err.Error())
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("监听端口失败: %s" + err.Error())
	}
	svr.waitGroup.Add(1)
	defer func() {
		listener.Close()
		svr.waitGroup.Done()
	}()
	for {
		select {
		case <-svr.exitChan:
			fmt.Println("服务退出。")
			return
		default:
		}
		// 设置超时时间
		listener.SetDeadline(time.Now().Add(svr.config.AcceptTimeout))
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		svr.waitGroup.Add(1)
		// 处理客户端连接
		go func() {
			NewConn(conn, svr).Handle()
			svr.waitGroup.Done()
		}()
	}
}

// 停止服务
func (svr *Server) Stop() {
	close(svr.exitChan)
	// 等待尚未结束的协成处理完
	svr.waitGroup.Wait()
}
