package wsx

import (
	"Fuse/core"
	"Fuse/httpx"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type WsHandlerFunc func(c core.Ctx, conn *websocket.Conn) error

// 转换器 把用户写的WsHandlerFunc转换成HandlerFunc
func Upgrade(wshandlerFunc WsHandlerFunc, allowedOrigins ...string) core.HandlerFunc {
	// 获取升级器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if len(allowedOrigins) == 0 {
				return true
			}
			// 获取请求头
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}

			// 校验ip
			for _, allowed := range allowedOrigins {
				if origin == allowed || strings.Contains(origin, allowed) {
					return true
				}
			}
			return false
		},
	}
	return func(c core.Ctx) core.Result {
		// 类型断言
		ctx, ok := c.(*httpx.Ctx)
		if !ok {
			return c.Fail(core.CodeBadRequest, "can not upgrade to websocket without http request")
		}
		// 获取ResponseWriter和Request
		w := ctx.Writer
		r := ctx.Request

		// 升级
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return c.Fail(core.CodeInternal, err.Error())
		}
		defer conn.Close()

		// 执行业务函数
		if err := wshandlerFunc(c, conn); err != nil {
			return c.Fail(core.CodeInternal, err.Error())
		}
		return c.Success(nil)
	}
}
