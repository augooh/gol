package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

func trace(message string) string {
	var pcs [32]uintptr
	// 这里使用 runtime.Callers 函数获取当前调用栈的信息。r
	// untime.Callers 接受两个参数，第一个参数是调用栈的起始深度，
	// 这里传递了 3 表示跳过前三个调用栈帧（函数调用）：第 0 个 Caller 是 Callers 本身，第 1 个是上一层 trace，第 2 个是再上一层的 defer func
	// pcs 是一个数组，用于存储返回的调用栈信息，n 是实际返回的调用栈帧数目。
	n := runtime.Callers(3, pcs[:])

	var str strings.Builder
	// 追加消息和堆栈信息
	str.WriteString(message + "\nTraceback:")

	// 遍历堆栈信息并获取函数、文件和行号
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				c.Fail(http.StatusInternalServerError, "INternal Server Error")
			}
		}()

		c.Next()
	}
}
