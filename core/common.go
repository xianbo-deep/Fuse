package core

const (
	CtxKeyRequestID = "request_id"
	CtxKeyMethod    = "method"
	CtxKeyPath      = "path"
	CtxKeyProtocol  = "protocol"
)

const (
	ProtocolHTTP = "HTTP"
	ProtocolGRPC = "GRPC"
	ProtocolCRON = "CRON"
	ProtocolWS   = "WEBSOCKET"
)

const (
	MethodGet    = "GET"
	MethodPost   = "POST"
	MethodPut    = "PUT"
	MethodDelete = "DELETE"

	MethodUnary  = "unary"
	MethodStream = "stream"
)

const (
	CodeSuccess      = 0
	CodeBadRequest   = 1001
	CodeUnauthorized = 2001
	CodeForbidden    = 3001
	CodeNotFound     = 4004
	CodeInternal     = 9001
)
