package core

import (
	"context"
)

type Ctx interface {
	// 返回上下文
	Context() context.Context
	// 更改上下文
	WithContext(context.Context)
	// 是否终止
	Aborted() bool

	// 推进
	Next() Result
	// 终止
	Abort()

	Set(key string, val any)
	Get(key string) (any, bool)

	// 根据协议渲染响应
	Render(result Result)

	// 拷贝上下文
	Copy() Ctx

	// 响应
	Success(data any) Result
	Fail(code int, msg string) Result

	// 获取路径参数
	Param(key string) string
	// 获取查询参数
	Query(key string) string

	// 记录错误
	Err(err error)
	// 最后一个错误
	Error() error
	// 全部错误
	Errors() []error
}
