# 设计模式

Fuse 框架在实现过程中应用了多种核心设计模式，以保证代码的灵活性与可扩展性。以下是其中最关键的几种模式：

## 1. 结构型模式

### 1.1 责任链模式 (Chain of Responsibility) & 装饰器模式 (Decorator Pattern)
**中间件 (Middleware)** 机制是这两种模式的典型结合。
- **核心思想**: 允许在请求处理流程中动态添加功能（如日志、鉴权、错误恢复），而无需修改业务逻辑。
- **Fuse 实现**: 
    - 采用洋葱模型，所有中间件和业务逻辑都符合 `core.HandlerFunc` 接口。
    - 通过 `Context.Next()` 方法控制链条的执行，请求在链中层层传递，响应则反向穿透。
- **代码位置**: `core/chain.go` 定义了链式调用逻辑，`middleware/` 目录包含了具体实现。

### 1.2 组合模式 (Composite Pattern)
**路由组 (RouterGroup)** 的设计体现了组合模式。
- **核心思想**: 将可以包含子路由组的路由组视为一个整体，构建树状层级结构。
- **Fuse 实现**: `httpx.RouterGroup` 可以无限嵌套，子组继承父组的前缀和中间件配置，实现了统一的路由管理。
- **代码位置**: `httpx/routergroup.go`。

### 1.3 适配器模式 (Adapter Pattern)
**统一接口适配**。
- **核心思想**: 将不同协议（HTTP, gRPC, Cron）的差异抹平，统一对接至框架内部接口。
- **Fuse 实现**: 
    - 框架定义了统一的 `core.Context` 和 `core.HandlerFunc`。
    - `httpx.Engine` 将 `net/http` 的请求适配为框架上下文。
    - `grpcx.Engine` 和 `cronx.Engine` 同样将各自的触发源适配为统一的处理流程。
- **代码位置**: `fuse/fuse.go`, `httpx/engine.go`。

## 2. 行为型 & 创建型模式

### 2.1 策略模式 (Strategy Pattern)
**多路复用连接分发**。
- **核心思想**: 根据不同的条件选择不同的算法或策略。
- **Fuse 实现**: `mux.Multiplexer` 根据连接的首字节特征（HTTP/1.1 vs HTTP/2），动态选择将其分发给 HTTP 监听器还是 gRPC 监听器。
- **代码位置**: `mux/multiplexer.go`。

### 2.2 选项模式 (Functional Options Pattern)
**灵活配置**。
- **核心思想**: 使用函数或变长参数来配置复杂对象，提供默认值并支持按需修改。
- **Fuse 实现**: 在 `grpcx.New` 和 `wsx.Upgrade` 等初始化方法中，使用 Option 模式传递配置参数，避免了构造函数参数过多的问题。

## 3. 并发模式

### 3.1 对象池模式 (Object Pool Pattern)
- **核心思想**: 复用昂贵的对象资源，减少内存分配和垃圾回收（GC）压力。
- **Fuse 实现**: 使用 `sync.Pool` 复用 `Context` 对象。请求结束后重置上下文并放回池中，极大提高了高并发场景下的性能。
- **代码位置**: `httpx/engine.go`, `grpcx/engine.go`。
