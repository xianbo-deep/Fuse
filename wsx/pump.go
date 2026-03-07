package wsx

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Pump struct {
	writeChan chan []byte
	conn      *websocket.Conn
	done      chan struct{}
	mu        *sync.Mutex
}

func NewPump(conn *websocket.Conn, done chan struct{}, writeChan chan []byte, mu *sync.Mutex) *Pump {
	return &Pump{
		writeChan: writeChan,
		conn:      conn,
		done:      done,
		mu:        mu,
	}
}

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
