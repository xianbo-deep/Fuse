<div align="center">
  <h1>Fuse</h1>
  <p>
    <strong>一个轻量级、可扩展的 Go 服务端框架骨架</strong>
  </p>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" />
</div>

<br />

**Fuse** 的核心设计理念是将 **HTTP** 和 **gRPC** (规划中) 的处理流程统一抽象为一套 **"Context + Middleware Chain"** 模型。通过定义通用的 `core` 接口，解耦底层协议与业务逻辑，让开发者能够以一致的方式编写中间件和业务处理器。

## 特性

- **统一核心抽象 (`core`)**：定义了通用的 `Ctx` 接口和 `HandlerFunc`，通过类型别名在 `fuse` 主包中统一暴露。
- **模块化设计**：
  - `fuse`：统一入口，提供简化的 API 和生命周期管理。
  - `httpx`：基于标准库 `net/http` 的封装，提供路由、参数绑定等能力。
  - `middleware`：开箱即用的通用中间件（Recovery, Logger, RequestID）。
  - `grpcx`：(WIP) 预留的 gRPC 扩展层。
- **中间件机制**：支持洋葱模型（Onion Architecture），轻松实现拦截、鉴权、日志等功能，且协议无关。
- **标准化响应**：内置 `Result` 结构，强制统一 API 返回格式。

## 目录结构

```text
.
├── core/           # [核心] 框架顶层抽象 (Ctx接口, Result定义, Handler定义)
├── fuse/           # [入口] 框架统一入口，类型别名导出
├── httpx/          # [HTTP] HTTP 协议实现 (Engine, Router, Context实现)
├── grpcx/          # [gRPC] gRPC 协议实现 (开发中)
├── middleware/     # [中间件] 通用中间件集合
├── test/           # [示例] 示例代码
├── go.mod          # 依赖管理
└── README.md       # 项目文档
```

## 快速开始

### 环境要求

- Go 1.25+

### 运行示例

```go
package main

import (
    "Fuse/fuse"
    "Fuse/middleware"
)

func main() {
    // 1. 初始化引擎
    app := fuse.New()

    // 2. 注册全局中间件
    app.Use(middleware.Defaults()...)

    // 3. 获取 HTTP 路由组并注册路由
    httpSrv := app.HTTP()
    httpSrv.Get("/ping", func(c fuse.Context) fuse.Result {
        // 使用 fuse.H 简化 Map 定义
        return c.Success(fuse.H{"message": "pong"})
    })

    // 4. 启动服务
    if err := app.Run(":8080"); err != nil {
        panic(err)
    }
}
```

运行服务：

```bash
go run ./test/main.go
# 访问 http://localhost:8080/ping
```

## 设计理念

Fuse 采用**核心接口 (`core`)** 与**具体实现 (`httpx`, `grpcx`)** 分离的策略。

- 用户主要与 `fuse` 包交互，该包通过类型别名（Type Alias）导出了 `core` 中的核心接口（如 `Context`, `Result`）。
- 这种设计即解决了包之间的循环依赖问题，又保证了对外 API 的简洁性。用户无需关心底层的 `core` 包。

## 规划中

- [x] **路由增强**：引入 Trie 树路由，支持路径参数和通配符。
- [ ] **gRPC 支持**：完善 `grpcx`，实现 gRPC Interceptor 到 Fuse Middleware 的适配。
- [ ] **配置管理**：集成统一的配置加载模块。
