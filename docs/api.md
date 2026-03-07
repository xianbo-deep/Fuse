# API 参考文档

本文档详细介绍了 Fuse 框架的核心 API，包括入口对象、上下文抽象以及统一响应结构。

## Fuse (入口对象)

`Fuse` 结构体是框架的统一入口，负责管理底层的 HTTP、gRPC 和 Cron 引擎，并提供统一的中间件挂载机制。

### 获取实例

*   `New() *Fuse`: 创建一个基础的 Fuse 实例。
*   `Default() *Fuse`: 创建一个带有默认配置（如默认恢复中间件等，具体取决于底层实现）的 Fuse 实例。

### 引擎访问

*   `HTTP() *httpx.Engine`: 获取 HTTP 引擎实例，用于注册 HTTP 路由。
*   `GRPC() *grpcx.Engine`: 获取 gRPC 引擎实例，用于注册 gRPC 服务。
*   `CRON() *cronx.Engine`: 获取 Cron 引擎实例，用于添加定时任务。

### 全局操作

*   `Use(mws ...core.HandlerFunc)`: 注册全局中间件。这些中间件会被自动分发到底层的 HTTP、gRPC 和 Cron 引擎中，确保所有协议的请求都能经过这些中间件的处理。
*   `Run(addr string) error`: 启动服务。该方法会初始化协议分发器 (CMUX)，在指定端口（默认为 `:8080`）监听 TCP 连接，并根据流量特征将连接分发给 HTTP 或 gRPC 引擎。同时也会启动 Cron 调度器。

## Context (上下文)

`fuse.Context` (即 `core.Ctx`) 是框架的核心抽象，用于屏蔽不同协议（HTTP, gRPC, WebSocket）之间的差异，提供统一的操作接口。

### 核心流程控制

*   `Next() Result`: 继续执行中间件链中的下一个处理函数。
*   `Abort()`: 终止中间件链的执行，后续的中间件将不再被调用。
*   `Aborted() bool`: 检查当前上下文是否已被标记为终止。

### 请求数据获取

*   `Param(key string) string`: 获取路径参数（例如 `/user/:id` 中的 `id`）。
*   `Query(key string) string`: 获取查询字符串参数（例如 `?name=alice` 中的 `name`）。
*   `Bind(v any) error`: 将请求体数据绑定到指定的结构体 `v` 中。支持根据 Content-Type 自动解析（如 JSON）。

### 上下文管理

*   `Set(key string, val any)`: 在上下文中存储键值对数据，通常用于在中间件之间传递信息（如用户 ID）。
*   `Get(key string) (any, bool)`: 从上下文中获取存储的数据。
*   `Context() context.Context`: 获取底层的标准库 `context.Context` 对象。
*   `WithContext(ctx context.Context)`: 替换底层的标准库 `context.Context` 对象。
*   `Copy() Ctx`: 创建当前上下文的副本，用于在协程中安全使用。

### 响应构建

*   `Success(data any) Result`: 构建一个表示成功的响应结果，标准状态码为 0。
*   `Fail(code int, msg string) Result`: 构建一个表示失败的响应结果，包含自定义错误码和错误信息。
*   `FailWithError(err error) Result`: 根据 error 对象构建失败响应。

## Result (统一响应)

`fuse.Result` (即 `core.Result`) 用于封装跨协议的响应数据。

*   `Code int`: 业务状态码（0 通常表示成功）。
*   `Msg string`: 提示信息。
*   `Data any`: 响应数据载荷。
*   `Meta map[string]string`: 元数据信息。

### 链式操作

Result 对象支持链式调用以设置特定协议的状态码：

*   `WithHttpStatus(status int) Result`: 设置 HTTP 响应的状态码（如 200, 404, 500）。
*   `WithGrpcStatus(status int) Result`: 设置 gRPC 响应的状态码。
*   `WithMsg(msg string) Result`: 覆盖响应消息。
*   `WithData(data any) Result`: 覆盖响应数据。

## SSE (Server-Sent Events)

`ssex` 包提供了对 Server-Sent Events 的支持，允许服务器向客户端主动推送事件流。

### 升级处理

*   `ssex.Upgrade(handler ssex.SSEHandlerFunc) core.HandlerFunc`: 将 HTTP 请求升级为 SSE 连接。该函数返回一个标准中间件，可直接在路由中使用。
    *   `handler`: 处理 SSE 连接的回调函数，签名 `func(c core.Ctx, stream *ssex.Stream) error`。

### Stream (数据流)

`ssex.Stream` 对象用于向客户端推送事件。

*   `Send(event string, data any) error`: 发送一个 SSE 事件。
    *   `event`: 事件名称。如果不指定，客户端通常默认为 `message`。
    *   `data`: 事件数据。如果是结构体或 map，会自动序列化为 JSON 格式。

## WebSocket

`wsx` 包提供了对 WebSocket 协议的支持，包含自动心跳管理和消息泵机制。

### 升级处理

*   `wsx.Upgrade(handler wsx.WsHandlerFunc, config ...wsx.WebsocketConfig) core.HandlerFunc`: 将 HTTP 请求升级为 WebSocket 连接。
    *   `handler`: 处理 WebSocket 消息的回调函数，签名 `func(ctx *wsx.WsContext) error`。
    *   `config`: 可选配置，用于设置心跳间隔 (`PingInterval`)、读写超时 (`WaitTimeout`) 和允许跨域的源 (`AllowedOrigins`)。

### WsContext (WebSocket 上下文)

`wsx.WsContext` 封装了 WebSocket 连接和当前接收到的消息。

*   `BindJSON(v any) error`: 将当前接收到的消息数据 (JSON 格式) 绑定到结构体 `v`。
*   `Send(data []byte)`: 发送原始字节数据给客户端。
*   `SendJSON(v any) error`: 将数据 `v` 序列化为 JSON 并发送给客户端。
*   `Close() error`: 关闭 WebSocket 连接。
*   `Data []byte`: 直接访问当前消息的原始字节数据。
*   `MsgType int`: 当前消息的消息类型 (Text/Binary)。

## HTTP (httpx)

`httpx` 包实现了基于 Radix Tree 的高性能 HTTP 路由引擎。

### Engine (HTTP 引擎)

*   `Group(prefix string, mws ...core.HandlerFunc) *httpx.RouterGroup`: 创建一个路由组。
*   `Use(mws ...core.HandlerFunc)`: 注册全局中间件。

### RouterGroup (路由组)

路由组支持标准的 HTTP 方法注册，并可嵌套使用。

*   `GET(pattern string, handler ...core.HandlerFunc)`
*   `POST(pattern string, handler ...core.HandlerFunc)`
*   `PUT(pattern string, handler ...core.HandlerFunc)`
*   `DELETE(pattern string, handler ...core.HandlerFunc)`
*   `Use(mws ...core.HandlerFunc)`: 为该组注册中间件。

## gRPC (grpcx)

`grpcx` 包提供了 gRPC 服务的集成，支持统一的中间件机制。

### Engine (gRPC 引擎)

*   `Server() *grpc.Server`: 获取底层的 `grpc.Server` 实例，用于注册 PB 生成的服务。
*   `Use(mws ...core.HandlerFunc)`: 注册 gRPC 全局拦截器（以中间件形式）。

该引擎会自动注入一元 (Unary) 和流式 (Stream) 拦截器，将 gRPC 请求转换为 `core.Ctx` 上下文，从而复用通用的中间件逻辑。

## Cron (cronx)

`cronx` 包封装了定时任务调度，支持中间件装饰。

### Engine (Cron 引擎)

*   `AddFunc(spec string, handler core.HandlerFunc) (cron.EntryID, error)`: 添加一个定时任务。
    *   `spec`: Cron 表达式 (支持秒级)。
    *   `handler`: 任务执行逻辑。
*   `Start()`: 启动调度器。
*   `Stop() context.Context`: 优雅停止调度器。

## Middleware (中间件)

`middleware` 包提供了一系列开箱即用的通用中间件。

*   `Defaults()`: 返回默认的中间件集合，包含 Recovery, RequestID 和 Logger。
*   `Recovery()`: 异常恢复中间件，防止 panic 导致服务崩溃。
*   `RequestID()`: 为每个请求注入唯一的 Request ID。
*   `Logger()`: 请求日志记录中间件。
