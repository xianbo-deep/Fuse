package wsx

import (
	"Fuse/core"
	"Fuse/httpx"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsHandlerFunc func(ctx *WsContext) error

type WebsocketConfig struct {
	PingInterval   time.Duration // 发送心跳的间隔
	WaitTimeout    time.Duration // 等待的超时时间
	AllowedOrigins []string
}

var defaultWebsocketConfig = WebsocketConfig{
	PingInterval: time.Second * 54,
	WaitTimeout:  time.Second * 60,
}

// 转换器 把用户写的WsHandlerFunc转换成HandlerFunc
func Upgrade(wshandlerFunc WsHandlerFunc, config ...WebsocketConfig) core.HandlerFunc {
	var cfg WebsocketConfig
	if len(config) != 0 {
		cfg = config[0]
	}
	// 填充默认值
	if cfg.PingInterval == 0 {
		cfg.PingInterval = defaultWebsocketConfig.PingInterval
	}
	if cfg.WaitTimeout == 0 {
		cfg.WaitTimeout = defaultWebsocketConfig.WaitTimeout
	}

	// 获取升级器
	upgrader := websocket.Upgrader{
		// 跨域校验
		CheckOrigin: func(r *http.Request) bool {
			if len(cfg.AllowedOrigins) == 0 {
				return true
			}
			// 获取请求头
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}

			// 校验ip
			for _, allowed := range cfg.AllowedOrigins {
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
		// 获取ResponseWriterWrapper和Request
		w := ctx.Writer
		r := ctx.Request

		// 升级
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return c.Fail(core.CodeInternal, err.Error())
		}

		// 用于监听客户端是否断连
		done := make(chan struct{}, 1)

		// 锁
		mu := &sync.Mutex{}

		var once sync.Once

		// 写泵通道
		writeChan := make(chan []byte, 256)

		// 获取泵对象
		pump := NewPump(conn, done, writeChan, mu)

		// 起一个协程 开启写泵
		go pump.WritePump()

		// 设置读取超时
		e := conn.SetReadDeadline(time.Now().Add(cfg.WaitTimeout))
		if e != nil {
			return c.Fail(core.CodeBadRequest, e.Error())
		}

		// 检测逻辑
		conn.SetPongHandler(func(pong string) error {
			// 重新设置超时时间
			return conn.SetReadDeadline(time.Now().Add(cfg.WaitTimeout))
		})

		// 开启一个协程跑心跳检测
		defer func() {
			once.Do(func() {
				select {
				case <-done:
				default:
					close(done)
				}
				close(writeChan)
				conn.Close()
			})
		}()
		go func() {
			// 创建定时器
			ticker := time.NewTicker(cfg.PingInterval)
			defer ticker.Stop()

			for {
				select {
				// 监听管道判断业务是否结束
				case <-done:
					return
				// 执行心跳检测
				case <-ticker.C:
					// 设置超时时间 防止协程卡死造成内存泄漏 需要加锁
					mu.Lock()
					err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(cfg.WaitTimeout))
					mu.Unlock()
					if err != nil {
						return
					}
				}
			}
		}()

		for {
			conn.SetReadDeadline(time.Now().Add(cfg.WaitTimeout))

			msgType, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			wsctx := NewWsContext(c, conn, msgType, data, writeChan)

			// 执行业务函数
			if err = wshandlerFunc(wsctx); err != nil {
				break
			}
		}

		return c.Success(nil)
	}
}
