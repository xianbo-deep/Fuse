
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

# 2026-03-08

今日实现协议动态注册，重构了很大一部分代码，具体如下

- 引入`Driver`接口，包括`Serve()`方法、`Stop()`方法、`ApplyMiddlewares()`方法、`Match()`方法，**用户可根据实际需求实现这个接口，从而传入自定义的驱动**
- 将`Multiplexer`结构体进行重构，使用`protocol`结构体存储`Matcher`与`FakeListener`，有利于后续扩展协议树，从而加快匹配速度，且可以根据传入的`net.Addr`
- 对`Fuse`的代码进行重构，解耦了优雅停机，新增注册驱动的方法，将挂载中间件的代码也进行解耦，同时对各个引擎返回进行类型断言


# 2026-03-12

写限流中间件的时候发现需要获取请求IP以限流，就去给Ctx接口加了个方法，用于获取客户端真实IP，但是发现需要实现可信代理，在http引擎加方法以设置可信代理

- 用户传入的代理是纯IP，需要加 /32 以表示这是单个主机，ipv6则是加 /128
- 使用 `net.ParseCIDR()`方法将字符串转换成 `ip.Net`对象
- 用户传入的是网段则直接添加到可信代理列表即可
- 后面的数字是网络掩码，表示前几位是固定的，是一个网段


# 2026-03-13

通过 LRU 缓存管理限流器，只保留最近的10000个限流器，防止OOM，用了封装好的双向链表进行查找。

除此之外，看了 Gin 对于 Query 的解析，它们会对 Query 进行提前的缓存，后续重复查询时调用缓存即可，防止高并发下性能的损失，但是我目前的方法是每次 Query 都会触发 URL.Query 的解析，高并发场景下性能会差一点


# 2026-03-14

今天看了下 GRPC 如何获取客户端 IP，GRPC 协议下请求头等信息都是在 metadata 里面的，因此要先用 `metadata.FromIncomingContext()` 从ctx里面拿到元信息，然后正常从 metadata 里面获取客户端IP

如果之前从请求头获取IP失败了，就需要从底层TCP连接获取：调用 `peer.FromContext()`方法从上下文获取 Peek 指针，进行类型断言，获取不带端口号的纯净IP。如果这条路走不通，直接拿原始字符串调用 `net.SplitHostPort()` 方法分隔端口获取IP

今天还发现 GRPC 的 Serve 方法是有严重 bug 的：在 Serve 方法我又重新初始化了 Server ，并且 grpc server 自带的优雅停机是阻塞的，如果客户端不断开连接，grpc 服务会永久阻塞，后续修改为开启一个协程执行gprc的优雅停机，并使用通道来进行监听，如果5s内不关闭进行强制停止



# 2026-03-21

实现了配置管理，只不过配置的内容还比较简陋，然后把一些冗余的代码删除了，比如底层引擎的`Default()`方法，这个本质是给引擎启用默认的中间件，但是我感觉可以在驱动层面或者框架侧面做这个事情，就把他删了

全局配置还是放在`core`包下，协议配置就放在每个协议包内，用户可以传入自定义配置，不过需要调用框架的`Register()`方法传入自定义配置

同时增强了SSE、Ws开发的可读性，引入`Option`模式，通过链式调用进行自定义配置，这样每个路由都可以有自己的配置，还不用写复杂的结构体。