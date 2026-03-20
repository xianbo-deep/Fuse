package ssex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xianbo-deep/Fuse/httpx"
)

// Stream 是 SSE 模块暴露给用户的流式上下文
type Stream struct {
	ctx    *httpx.Ctx
	reqCtx context.Context
	mu     sync.Mutex
	done   chan struct{}
	cfg    Config
}

// NewStream 返回 [Stream] 实例。
func NewStream(ctx *httpx.Ctx, cfg Config) *Stream {
	return &Stream{
		ctx:    ctx,
		reqCtx: ctx.Request.Context(),
		done:   make(chan struct{}),
		cfg:    cfg,
	}
}

// errClosed 连接被客户端关闭。
var errClosed = errors.New("connection closed by client")

// Send 从服务端发送数据给客户端
func (s *Stream) Send(event string, data any) error {
	// 进行心跳检测 客户端断连直接返回错误
	select {
	case <-s.reqCtx.Done():
		return errClosed
	case <-s.done:
		return errClosed
	default:
	}

	// 格式化数据
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		dataStr = string(b)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 写入event
	if event != "" {
		_, err := s.ctx.Writer.Write([]byte(fmt.Sprintf("event: %s\n", event)))
		if err != nil {
			return err
		}
	}

	// 写入data 多行写入
	if dataStr != "" {
		dataline := strings.Split(dataStr, "\n")
		for _, line := range dataline {
			_, err := s.ctx.Writer.Write([]byte(fmt.Sprintf("data: %s\n", line)))
			if err != nil {
				return err
			}
		}

	}
	// 补充最后的换行符
	_, err := s.ctx.Writer.Write([]byte("\n"))
	if err != nil {
		return err
	}

	// 每次推送完数据堆缓冲区进行刷新
	s.ctx.Writer.Flush()

	return nil
}

// startHeartPingPong 执行心跳检测，会单独开启一个协程执行此任务，因此要防止资源泄漏。
//
// 发送 ping 时添加写锁，防止流式传输信息与心跳检测任务冲突，保证线程安全。
//
// 使用定时器每隔10s执行一次 ping ，防止 TCP 连接无字节传输时网关（如Nginx）对连接进行掐断。
//
// 对用户请求的 Context 的 Done 通道进行监听，在用户断连时可以停止心跳检测，防止资源泄漏。
func (s *Stream) startHeartPingPong() {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.reqCtx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			_, err := s.ctx.Writer.Write([]byte(":ping\n\n")) // 在SSE中这是注释
			// 客户端断连
			if err != nil {
				s.mu.Unlock()
				close(s.done)
				return
			}
			s.ctx.Writer.Flush()
			s.mu.Unlock()
		}
	}
}
