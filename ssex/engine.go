package ssex

import (
	"errors"
	"net/http"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/httpx"
)

// SSEHandlerFunc 是 func(c core.Ctx, stream *Stream) error 的类型别名，用户需要返回这个类型方法用于 Http 到 SSE 的升级。
type SSEHandlerFunc func(c core.Ctx, stream *Stream) error

// Upgrade 协议升级器，用户将用户传入的 [SSEHandlerFunc] 转换成 HTTP 模块需要的 [core.HandlerFunc]。
func Upgrade(sseHandler SSEHandlerFunc) core.HandlerFunc {
	return func(c core.Ctx) core.Result {
		// 类型断言
		ctx, ok := c.(*httpx.Ctx)
		if !ok {
			return c.Fail(core.CodeBadRequest, "can not upgrade to sse without http request")
		}

		// 设置SSE响应头
		ctx.SetSSEHeader()
		ctx.Status(http.StatusOK)
		ctx.Writer.Flush()

		// 初始化stream实例
		stream := NewStream(ctx)

		// 启动守护进程 监听客户端是否断连
		go stream.startHeartPingPong()

		// 执行业务逻辑
		if err := sseHandler(ctx, stream); err != nil {
			if !errors.Is(err, errClosed) {
				c.FailWithError(err)
			}
		}

		return core.Result{}
	}
}
