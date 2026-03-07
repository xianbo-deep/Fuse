# 协议与连接分发

Fuse 框架的核心特性之一是支持在单端口上同时运行 HTTP/1.1 和 HTTP/2 (gRPC) 服务。这是通过 TCP 层面的流量分析与分发实现的。

## 核心组件：Multiplexer (分发器)

`mux.Multiplexer` 是协议分发的核心组件。它接管了底层的 TCP 监听，负责接受所有连接，并根据协议类型将连接分发给不同的处理引擎。

### 工作流程

1.  **监听**: `fuse.Run` 启动时，首先在指定端口创建一个 TCP 监听器 (`net.Listener`)。
2.  **接收连接**: `Multiplexer` 循环调用 `Accept` 接收客户端连接。
3.  **封装连接**: 每个新连接被封装为 `FuseConn`，这是一个带有缓冲读取能力的连接包装器。
4.  **预读识别**: 读取连接的前几个字节（Peek），根据特征判断协议类型。
    *   **HTTP/1.1**: 检查是否以 HTTP 方法 (`GET`, `POST` 等) 开头。
    *   **HTTP/2**: 检查是否有各个 HTTP/2 连接前奏 (`PRI * HTTP/2.0...`)。
5.  **分发**:
    *   识别为 HTTP/1.1 的连接推送到 `HTTP1Listener`。
    *   识别为 HTTP/2 的连接推送到 `HTTP2Listener`。
    *   无法识别的协议将被关闭。

## 核心技术实现

### 1. FuseConn 与 预读 (Peeking)

标准 `net.Conn` 读取数据后，数据就会从缓冲区移出，后续处理器无法再次读取。为了实现协议识别，必须能够"偷看"数据而不消耗它们。

`FuseConn` 组合了 `net.Conn` 和 `bufio.Reader`：
*   **预读**: 使用 `bufio.Reader.Peek()` 读取头部字节进行匹配。这些数据仍然保留在缓冲区中。
*   **读取**: 当上层协议（如 `http.Server`）读取数据时，`FuseConn` 优先返回缓冲区中的数据，然后再从底层 Socket 读取。

### 2. FakeListener (虚拟监听器)

Go 标准库的 `http.Server` 和 `grpc.Server` 都依赖 `net.Listener` 接口来获取连接。为了复用这些标准组件，Fuse 实现了 `FakeListener`。

*   `FakeListener` 实现了 `net.Listener` 接口。
*   它不直接绑定端口，而是通过一个 Go Channel 接收来自 `Multiplexer` 的连接。
*   `Accept` 方法只是从 Channel 中取出已经完成握手和协议识别的连接。

## 支持的协议详解

### HTTP/1.1 & WebSocket & SSE

*   **HTTP/1.1**: 标准文本协议。通过匹配请求方法（`GET`, `POST`, `PUT`, `DELETE` 等）识别。
*   **WebSocket**: 握手阶段是标准的 HTTP/1.1 请求。连接被分发到 HTTP 引擎后，由 `wsx` 包处理 Upgrade 头并升级协议。
*   **SSE (Server-Sent Events)**: 本质是长连接的 HTTP 请求。由 `ssex` 包处理。

### HTTP/2 (gRPC)

*   **gRPC**: 默认使用 HTTP/2 协议。
*   **识别特征**: HTTP/2 连接建立时须发送固定的 24 字节前奏 `PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n`。Fuse 通过匹配此前奏来识别 gRPC 流量。

## 安全与超时

为了防止攻击者建立连接后不发送数据导致资源耗尽，`Multiplexer` 在接收连接后会设置一个短暂的读取超时（默认 3 秒）。如果在该时间内无法完成协议识别，连接将被强制关闭。

