package httpx

import (
	"Fuse/core"
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

type Ctx struct {
	ctx context.Context

	// http相关
	Writer  *core.ResponseWriterWrapper
	Request *http.Request

	values  map[string]any
	aborted bool

	// 处理器
	handlers []core.HandlerFunc
	index    int

	// 错误
	errs []error

	// 锁
	mu sync.RWMutex
}

// 实现core.Ctx接口
func (c *Ctx) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

func (c *Ctx) WithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
}

func (c *Ctx) Set(key string, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

func (c *Ctx) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *Ctx) Next() core.Result {
	c.index++
	if c.index < len(c.handlers) {
		if c.aborted {
			return core.Result{}
		}
		return c.handlers[c.index](c)
	}
	return core.Result{}
}

func (c *Ctx) resetHandlers(hs []core.HandlerFunc) {
	c.handlers = hs
	c.index = -1
}

func (c *Ctx) Abort() {
	c.aborted = true
	c.index = len(c.handlers)
}

func (c *Ctx) Copy() core.Ctx {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := &Ctx{
		ctx:      c.ctx,
		Writer:   c.Writer,
		Request:  c.Request,
		aborted:  c.aborted,
		handlers: c.handlers,
		index:    c.index,
		values:   make(map[string]any),
	}

	// 拷贝哈希表
	for k, v := range c.values {
		cp.values[k] = v
	}

	// 拷贝错误列表
	if c.errs != nil {
		cp.errs = make([]error, len(c.errs))
		copy(cp.errs, c.errs)
	}

	return cp
}

func (c *Ctx) Aborted() bool {
	return c.aborted
}

func (c *Ctx) Err(err error) {
	if err == nil {
		return
	}
	c.errs = append(c.errs, err)
}

func (c *Ctx) Error() error {

	if len(c.errs) == 0 {
		return nil
	}
	return c.errs[len(c.errs)-1]
}

func (c *Ctx) Errors() []error {
	// 返回拷贝，防止外部篡改
	out := make([]error, len(c.errs))
	copy(out, c.errs)
	return out
}

func NewCtx(ctx context.Context) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{ctx: ctx, values: make(map[string]any)}
}

// 设置状态码
func (c *Ctx) Status(code int) {
	if c.Writer == nil {
		return
	}
	if !c.Writer.Written() {
		c.Writer.WriteHeader(code)
	}
}

// 设置text/plain
func (c *Ctx) String(code int, s string) {
	h := c.Writer.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/plain; charset=utf-8")
	}
	c.Writer.WriteHeader(code)
	_, _ = c.Writer.Write([]byte(s))
}

// 设置json
func (c *Ctx) JSON(code int, v any) {
	h := c.Writer.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json; charset=utf-8")
	}
	// 写入状态码
	c.Status(code)

	// 执行序列化
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = c.Writer.Write(b)

}

// 渲染
func (c *Ctx) Render(res core.Result) {

	// 写入元数据
	for k, v := range res.Meta {
		c.Writer.Header().Set(k, v)
	}

	// 映射状态码
	status := res.GetHttpStatus()

	// 没设置则走框架默认
	if status == 0 {
		status = httpStatusFromBizCode(res.Code)
	}

	// 响应
	c.JSON(status, res)
}

// 获取路径参数
func (c *Ctx) Param(key string) string {
	val, ok := c.values["param-"+key]
	if !ok {
		return ""

	}
	return val.(string)
}

// 获取查询参数
func (c *Ctx) Query(key string) string {
	if c.Request == nil {
		return ""
	}
	return c.Request.URL.Query().Get(key)
}

func (c *Ctx) Success(data any) core.Result {
	return core.Result{
		Code: core.CodeSuccess,
		Data: data,
	}
}

func (c *Ctx) Fail(code int, msg string) core.Result {
	return core.Result{
		Code: code,
		Msg:  msg,
	}
}

// 状态码切换
func httpStatusFromBizCode(code int) int {
	switch code {
	case core.CodeSuccess:
		return http.StatusOK
	case core.CodeBadRequest:
		return http.StatusBadRequest
	case core.CodeUnauthorized:
		return http.StatusUnauthorized
	case core.CodeForbidden:
		return http.StatusForbidden
	case core.CodeNotFound:
		return http.StatusNotFound
	case core.CodeInternal:
		return http.StatusInternalServerError
	default:
		// 兜底：业务失败但没分类 -> 500
		if code != 0 {
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}
}
