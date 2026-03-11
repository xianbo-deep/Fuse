package mux

import "strings"

// IsHTTP1 HTTP1的协议匹配器 [Matcher]。
func IsHTTP1(fc *FuseConn) bool {
	b, err := fc.Peek(8)
	if err != nil {
		return false
	}
	var method = string(b)
	return strings.HasPrefix(method, "GET ") ||
		strings.HasPrefix(method, "HEAD") ||
		strings.HasPrefix(method, "POST") ||
		strings.HasPrefix(method, "PUT ") ||
		strings.HasPrefix(method, "DELETE") ||
		strings.HasPrefix(method, "CONNECT") ||
		strings.HasPrefix(method, "OPTIONS") ||
		strings.HasPrefix(method, "TRACE")
}

// IsHTTP2 HTTP2的协议匹配器 [Matcher]。
func IsHTTP2(fc *FuseConn) bool {
	b, err := fc.Peek(24)
	if err != nil {
		return false
	}
	return string(b) == "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
}
