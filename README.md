<div align="center">
  <h1>Fuse</h1>
  <p>
    <strong>一个轻量级、协议无关的 Go 服务端框架骨架</strong>
  </p>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" />
</div>

<br />

**Fuse** 是一个旨在统一服务端开发体验的 Go 框架。

它通过 **Core Abstraction (核心抽象)** 设计模式，将 HTTP、gRPC 和 Cron 等不同协议的处理流程统一抽象为 **"Context + Middleware Chain"** 模型。这不仅解耦了底层协议与业务逻辑，还优雅地解决了 Go 项目中常见的 **循环引用 (Circular Dependencies)** 问题。

详细文档请移步 [docs](./docs) 目录。

## 核心特性

- **协议统一**：无论是 HTTP 请求、gRPC 调用还是定时任务 (Cron)，都使用同一套中间件和 Handler 逻辑。
- **Core 抽象层**：通过独立的 `core` 包定义接口，打破包依赖环，让模块间交互更顺畅。
- **模块化设计**：
  - `fuse`：统一门面 (Facade)，提供极简 API。
  - `httpx` / `grpcx` / `cronx`：各协议的具体实现适配器。
  - `middleware`：开箱即用的通用拦截器 (Logger, Recovery, etc.)。

## 目录结构

```text
.
├── core/           # [核心] 顶层抽象 (Ctx, Result, Handler)，解决循环依赖的关键
├── fuse/           # [门面] 用户使用的统一入口，类型别名导出
├── httpx/          # [HTTP] 基于 net/http 的适配层
├── grpcx/          # [gRPC] gRPC 适配层
├── cronx/          # [Cron] 定时任务适配层
├── middleware/     # [插件] 通用中间件
└── docs/           # [文档] 详细开发指南
```

## 快速开始

### 安装

```bash
go get github.com/xianbo-deep/fuse
```

### Hello World

```go
package main

import (
    "Fuse/fuse"
)

func main() {
    // 1. 初始化引擎
    app := fuse.New()

    // 2. 注册 HTTP 路由
    app.HTTP().Get("/ping", func(c fuse.Context) fuse.Result {
        return c.Success(fuse.H{"message": "pong"})
    })
    
    // 3. 注册定时任务 (可选)
    // app.Cron().AddFunc("@every 10s", func(c fuse.Context) fuse.Result {
    //     println("tick")
    //     return c.Success(nil)
    // })

    // 4. 启动服务
    if err := app.Run(":8080"); err != nil {
        panic(err)
    }
}
```

更多使用方式请参考 [Quick Start](./docs/quickstart.md).

## 文档

- [API 参考](./docs/api.md)
- [协议适配指南](./docs/protocols.md)
- [快速开始](./docs/quickstart.md)
- [设计模式](./docs/designpatterns.md)

## 未来规划

- [ ] 完善 gRPC 协议支持
- [x] 新增 WebSocket 协议适配 (wsx)
- [ ] 增加更多通用中间件 (限流、鉴权、链路追踪)
- [ ] 完善单元测试与覆盖率
- [ ] 完善GoDoc
- [ ] 完成HTTP路由分组和字段校验
- [ ] 补充相关文档
- [ ] 写压测代码，做Benchmark测试，和常见框架做对比
