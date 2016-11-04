package seeker

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Conn struct {
	server *Server
	// 链接信息
	conn *net.TCPConn
	// 保存额外数据
	extraData interface{}
	// 保证每个实例执行一次操作
	closeOnce sync.Once
	// 关闭标识
	closeFlag int32
	// 链接关闭channel
	closeChan chan struct{}
	// 发消息channel
	packetSendChan chan Packet
	// 收消息channel
	packetReceiveChan chan Packet
}

type ConnCallback interface {
	// 客户端链接上，回调该方法, 如果返回false,标识关闭链接
	OnConnect(*Conn) bool
	// 当服务收到一个数据包的时候回调，如果返回false,标识关闭链接
	OnMessage(*Conn, Packet) bool
	// 链接断开时回调
	OnClose(*Conn)
}

// 初始化一个链接信息
func NewConn(conn *net.TCPConn, server *Server) *Conn {
	return &Conn{
		server:            server,
		conn:              conn,
		closeChan:         make(chan struct{}),
		packetSendChan:    make(chan Packet, server.config.PacketSendChanLimit),
		packetReceiveChan: make(chan Packet, server.config.PacketReceiveChanLimit),
	}
}

// SetExtraData 设置额外参数
func (conn *Conn) SetExtraData(data interface{}) {
	conn.extraData = data
}

// GetExtraData 获取额外参数
func (conn *Conn) GetExtraData() interface{} {
	return conn.extraData
}

// GetRawConn 获取当前连接的指针
func (conn *Conn) GetRawConn() *net.TCPConn {
	return conn.conn
}

// Close 关闭连接处理（原子操作）
func (conn *Conn) Close() {
	conn.closeOnce.Do(func() {
		// 原子操作
		atomic.StoreInt32(&conn.closeFlag, 1)
		//		close(conn.server.exitChan)
		close(conn.closeChan)
		close(conn.packetReceiveChan)
		close(conn.packetSendChan)
		// 关闭当前连接
		conn.conn.Close()
		// 通知回调
		conn.server.callback.OnClose(conn)
	})
}

// IsClosed 检测客户端是否已经关闭
func (conn *Conn) IsClosed() bool {
	return atomic.LoadInt32(&conn.closeFlag) == 1
}

// SendPacket 发送消息协议包给客户端
func (conn *Conn) SendPacket(data Packet, timeout time.Duration) (err error) {
	if conn.IsClosed() {
		return errors.New("连接已经关闭，写入失败")
	}
	defer func() {
		if e := recover(); e != nil {
			err = errors.New("系统错误")
		}
	}()
	if timeout == 0 {
		select {
		case conn.packetSendChan <- data:
			return nil
		default:
			return errors.New("数据写入阻塞中.")
		}
	} else {
		select {
		case conn.packetSendChan <- data:
			return nil
		case <-conn.closeChan:
			return errors.New("写入通道已被关闭，写入失败")
		case <-time.After(timeout):
			return errors.New("数据写入阻塞中.")
		}
	}
}

// Handle 维护接收，发送，协成的处理
func (conn *Conn) Handle() {
	if !conn.server.callback.OnConnect(conn) {
		return
	}
	doCallBackAction(conn.message, conn.server.waitGroup)
	doCallBackAction(conn.read, conn.server.waitGroup)
	doCallBackAction(conn.write, conn.server.waitGroup)
}

// read 处理从连接中读取一个协议包到接收通道中
func (conn *Conn) read() {
	defer func() {
		recover()
		conn.Close()
	}()
	for {
		select {
		case <-conn.server.exitChan:
			return

		case <-conn.closeChan:
			return

		default:
		}
		packetData, err := conn.server.protocol.ReadPacket(conn.conn)
		if err != nil {
			return
		}
		conn.packetReceiveChan <- packetData
	}
}

// write 将要发送的数据写入发送消息通道
func (conn *Conn) write() {
	defer func() {
		recover()
		conn.Close()
	}()
	for {
		select {
		case <-conn.server.exitChan:
			return

		case <-conn.closeChan:
			return

		case packetData := <-conn.packetSendChan:
			if conn.IsClosed() {
				return
			}
			_, err := conn.conn.Write(packetData.Serialize())
			if err != nil {
				return
			}
		}
	}
}

// message 处理收包通道的协议，并回调用户的callback
func (conn *Conn) message() {
	defer func() {
		//		recover()
		conn.Close()
	}()
	for {
		select {
		case <-conn.server.exitChan:
			return

		case <-conn.closeChan:
			return

		case packetData := <-conn.packetReceiveChan:
			if conn.IsClosed() {
				return
			}
			if !conn.server.callback.OnMessage(conn, packetData) {
				return
			}
		}
	}
}

// doCallBackAction 异步统一处理
func doCallBackAction(callback func(), wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		callback()
		wg.Done()
	}()
}
