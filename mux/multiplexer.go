package mux

import (
	"errors"
	"log"
	"net"
	"time"
)

type Matcher func(*FuseConn) bool

type Handler func(*FuseConn)

// 多路复用器
type protocol struct {
	matcher  Matcher
	listener *FakeListener
}
type Multiplexer struct {
	protocols []*protocol
	addr      net.Addr
}

func NewMultiplexer(addr net.Addr) *Multiplexer {
	return &Multiplexer{
		protocols: make([]*protocol, 0),
		addr:      addr,
	}
}

// 暴露监听器
func (mux *Multiplexer) Match(m Matcher) *FakeListener {
	ln := NewFakeListener(mux.addr)
	mux.protocols = append(mux.protocols, &protocol{matcher: m, listener: ln})
	return ln
}

// 分发
func (mux *Multiplexer) Serve(conn net.Conn) {
	// 包装
	fc := NewFuseConn(conn)

	// 设置握手超时
	_ = fc.SetReadDeadline(time.Now().Add(3 * time.Second))
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

// 监听
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
