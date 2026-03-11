package mux

import (
	"net"
)

// FakeListener 是虚拟的网络监听器，实现了 [net.Listener] 接口。
//
// 它不实际监听网络端口，而是通过内部通道接收和分发网络连接。
//
// 由分发器 [Multiplexer] 对协议进行检测后分发到不同的虚拟监听器 [FakeListener],
// 实现单个端口上多协议的共存。
type FakeListener struct {
	addr     net.Addr
	connChan chan net.Conn
	done     chan struct{}
}

// NewFakeListener 创建一个新的 FakeListener 实例。
func NewFakeListener(addr net.Addr) *FakeListener {
	return &FakeListener{
		addr:     addr,
		connChan: make(chan net.Conn, 128),
		done:     make(chan struct{}),
	}
}

// 以下方法实现 net.Listener 接口

// Accept 等待并返回下一个连接到监听器的连接。
//
// 这是 [net.Listener] 接口的核心方法，会被 [github.com/xianbo-deep/Fuse/httpx.Driver] 和 [github.com/xianbo-deep/Fuse/grpcx.Driver] 的 Serve 方法调用。
func (l *FakeListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connChan:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Addr 返回底层监听 TCP 连接的地址。
func (l *FakeListener) Addr() net.Addr {
	return l.addr
}

// Close 关闭虚拟监听器，停止接受新连接。
func (l *FakeListener) Close() error {
	close(l.done)
	return nil
}

// Push 暴露给分发器 [Multiplexer]，用于推送连接到虚拟监听器。
//
// 执行逻辑：
//  1. 尝试将连接推送到缓冲通道
//  2. 如果成功，连接将被对应的服务器 [Accept] 获取
//  3. 如果监听器已关闭，立即关闭传入的连接，避免泄漏
func (l *FakeListener) Push(conn net.Conn) {
	select {
	case l.connChan <- conn:
	case <-l.done:
		conn.Close()
	}
}
