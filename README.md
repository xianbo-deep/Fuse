# Fuse

**一个轻量级、协议无关的 Go 服务端框架骨架**

<img alt="Go" src="https://img.shields.io/badge/Go-1.24.0-00ADD8?logo=go&logoColor=white" />



Fuse 旨在提供统一的服务端开发体验。通过高度抽象的 Context + Middleware 模型，它将 HTTP、gRPC、Cron、WebSocket、SSE 等多种协议的处理流程归一化，让开发者能以一致的范式构建服务。

代码库目前采用 Core + Facade 的架构模式，通过 `fuse` 包作为统一入口，屏蔽底层复杂的依赖关系。

## 核心设计：统一抽象

Fuse 的核心理念是定义一套通用的接口标准，所有具体协议的实现都围绕这套标准展开。

*   **抽象契约 (Core)**：定义了框架最基础的接口（如 Ctx、Result），所有协议适配层（httpx、wsx 等）均依赖于此。这确保了不同协议之间互不干扰，实现了架构的清晰分层。
*   **统一门面 (Facade)**：用户只需关注 `fuse` 包提供的 API，无需关心底层的具体实现和复杂的依赖关系。

这种设计使得框架既保持了内部组件的低耦合，又为用户提供了简洁一致的开发体验。

## 协议支持

Fuse 通过连接多路复用（CMUX）技术，支持在单端口上运行多种协议。详细协议实现请参考 [协议说明](./docs/protocols.md)。

*   **HTTP/1.1**: 内置高性能路由，支持 RESTful API。
*   **HTTP/2 (gRPC)**: 无缝集成的 RPC 支持，与 HTTP 服务共存。
*   **WebSocket**: 提供升级处理、消息泵及心跳机制。
*   **SSE**: 支持 Server-Sent Events 推送。
*   **Cron**: 集成的定时任务调度。

## 目录结构

```text
Fuse
+---core            # 核心抽象层 (Ctx, Result)
+---cronx           # 定时任务适配器
+---docs            # 开发文档
+---fuse            # 用户统一入口 (Facade)
+---grpcx           # gRPC 协议适配器
+---httpx           # HTTP 协议适配器
+---middleware      # 通用中间件集合
+---mux             # 协议多路复用器 (CMUX)
+---ssex            # SSE 协议适配器
\---wsx             # WebSocket 协议适配器
```

## 快速开始

### 安装

```bash
go get github.com/xianbo-deep/Fuse
```

### 示例代码

```go
package main

import "github.com/xianbo-deep/Fuse/fuse"

func main() {
    // 1. 初始化引擎
    app := fuse.New()

    // 2. 注册路由 (使用 fuse.Context)
    app.HTTP().Get("/ping", func(c fuse.Context) fuse.Result {
        return c.Success(fuse.H{"message": "pong"})
    })

    // 3. 启动
    if err := app.Run(":8080"); err != nil {
        panic(err)
    }
}
```

## 未来规划

- [x] 完善 gRPC 协议支持
- [x] 优化中间件链内存分配
- [x] 实现优雅停机
- [x] WebSocket 协议适配 (wsx) 及心跳检测
- [x] HTTP 路由优先级控制及 Radix Tree 实现
- [x] HTTP 路由分组
- [x] SSE 协议适配 (ssex) 及 Keep-Alive
- [x] 实现单端口协议分发 (CMUX)
- [x] WebSocket 消息泵 (Message Pump)
- [ ] CMUX 实现协议树
- [x] CMUX 实现协议动态注册
- [x] 增加更多通用中间件 (Limit, Trace)
- [x] WebSocket JSON 解析
- [x] HTTP 参数校验 (Validator)
- [ ] 完善单元测试与 Benchmarks 对比
- [x] 完善 GoDoc
- [ ] 补充相关文档

更多详细设计和用法请阅读 [docs](./docs) 目录下的文档。

