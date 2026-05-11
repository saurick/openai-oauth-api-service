package v1

import "fmt"

func (x *GetJsonrpcRequest) Redact() string {
	if x == nil {
		return "<nil>"
	}
	return fmt.Sprintf("url:%q jsonrpc:%q method:%q id:%q params:%q", x.Url, x.Jsonrpc, x.Method, x.Id, "[REDACTED]")
}

func (x *PostJsonrpcRequest) Redact() string {
	if x == nil {
		return "<nil>"
	}
	return fmt.Sprintf("url:%q jsonrpc:%q method:%q id:%q params:%q", x.Url, x.Jsonrpc, x.Method, x.Id, "[REDACTED]")
}
