# 协议说明

## 单端口协议分发

**实现单端口协议分发需要在TCP层对请求进行预读，将流量分发到不同的处理器**

### 预读

TCP本质是**字节流**，没有消息边界

- 正常读取字节会导致**字节在缓冲区丢失**
- 使用`MSG_PEEK`可以实现**读取数据且防止字节丢失**
  - 从`net.Conn`读取部分字节到`bufio.Reader`，这部分字节在`net.Conn`丢失，但是会进入`bufio.Reader`的缓冲区

具体需要对`net.Conn`对象进行包装
- 读取数据时先读取`bufio.Reader`的缓存，再读取`net.Conn`的剩余字节
- 包装后的对象可以给后续应用层协议复用，因为它们只接受`net.Conn`对象
- 需要实现`net.Conn`接口

### 协议指纹

**HTTP/1.1**

- 纯文本协议
- 前几个字节结构:`Method Path Version`
- 读取前几个字节就可以得出是HTTP/1.1


**HTTP2**

- 二进制协议，定义了**固定连接前奏**:`PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n`
- 读取到`PRI *`就可以确定是HTTP2
- grpc基于HTTP2

**Websocket**

- 基于HTTP，先识别HTTP，再解析Header，如果发现有`Upgrade: websocket`，就说明是Websocket
- 传输层不进行预读，在HTTP层判断

**SSE**

- 传输层不进行预读，在HTTP层判断
- Header通常是
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

### 连接分发

底层使用TCP进行监听，使用`Accept()`方法获取请求并解析成`conn`对象后，开协程对每个`conn`对象进行处理

除此之外需要设置**握手超时**，防止恶意请求（连接但不发送数据）耗尽资源

- 需要对预读字节进行判断，查看是否是因为发送了空字节的TCP连接引起的


### Fake Listener

虚拟监听器，接收`multiplexer`发送的请求，分发到不同引擎进行处理

- 需要实现`net.Listener`接口
- 挂载到自定义的http、grpc引擎进行请求监听，，`Serve()`会调用虚拟监听器实现的`Accept()`方法进行请求的处理
- 分发器将请求发送到有缓冲通道，防止应用层对请求无法及时处理造成分发器阻塞
