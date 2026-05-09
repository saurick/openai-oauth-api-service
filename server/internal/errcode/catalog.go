package errcode

// Definition 统一描述一个对外错误码，保持“一码一义”，避免前后端语义漂移。
type Definition struct {
	Name    string
	Code    int32
	Message string
}

var (
	OK = Definition{Name: "OK", Code: 0, Message: "OK"}

	JSONRPCUnknownURL = Definition{Name: "JSONRPCUnknownURL", Code: 40001, Message: "未知 RPC 域"}
	InvalidParam      = Definition{Name: "InvalidParam", Code: 40010, Message: "参数不合法"}
	UnknownMethod     = Definition{Name: "UnknownMethod", Code: 40020, Message: "未知接口"}
	UserInvalidParam  = Definition{Name: "UserInvalidParam", Code: 40030, Message: "参数不合法"}

	UserSetDisabledInvalid = Definition{Name: "UserSetDisabledInvalid", Code: 40071, Message: "参数错误：user_id 无效"}
	APIKeyInvalid          = Definition{Name: "APIKeyInvalid", Code: 40101, Message: "API key 无效"}
	APIKeyDisabled         = Definition{Name: "APIKeyDisabled", Code: 40102, Message: "API key 已禁用"}
	APIModelDisabled       = Definition{Name: "APIModelDisabled", Code: 40320, Message: "模型已禁用"}
	APIModelNotAllowed     = Definition{Name: "APIModelNotAllowed", Code: 40321, Message: "当前 key 不允许使用该模型"}

	AdminRequired    = Definition{Name: "AdminRequired", Code: 40301, Message: "需要管理员权限"}
	AuthRequired     = Definition{Name: "AuthRequired", Code: 40302, Message: "未登录"}
	AdminDisabled    = Definition{Name: "AdminDisabled", Code: 40303, Message: "管理员已禁用"}
	PermissionDenied = Definition{Name: "PermissionDenied", Code: 40304, Message: "权限不足"}

	AuthUserNotFound    = Definition{Name: "AuthUserNotFound", Code: 10001, Message: "用户不存在"}
	AuthInvalidPassword = Definition{Name: "AuthInvalidPassword", Code: 10002, Message: "密码错误"}
	AuthUserDisabled    = Definition{Name: "AuthUserDisabled", Code: 10003, Message: "用户已被禁用"}
	AuthUserExists      = Definition{Name: "AuthUserExists", Code: 10004, Message: "用户名已存在"}
	AuthExpired         = Definition{Name: "AuthExpired", Code: 10005, Message: "登录已过期，请重新登录"}
	AuthInvalid         = Definition{Name: "AuthInvalid", Code: 10006, Message: "登录无效，请重新登录"}

	Internal              = Definition{Name: "Internal", Code: 50000, Message: "服务器内部错误"}
	AuthCurrentUserFailed = Definition{Name: "AuthCurrentUserFailed", Code: 50001, Message: "获取用户信息失败"}
	UserListFailed        = Definition{Name: "UserListFailed", Code: 50020, Message: "获取用户列表失败"}
	APIOperationFailed    = Definition{Name: "APIOperationFailed", Code: 50040, Message: "API 操作失败"}
)

var definitions = []Definition{
	OK,
	JSONRPCUnknownURL,
	InvalidParam,
	UnknownMethod,
	UserInvalidParam,
	UserSetDisabledInvalid,
	APIKeyInvalid,
	APIKeyDisabled,
	APIModelDisabled,
	APIModelNotAllowed,
	AdminRequired,
	AuthRequired,
	AdminDisabled,
	PermissionDenied,
	AuthUserNotFound,
	AuthInvalidPassword,
	AuthUserDisabled,
	AuthUserExists,
	AuthExpired,
	AuthInvalid,
	Internal,
	AuthCurrentUserFailed,
	UserListFailed,
	APIOperationFailed,
}

func Definitions() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
	return out
}

// IsAuthFailureCode 仅识别“需要重新登录”的登录态错误，避免把权限不足误处理成登出。
func IsAuthFailureCode(code int32) bool {
	switch code {
	case AuthExpired.Code, AuthInvalid.Code, AuthRequired.Code:
		return true
	default:
		return false
	}
}
