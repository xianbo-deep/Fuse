package wsx

import (
	"Fuse/core"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type WsContext struct {
	core.Ctx
	Conn      *websocket.Conn
	MsgType   int
	Data      []byte
	WriteChan chan<- []byte // 只写通道
}

func NewWsContext(c core.Ctx, conn *websocket.Conn, msgType int, data []byte, writeChan chan<- []byte) *WsContext {
	return &WsContext{
		Conn:      conn,
		Ctx:       c,
		MsgType:   msgType,
		Data:      data,
		WriteChan: writeChan,
	}
}

func (wsc *WsContext) Send(data []byte) {
	wsc.WriteChan <- data
}

func (wsc *WsContext) BindJSON(obj interface{}) error {
	return json.Unmarshal(wsc.Data, obj)
}

func (wsc *WsContext) SendJSON(obj interface{}) error {
	// 序列化成字节
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	wsc.Send(data)

	return nil
}

// 关闭TCP连接
func (wsc *WsContext) Close() error {
	return wsc.Conn.Close()
}
