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

type Multiplexer struct {
	http1Listener *FakeListener
	http2Listener *FakeListener
}

func NewMultiplexer(addr net.Addr) *Multiplexer {
	return &Multiplexer{
		http1Listener: NewFakeListener(addr),
		http2Listener: NewFakeListener(addr),
	}
}

// 暴露监听器
func (mux *Multiplexer) HTTP1Listener() *FakeListener {
	return mux.http1Listener
}
func (mux *Multiplexer) HTTP2Listener() *FakeListener {
	return mux.http2Listener
}

// 分发
func (mux *Multiplexer) Serve(conn net.Conn) {
	// 包装
	fc := NewFuseConn(conn)

	// 设置握手超时
	_ = fc.SetReadDeadline(time.Now().Add(3 * time.Second))
	defer fc.SetReadDeadline(time.Time{}) // 清除超时

	if IsHTTP1(fc) {
		mux.http1Listener.Push(fc)
		return
	}
	if IsHTTP2(fc) {
		mux.http2Listener.Push(fc)
		return
	}
	// 未知协议
	preview, _ := fc.Peek(8)
	log.Printf("Unknow Protocol from %s, preview: %s", conn.RemoteAddr(), string(preview))
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
