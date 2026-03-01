package core

/* 简短类型别名 */
type H map[string]any

/*
跨协议返回值
*/
type Result struct {
	// 0表示成功 其它表示失败
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`

	// 跨协议元信息
	Meta map[string]string `json:"meta,omitempty"`

	// http状态码
	httpStatus int

	// grpc状态码
	grpcStatus int
}

func Success(data any) Result {
	return Result{
		Code: 0,
		Data: data,
	}
}

func Fail(code int, msg string) Result {
	return Result{
		Code: code,
		Msg:  msg,
	}
}

func (r Result) WithMsg(msg string) Result {
	r.Msg = msg
	return r
}

func (r Result) WithMeta(k, v string) Result {
	if r.Meta == nil {
		r.Meta = map[string]string{}
	}
	r.Meta[k] = v
	return r
}

func (r Result) WithData(data any) Result {
	r.Data = data
	return r
}

func (r Result) WithHttpStatus(status int) Result {
	r.httpStatus = status
	return r
}
func (r Result) GetHttpStatus() int {
	return r.httpStatus
}

func (r Result) WithGrpcStatus(grpcStatus int) Result {
	r.grpcStatus = grpcStatus
	return r
}
func (r Result) GetGrpcStatus() int {
	return r.grpcStatus
}
