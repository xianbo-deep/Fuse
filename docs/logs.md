# 2026-03-02
# 2026-03-03
# 2026-03-04
# 2026-03-05
# 2026-03-06

今日实现sse，完善心跳检测

- 通过启动一个守护进程进行心跳检测，防止负载均衡器、网关等组件因无字节传输掐断连接
- 设置SSE响应头

```go
// 设置SSE响应头
func (c *Ctx) SetSSEHeader() {
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
}
```

- 发送的消息需要是`key:value`的形式，需要以`\n`结尾，且每一行都需要有`\n`
- 可通过请求头来判断是否为SSE连接

```go
if strings.Contains(request.Header.Get("Accept"), "text/event-stream") {
		c.Set(core.CtxKeyProtocol, core.ProtocolSSE)
}
```


# 2026-03-07

今日实现Websocket的消息泵，提升框架封装性且实现客户端与服务端的双向通信

封装了`WsContext`，里面包含连接对象、数据、数据类型等信息

- 用户无需关心心跳机制、双向通信如何实现，可直接从`WsContext`直接获取客户端发来的信息类型与信息
- 用户可直接调用`WsContext`的`Send()`方法发送信息给客户端，底层已经做好封装
  - 底层基于Channel发送信息，将信息发送到channel，由消息泵通过其内部封装的`conn`对象消费channel中的信息
