package middleware

import (
	"fmt"

	"github.com/xianbo-deep/Fuse/core"
)

// Recovery 捕获项目出现的 panic ，防止系统崩溃。
func Recovery() core.HandlerFunc {
	// 使用命名返回值 防止捕获panic后无返回值
	return func(c core.Ctx) (res core.Result) {
		// 捕获panic
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic: %v", r)
				c.Err(err)

				c.Abort()
				res = core.Fail(core.CodeInternal, "catch panic from service")
			}
		}()

		res = c.Next()
		return
	}
}
