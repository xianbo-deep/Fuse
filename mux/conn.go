package mux

import (
	"bufio"
	"net"
	"time"
)

// FuseConn 是 [net.Conn] 的包装器，为协议多路复用（multiplexing）提供增强功能。
//
// 它封装了原始的 TCP 连接，并添加了带缓冲的读取器和预读（peek）能力。
//
// 主要特点：
//
//   - 支持从连接中预读字节而不移动读取指针
//
//   - 提供带缓冲的高效读取，减少系统调用次数
//
//   - 完全实现 [net.Conn] 接口，保持向后兼容性
//
// 包括原始 TCP 连接对象 conn 与带缓冲的读取器。
//
//   - 标准的 [net.Conn] 读取字节后指针会发生移动，数据会丢失。
//
//   - 因此包装了带缓冲的读取器，每次从 [net.Conn] 原始字节流获取字节装入缓冲区进行预读，当字节数不够会持续从原始字节流获取字节，
//     后续处理器获取字节会先优先返回缓冲区中的数据，然后再从底层 Conn 读取
//
// [FuseConn] 用于嗅探连接的前几个字节以确定协议类型。
type FuseConn struct {
	// conn 是底层的原始 TCP 连接。
	conn net.Conn
	// reader 是带缓冲的读取器，包装了原始连接以提高读取性能，允许预读字节以进行协议识别。
	reader *bufio.Reader
}

// NewFuseConn 接收原始 TCP 连接，返回一个 *[FuseConn] 实例。
//
// conn：类型为 [net.Conn]，底层 TCP 连接。
func NewFuseConn(conn net.Conn) *FuseConn {
	return &FuseConn{
		conn:   conn,
		reader: bufio.NewReader(conn), // 相当于reader和conn绑定了 当reader的buffer不够时 会自动从conn的字节流获取
	}
}

// 以下方法实现 [net.Conn] 接口

// Read 从连接中读取数据到指定的字节切片中。
//
// p: 目标字节切片，用于存储读取的数据。
//
// 返回值: 实际读取的字节数和可能的错误
func (c *FuseConn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

// Write 将数据写入底层连接。
//
// p: 要写入的字节切片。
//
// 返回值: 实际写入的字节数和可能的错误。
func (c *FuseConn) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

// Close 关闭底层网络连接。
func (c *FuseConn) Close() error {
	return c.conn.Close()
}

// LocalAddr 返回本地网络地址。
func (c *FuseConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr 返回远程网络地址（如客户端的 IP 地址和端口）。
func (c *FuseConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline 设置连接的读取和写入操作的绝对截止时间。
func (c *FuseConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline 设置连接读取操作的绝对截止时间。
func (c *FuseConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置连接写入操作的绝对截止时间。
func (c *FuseConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// Peek 从缓冲读取器中预读指定数量的字节，而不消耗这些数据。
//
// 这是 [FuseConn] 的关键特性，用于协议检测。预读的数据在后续的 [FuseConn.Read] 调用中
// 仍然可用，这使得多路复用器可以查看连接的前几个字节以确定协议类型，
func (c *FuseConn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}
