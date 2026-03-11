package wsx

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Pump 是 websocket 模块的消息泵，执行从服务端发送消息到客户端的任务。
//
// 用户通过 [WsContext.Send] 发送信息到 writeChan，消息泵从 writeChan（有缓冲通道）接收需要发送到客户端的信息并执行发送。
//
// 使用读锁防止发信息与心跳检测产生冲突，保证线程安全。
//
// 内部维护 done 监听用户是否断连，防止资源泄漏。
type Pump struct {
	writeChan chan []byte
	conn      *websocket.Conn
	done      chan struct{}
	mu        *sync.Mutex
}

// NewPump 返回一个 [Pump] 实例。
func NewPump(conn *websocket.Conn, done chan struct{}, writeChan chan []byte, mu *sync.Mutex) *Pump {
	return &Pump{
		writeChan: writeChan,
		conn:      conn,
		done:      done,
		mu:        mu,
	}
}

// WritePump 写泵，从服务端发送消息给客户端。
//
// 负责
//
//   - 监听 writeChan 通道，当你调用 [WsContext] 的 Send 方法时，它会将消息推送到 writeChan 通道，写泵在这里真正地将消息推送到客户端。
//   - 监听 done 通道，当客户端断连，可以释放写泵资源。
//   - 设置写超时时间，防止客户端恶意占用连接导致资源浪费。
func (p *Pump) WritePump() {
	for {
		select {
		case msg, ok := <-p.writeChan:
			if !ok {
				return
			}
			p.mu.Lock()
			p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) // 防止客户端网络不好 消息无法发出阻塞协程
			p.conn.WriteMessage(websocket.TextMessage, msg)
			p.mu.Unlock()
		case <-p.done:
			return
		}
	}
}
