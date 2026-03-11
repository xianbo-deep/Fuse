package wsx

import (
	"encoding/json"

	"github.com/xianbo-deep/Fuse/core"

	"github.com/gorilla/websocket"
)

// WsContext 是 Websocket 协议中所需的上下文。
//
// 继承了 [core.Ctx] 的方法与成员变量。
type WsContext struct {
	core.Ctx
	Conn      *websocket.Conn
	MsgType   int
	Data      []byte
	WriteChan chan<- []byte // 只写通道
}

// NewWsContext 返回 [WsContext] 实例。
func NewWsContext(c core.Ctx, conn *websocket.Conn, msgType int, data []byte, writeChan chan<- []byte) *WsContext {
	return &WsContext{
		Conn:      conn,
		Ctx:       c,
		MsgType:   msgType,
		Data:      data,
		WriteChan: writeChan,
	}
}

// Send 发送信息到客户端。
//
// 先将信息发送到 WriteChan 只写通道，后续由写泵 [Pump] 真正将信息从服务端发送到客户端。
func (wsc *WsContext) Send(data []byte) {
	wsc.WriteChan <- data
}

// BindJSON 将收到的信息绑定在结构体上。
func (wsc *WsContext) BindJSON(obj interface{}) error {
	return json.Unmarshal(wsc.Data, obj)
}

// SendJSON 发送JSON信息。
func (wsc *WsContext) SendJSON(obj interface{}) error {
	// 序列化成字节
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	wsc.Send(data)

	return nil
}

// Close 用于关闭底层的 TCP 连接。
func (wsc *WsContext) Close() error {
	return wsc.Conn.Close()
}
