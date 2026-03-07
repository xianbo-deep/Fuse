# 快速开始

本指南将帮助你快速搭建并运行一个基于 Fuse 框架的服务。

## 1. 环境准备

确保你的环境已安装 **Go 1.25** 或更高版本。

## 2. 安装

在你的项目目录下执行：

```bash
go get github.com/xianbo-deep/Fuse
```

*(注意：如果是本地开发，请确保 `go.mod` 中正确配置了模块路径)*

## 3. 编写第一个服务

创建一个 `main.go` 文件，写入以下代码。这个示例展示了如何启动一个 HTTP 服务并注册一个简单的路由。

```go
package main

import (
    "github.com/xianbo-deep/Fuse/fuse"
    "log"
)

func main() {
    // 1. 初始化 Fuse 引擎
    app := fuse.New()

    // 2. 注册 HTTP 路由
    // 使用核心上下文 fuse.Context (即 core.Ctx)
    app.HTTP().Get("/hello", func(c fuse.Context) fuse.Result {
        // 获取查询参数 ?name=World
        name := c.Query("name")
        if name == "" {
            name = "Fuse"
        }

        // 返回 JSON 响应
        // Success 方法自动封装标准响应结构: {code: 0, data: ...}
        return c.Success(fuse.H{
            "message": "Hello, " + name,
            "status":  "ok",
        })
    })

    // 3. 启动服务，默认监听 :8080
    // Run 方法会自动启动协议多路复用器，同时支持 HTTP/1.1 和 HTTP/2(gRPC)
    log.Println("Server is running at :8080")
    if err := app.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

## 4. 运行服务

在终端中执行：

```bash
go run main.go
```

你将看到终端输出服务启动日志。

## 5. 验证

你可以使用浏览器或 `curl` 工具来验证服务是否正常工作。

### 验证 HTTP 接口

打开终端，执行以下命令：

```bash
curl "http://localhost:8080/hello?name=Developer"
```

或者在浏览器访问 [http://localhost:8080/hello?name=Developer](http://localhost:8080/hello?name=Developer)。

**预期输出：**

```json
{
    "code": 0,
    "msg": "",
    "data": {
        "message": "Hello, Developer",
        "status": "ok"
    }
}
```

## 6. 下一步

现在你已经成功运行了第一个 Fuse 服务！

你可以尝试：
*   **添加更多路由**：使用 `app.HTTP().POST(...)` 等方法。
*   **使用中间件**：使用 `app.Use(...)` 添加日志 (`fuse.Logger`) 或恢复 (`fuse.Recovery`) 中间件。
*   **探索其他协议**：查看 API 文档了解如何使用 gRPC, WebSocket 或 Cron。

