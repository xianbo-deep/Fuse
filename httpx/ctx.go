package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	"github.com/xianbo-deep/Fuse/core"
)

// validate 结构体参数校验器。
var validate = validator.New()

// Ctx
type Ctx struct {
	ctx context.Context

	// 底层引擎
	engine *Engine

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

	// 查询缓存
	queryCache url.Values
}

// 实现core.Ctx接口

// Context
func (c *Ctx) Context() context.Context {
	return c.ctx
}

func (c *Ctx) WithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.ctx = ctx
}

func (c *Ctx) Set(key string, value any) {
	c.values[key] = value
}

func (c *Ctx) Get(key string) (any, bool) {
	v, ok := c.values[key]

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

func (c *Ctx) resetHandlers() {
	c.index = -1
}

func (c *Ctx) Abort() {
	c.aborted = true
	c.index = len(c.handlers)
}

func (c *Ctx) Copy() core.Ctx {
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

func NewCtx(ctx context.Context, engine *Engine) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{ctx: ctx, values: make(map[string]any), handlers: make([]core.HandlerFunc, 0, 64), engine: engine}
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

// Bind 读取数据到结构体中。
//
// 优先级：JSON > Param > Query
//
// 已经设置的值不会被覆盖。
func (c *Ctx) Bind(v any) error {
	// 获取传入数据的值
	val := reflect.ValueOf(v)
	// 必须要是指针并且指向结构体
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("bind: expected pointer to struct")
	}

	if err := c.bindJSON(v); err != nil {
		return err
	}
	if err := c.bindQuery(v); err != nil {
		return err
	}
	if err := c.bindParam(v); err != nil {
		return err
	}
	return validateStruct(v)
}

// 进行 JSON 参数解析
func (c *Ctx) bindJSON(v any) error {
	// 尝试解析JSON
	if c.Request.Body != nil && strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		// 读取原始字节
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		// 重新放回请求头
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// JSON格式错误
		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return err
		}
	}
	return nil
}

func (c *Ctx) bindParam(v any) error {
	// 获取传入数据的值
	val := reflect.ValueOf(v)

	// 解析结构体中的路由和查询参数
	elem := val.Elem()
	typ := elem.Type()

	for i := 0; i < typ.NumField(); i++ {
		// 字段信息
		field := typ.Field(i)
		// 字段实际值
		fieldVal := elem.Field(i)

		// 小写字段不可渲染
		if !fieldVal.CanSet() {
			continue
		}

		// 已有的值不覆盖
		if !fieldVal.IsZero() {
			continue
		}

		var valStr string
		if paramTag := field.Tag.Get("param"); paramTag != "" {
			valStr = c.Param(paramTag)
		}
		if valStr == "" {
			continue
		}
		if err := setField(fieldVal, valStr, field.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *Ctx) bindQuery(v any) error {
	// 获取传入数据的值
	val := reflect.ValueOf(v)

	// 解析结构体中的路由和查询参数
	elem := val.Elem()
	typ := elem.Type()

	for i := 0; i < typ.NumField(); i++ {
		// 字段信息
		field := typ.Field(i)
		// 字段实际值
		fieldVal := elem.Field(i)

		// 小写字段不可渲染
		if !fieldVal.CanSet() {
			continue
		}

		// 已有的值不覆盖
		if !fieldVal.IsZero() {
			continue
		}

		var valStr string

		if queryTag := field.Tag.Get("query"); queryTag != "" {
			valStr = c.Query(queryTag)
		}
		if valStr == "" {
			continue
		}
		if err := setField(fieldVal, valStr, field.Name); err != nil {
			return err
		}
	}
	return nil
}

// 进行参数验证
func validateStruct(v any) error {
	return validate.Struct(v)
}

func setField(fieldVal reflect.Value, valStr string, fieldName string) error {
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(valStr)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s parse int error: %v", fieldName, err)
		}
		fieldVal.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s parse uint error: %v", fieldName, err)
		}
		fieldVal.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return fmt.Errorf("field %s parse float error: %v", fieldName, err)
		}
		fieldVal.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("field %s parse bool error: %v", fieldName, err)
		}
		fieldVal.SetBool(boolVal)
	}
	return nil
}

// 获取路径参数
func (c *Ctx) Param(key string) string {
	val, ok := c.Get("param-" + key)
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
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
		return c.queryCache.Get(key)
	}
	return c.queryCache.Get(key)
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

func (c *Ctx) FailWithError(err error) core.Result {
	if err == nil {
		return c.Success(nil)
	}

	// 类型断言
	if bizErr, ok := err.(*core.BizError); ok {
		res := c.Fail(bizErr.Code, bizErr.Msg)
		if bizErr.HttpStatus != 0 {
			res = res.WithHttpStatus(bizErr.HttpStatus)
		}
		return res
	}
	return c.Fail(core.CodeInternal, err.Error()).WithHttpStatus(http.StatusInternalServerError)
}

// 重置上下文状态 清空遗留信息
func (c *Ctx) reset() {
	c.ctx = nil
	c.index = -1
	c.aborted = false
	c.Request = nil
	c.Writer = nil

	// 清空数据但保留底层容量
	clear(c.values)
	clear(c.queryCache)
	clear(c.errs)
	clear(c.handlers)

	c.handlers = c.handlers[:0]
	c.errs = c.errs[:0]
}

// 设置SSE响应头
func (c *Ctx) SetSSEHeader() {
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
}

func (c *Ctx) ClientIP() string {
	// 获取底层TCP连接的IP并进行分割
	remoteIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		remoteIP = c.Request.RemoteAddr
	}
	// 判断是否是可信代理
	if !c.engine.IsTrustedProxy(remoteIP) {
		return remoteIP
	}
	// 从 X-Forwarded-For 获取
	xForwardedFor := c.Request.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return ip
			}
		}
	}
	// 从 X-Real-IP 获取
	xRealIP := c.Request.Header.Get("X-Real-Ip")
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	// 从 TCP 连接获取
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}

	return ip
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
