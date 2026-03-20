package mux

import (
	"errors"
	"log"
	"net"
	"time"
)

// Matcher 是 func(*FuseConn) bool 的类型别名，为协议适配器，内部需要实现对协议类型的判断。
type Matcher func(*FuseConn) bool

type protocol struct {
	matcher  Matcher
	listener *FakeListener
}

// Multiplexer 多路复用器。
//
// 负责
//
//   - 监听端口，处理连接
//   - 根据内部已注册的协议列表找到连接对应的虚拟监听器，对连接进行分发
//   - 接收用户自定义驱动 [Driver] 并将对应的虚拟监听器 [FakeListener] 进行返回，将用户自定义的驱动集成到框架中
type Multiplexer struct {
	protocols        []*protocol
	addr             net.Addr
	handshakeTimeout time.Duration
}

// NewMultiplexer 根据传入的 [net.Addr] 返回 *[Multiplexer] 实例。
func NewMultiplexer(addr net.Addr, d time.Duration) *Multiplexer {
	return &Multiplexer{
		protocols:        make([]*protocol, 0),
		addr:             addr,
		handshakeTimeout: d,
	}
}

// Match 接收 [Matcher] 参数，在多路复用器 [Multiplexer] 中进行虚拟监听器 [FakeListener] 的注册并返回给用户。
func (mux *Multiplexer) Match(m Matcher) *FakeListener {
	ln := NewFakeListener(mux.addr)
	mux.protocols = append(mux.protocols, &protocol{matcher: m, listener: ln})
	return ln
}

// Serve 根据传入的连接从已注册的协议列表进行分发。
//
// 遍历协议列表并使用对应的 [Matcher] 进行匹配。
func (mux *Multiplexer) Serve(conn net.Conn) {
	// 包装
	fc := NewFuseConn(conn)

	// 设置握手超时
	_ = fc.SetReadDeadline(time.Now().Add(mux.handshakeTimeout))
	defer fc.SetReadDeadline(time.Time{}) // 清除超时

	for _, p := range mux.protocols {
		if p.matcher(fc) {
			p.listener.Push(fc)
			return
		}
	}
	// 未知协议
	preview, _ := fc.Peek(8)
	// 可能没有传输数据
	if len(preview) == 0 {
		_ = conn.Close()
		return
	}
	log.Printf("Unknow protocol from %s, preview: %s", conn.RemoteAddr(), string(preview))
	// 关闭底层连接
	_ = conn.Close()
}

// ServeLoop 负责监听连接，阻塞执行，开启新的协程对新连接进行分发。
func (mux *Multiplexer) ServeLoop(ln net.Listener) {
	for {
		// 获取连接对象
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("[FUSE] Accept error: %v", err)
			continue
		}

		go mux.Serve(conn)
	}
}
