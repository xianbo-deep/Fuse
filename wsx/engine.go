package wsx

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/httpx"

	"github.com/gorilla/websocket"
)

// Option 模式，用于用户进行链式自定义配置
type Option func(*Config)

// Config Websocket模块的配置。
//
// 待完善。
type Config struct {
	PingInterval       time.Duration // Ping 的时间间隔
	WaitTimeout        time.Duration // 服务端等待的超时时间
	ClientWriteTimeout time.Duration // 客户端写的超时时间
	AllowedOrigins     []string      // 允许跨域的域名列表
}

// WithPingInterval 设置 Ping 的间隔。
func WithPingInterval(d time.Duration) Option {
	return func(c *Config) {
		c.PingInterval = d
	}
}

// WithWaitTimeout 设置服务端等待超时时间。
func WithWaitTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.WaitTimeout = d
	}
}

// WithClientWriteTimeout 设置客户端写的超时时间。
func WithClientWriteTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.ClientWriteTimeout = d
	}
}

// WithAllowedOrigins 设置允许跨域的域名列表。
func WithAllowedOrigins(origins []string) Option {
	return func(c *Config) {
		c.AllowedOrigins = origins
	}
}

// DefaultConfig 默认的 websocket 配置。
func DefaultConfig() Config {
	return Config{
		// PingInterval 每次 ping 间隔时间: 54s
		PingInterval: time.Second * 54,
		// WaitTimeout 等待客户端响应超时时间: 60s
		WaitTimeout: time.Second * 60,
		// ClientWriteTimeout 客户端写超时时间
		ClientWriteTimeout: 10 * time.Second,
		// AllowedOrigins 允许跨域的域名列表
		AllowedOrigins: []string{},
	}
}

// WsHandlerFunc 是 func(ctx *WsContext) error 的类型别名，用户需要传入 [WsHandlerFunc] 以进行 Http 到 Websocket 的协议升级。
type WsHandlerFunc func(ctx *WsContext) error

// Upgrade 协议升级器，将你传入的 [WsHandlerFunc] 转换成 [core.HandlerFunc]，供 Http 模块调用。
func Upgrade(wshandlerFunc WsHandlerFunc, opts ...Option) core.HandlerFunc {
	var cfg = DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
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
		pump := NewPump(conn, done, writeChan, mu, cfg)

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
		// 开启一个协程跑心跳检测
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
						conn.Close()
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
