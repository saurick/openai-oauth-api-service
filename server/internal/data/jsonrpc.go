// server/internal/data/jsonrpc.go
package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "server/api/jsonrpc/v1"
	"server/internal/biz"
	"server/internal/conf"
	"server/internal/errcode"

	"github.com/go-kratos/kratos/v2/log"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/types/known/structpb"
)

type adminAccountReader interface {
	GetAdminByID(ctx context.Context, id int) (*biz.AdminUser, error)
}

// JsonrpcData 是 JSON-RPC 的唯一业务入口，当前项目默认只保留通用鉴权和账号目录能力。
type JsonrpcData struct {
	data *Data
	log  *log.Helper
	cfg  *conf.Data

	authUC      *biz.AuthUsecase
	adminAuthUC *biz.AdminAuthUsecase
	userAdminUC *biz.UserAdminUsecase
	gatewayUC   *biz.GatewayUsecase

	adminReader adminAccountReader
}

// NewJsonrpcData：由 wire 注入底层 repo，再在入口层组装 usecase，保持模板的“入口聚合”风格。
func NewJsonrpcData(
	data *Data,
	c *conf.Data,
	logger log.Logger,
	authRepo *authRepo,
	adminAuthRepo *adminAuthRepo,
	tokenGenerator biz.TokenGenerator,
	adminTokenGenerator biz.AdminTokenGenerator,
	userAdminRepo biz.UserAdminRepo,
	gatewayUC *biz.GatewayUsecase,
	tracerProvider *tracesdk.TracerProvider,
) *JsonrpcData {
	helper := log.NewHelper(log.With(logger, "module", "data.jsonrpc"))

	if authRepo == nil {
		panic("NewJsonrpcData: authRepo is nil")
	}
	if adminAuthRepo == nil {
		panic("NewJsonrpcData: adminAuthRepo is nil")
	}
	if userAdminRepo == nil {
		panic("NewJsonrpcData: userAdminRepo is nil")
	}
	if tokenGenerator == nil {
		panic("NewJsonrpcData: tokenGenerator is nil")
	}
	if adminTokenGenerator == nil {
		panic("NewJsonrpcData: adminTokenGenerator is nil")
	}
	if gatewayUC == nil {
		panic("NewJsonrpcData: gatewayUC is nil")
	}
	if tracerProvider == nil {
		panic("NewJsonrpcData: tracerProvider is nil")
	}

	authUC := biz.NewAuthUsecase(authRepo, tokenGenerator, logger, tracerProvider)
	adminAuthUC := biz.NewAdminAuthUsecase(adminAuthRepo, adminTokenGenerator, logger, tracerProvider)
	userAdminUC := biz.NewUserAdminUsecase(userAdminRepo, logger, tracerProvider)

	helper.Info("JsonrpcData created (auth/admin auth/user admin usecases constructed inside)")

	return &JsonrpcData{
		data:        data,
		log:         helper,
		cfg:         c,
		authUC:      authUC,
		adminAuthUC: adminAuthUC,
		userAdminUC: userAdminUC,
		gatewayUC:   gatewayUC,
		adminReader: adminAuthRepo,
	}
}

var _ biz.JsonrpcRepo = (*JsonrpcData)(nil)

func (d *JsonrpcData) Handle(
	ctx context.Context,
	url, jsonrpc, method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	d.log.WithContext(ctx).Infof(
		"[jsonrpc] handle url=%s jsonrpc=%s method=%s id=%s",
		url, jsonrpc, method, id,
	)

	if params == nil {
		d.log.WithContext(ctx).Info("[jsonrpc] params=<nil>")
	} else {
		b, _ := json.MarshalIndent(redactJSONRPCParams(params.AsMap()), "", "  ")
		d.log.WithContext(ctx).Infof("[jsonrpc] params=%s", string(b))
	}

	if !d.isPublic(url, method) {
		if _, res := d.requireLogin(ctx); res != nil {
			return id, res, nil
		}
	}

	switch url {
	case "system":
		return d.handleSystem(ctx, id, method, params)
	case "auth":
		return d.handleAuth(ctx, method, id, params)
	case "user":
		return d.handleUser(ctx, method, id, params)
	case "api":
		return d.handleGateway(ctx, method, id, params)
	default:
		return id, &v1.JsonrpcResult{
			Code:    errcode.JSONRPCUnknownURL.Code,
			Message: fmt.Sprintf("unknown jsonrpc url=%s", url),
		}, nil
	}
}

func redactJSONRPCParams(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			if isSensitiveJSONRPCParamKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactJSONRPCParams(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = redactJSONRPCParams(item)
		}
		return out
	default:
		return value
	}
}

func isSensitiveJSONRPCParamKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "password", "token", "access_token", "refresh_token", "authorization",
		"api_key", "plain_key", "client_secret", "jwt_secret":
		return true
	default:
		return false
	}
}

func (r *JsonrpcData) handleSystem(
	ctx context.Context,
	id, method string,
	_ *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	logger := r.log.WithContext(ctx)
	logger.Info("Jsonrpc.system: start", "method", method, "id", id)

	switch method {
	case "ping":
		data := newDataStruct(map[string]any{"pong": "pong"})
		logger.Info("Jsonrpc.system.ping: success", "id", id)
		return id, &v1.JsonrpcResult{Code: errcode.OK.Code, Message: errcode.OK.Message, Data: data}, nil
	case "version":
		data := newDataStruct(map[string]any{"version": "1.0.0"})
		logger.Info("Jsonrpc.system.version: success", "id", id)
		return id, &v1.JsonrpcResult{Code: errcode.OK.Code, Message: errcode.OK.Message, Data: data}, nil
	default:
		logger.Warn("Jsonrpc.system: unknown method", "method", method, "id", id)
		return id, &v1.JsonrpcResult{
			Code:    errcode.UnknownMethod.Code,
			Message: fmt.Sprintf("unknown system method: %s", method),
		}, nil
	}
}

func (d *JsonrpcData) handleAuth(
	ctx context.Context,
	method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	d.log.WithContext(ctx).Infof("[auth] method=%s id=%s", method, id)

	pm := map[string]any{}
	if params != nil {
		pm = params.AsMap()
	}

	switch method {
	case "login":
		username := getString(pm, "username")
		password := getString(pm, "password")

		if username == "" || password == "" {
			return id, &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "缺少用户名或密码"}, nil
		}

		token, expireAt, user, err := d.authUC.Login(ctx, username, password)
		if err != nil {
			return id, d.mapAuthError(ctx, err), nil
		}

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: "登录成功",
			Data: newDataStruct(map[string]any{
				"user_id":      user.ID,
				"username":     user.Username,
				"access_token": token,
				"expires_at":   expireAt.Unix(),
				"token_type":   "Bearer",
				"issued_at":    time.Now().Unix(),
			}),
		}, nil

	case "admin_login":
		username := getString(pm, "username")
		password := getString(pm, "password")

		if username == "" || password == "" {
			return id, &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "缺少用户名或密码"}, nil
		}

		token, expireAt, admin, err := d.adminAuthUC.Login(ctx, username, password)
		if err != nil {
			return id, d.mapAuthError(ctx, err), nil
		}

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: "登录成功",
			Data: newDataStruct(map[string]any{
				"user_id":      admin.ID,
				"username":     admin.Username,
				"access_token": token,
				"expires_at":   expireAt.Unix(),
				"token_type":   "Bearer",
				"issued_at":    time.Now().Unix(),
			}),
		}, nil

	case "register":
		return id, &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "公开注册已关闭，请联系管理员创建组织账号"}, nil

	case "logout":
		claims, _ := biz.GetClaimsFromContext(ctx)
		if claims != nil {
			d.log.WithContext(ctx).Infof(
				"[auth] user logout uid=%d uname=%s role=%d id=%s",
				claims.UserID,
				claims.Username,
				claims.Role,
				id,
			)
		} else {
			d.log.WithContext(ctx).Warnf("[auth] user logout without claims id=%s", id)
		}

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: errcode.OK.Message,
		}, nil

	case "me":
		claims, ok := biz.GetClaimsFromContext(ctx)
		if !ok || claims == nil {
			return id, &v1.JsonrpcResult{Code: errcode.AuthRequired.Code, Message: errcode.AuthRequired.Message}, nil
		}

		if claims.Role == biz.RoleAdmin {
			admin, err := d.getCurrentAdmin(ctx, claims)
			if err != nil {
				d.log.WithContext(ctx).Warnf("auth.me GetCurrentAdmin failed uid=%d err=%v", claims.UserID, err)
				return id, &v1.JsonrpcResult{Code: errcode.AuthCurrentUserFailed.Code, Message: errcode.AuthCurrentUserFailed.Message}, nil
			}

			return id, &v1.JsonrpcResult{
				Code:    errcode.OK.Code,
				Message: errcode.OK.Message,
				Data: newDataStruct(map[string]any{
					"id":       admin.ID,
					"username": admin.Username,
					"role":     int(biz.RoleAdmin),
					"disabled": admin.Disabled,
				}),
			}, nil
		}

		u, err := d.authUC.GetCurrentUser(ctx, claims.UserID)
		if err != nil {
			d.log.WithContext(ctx).Warnf("auth.me GetCurrentUser failed uid=%d err=%v", claims.UserID, err)
			return id, &v1.JsonrpcResult{Code: errcode.AuthCurrentUserFailed.Code, Message: errcode.AuthCurrentUserFailed.Message}, nil
		}

		data := map[string]any{
			"id":         u.ID,
			"username":   u.Username,
			"role":       u.Role,
			"disabled":   u.Disabled,
			"created_at": u.CreatedAt.Unix(),
		}
		if u.LastLoginAt != nil {
			data["last_login_at"] = u.LastLoginAt.Unix()
		}

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: errcode.OK.Message,
			Data:    newDataStruct(data),
		}, nil

	default:
		return id, &v1.JsonrpcResult{
			Code:    errcode.UnknownMethod.Code,
			Message: fmt.Sprintf("auth: 未知方法=%s", method),
		}, nil
	}
}

func (d *JsonrpcData) mapAuthError(ctx context.Context, err error) *v1.JsonrpcResult {
	logger := d.log.WithContext(ctx)

	switch err {
	case biz.ErrUserNotFound:
		logger.Warn("[auth] user not found")
		return &v1.JsonrpcResult{
			Code:    errcode.AuthUserNotFound.Code,
			Message: errcode.AuthUserNotFound.Message,
		}
	case biz.ErrInvalidPassword:
		logger.Warn("[auth] invalid password")
		return &v1.JsonrpcResult{
			Code:    errcode.AuthInvalidPassword.Code,
			Message: errcode.AuthInvalidPassword.Message,
		}
	case biz.ErrUserDisabled:
		logger.Warn("[auth] user disabled")
		return &v1.JsonrpcResult{
			Code:    errcode.AuthUserDisabled.Code,
			Message: errcode.AuthUserDisabled.Message,
		}
	case biz.ErrUserExists:
		logger.Warn("[auth] user already exists")
		return &v1.JsonrpcResult{
			Code:    errcode.AuthUserExists.Code,
			Message: errcode.AuthUserExists.Message,
		}
	default:
		logger.Errorf("[auth] internal error: %v", err)
		return &v1.JsonrpcResult{
			Code:    errcode.Internal.Code,
			Message: errcode.Internal.Message,
		}
	}
}

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func getInt(m map[string]any, key string, def int) int {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return def
	}
}

func getInt64(m map[string]any, key string, def int64) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}

	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n
		}
		return def
	default:
		return def
	}
}

func getInt64WithFallback(m map[string]any, key, fallbackKey string) int64 {
	if v, ok := m[key]; ok && v != nil {
		return getInt64(m, key, 0)
	}
	return getInt64(m, fallbackKey, 0)
}

func getFloat(m map[string]any, key string, def float64) float64 {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		if n, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil {
			return n
		}
		return def
	default:
		return def
	}
}

func getBool(m map[string]any, key string, def bool) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func newDataStruct(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func (d *JsonrpcData) requireLogin(ctx context.Context) (*biz.AuthClaims, *v1.JsonrpcResult) {
	if c, ok := biz.GetClaimsFromContext(ctx); ok && c != nil {
		return c, nil
	}

	switch biz.AuthStateFrom(ctx) {
	case biz.AuthExpired:
		return nil, &v1.JsonrpcResult{Code: errcode.AuthExpired.Code, Message: errcode.AuthExpired.Message}
	case biz.AuthInvalid:
		return nil, &v1.JsonrpcResult{Code: errcode.AuthInvalid.Code, Message: errcode.AuthInvalid.Message}
	default:
		return nil, &v1.JsonrpcResult{Code: errcode.AuthRequired.Code, Message: errcode.AuthRequired.Message}
	}
}

func (d *JsonrpcData) requireAdmin(ctx context.Context) (*biz.AuthClaims, *v1.JsonrpcResult) {
	c, res := d.requireLogin(ctx)
	if res != nil {
		return nil, res
	}
	if c.Role != biz.RoleAdmin {
		return nil, &v1.JsonrpcResult{Code: errcode.AdminRequired.Code, Message: errcode.AdminRequired.Message}
	}

	admin, err := d.getCurrentAdmin(ctx, c)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, &v1.JsonrpcResult{Code: errcode.AdminRequired.Code, Message: errcode.AdminRequired.Message}
		case errors.Is(err, errAdminUsernameMismatch):
			return nil, &v1.JsonrpcResult{Code: errcode.AdminRequired.Code, Message: errcode.AdminRequired.Message}
		default:
			d.log.WithContext(ctx).Errorf("[auth] requireAdmin verify current admin failed err=%v", err)
			return nil, &v1.JsonrpcResult{Code: errcode.Internal.Code, Message: errcode.Internal.Message}
		}
	}
	if admin.Disabled {
		return nil, &v1.JsonrpcResult{Code: errcode.AdminDisabled.Code, Message: errcode.AdminDisabled.Message}
	}

	return c, nil
}

var errAdminUsernameMismatch = errors.New("admin username mismatch")

func (d *JsonrpcData) getCurrentAdmin(ctx context.Context, claims *biz.AuthClaims) (*biz.AdminUser, error) {
	if claims == nil {
		return nil, errors.New("missing auth claims")
	}
	if d.adminReader == nil {
		return nil, errors.New("admin reader is nil")
	}

	admin, err := d.adminReader.GetAdminByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, sql.ErrNoRows
	}
	// 安全兜底：管理员 token 的 uid/uname 必须同时匹配，避免旧 token 在账号重建后误复用。
	if admin.Username != claims.Username {
		return nil, errAdminUsernameMismatch
	}
	return admin, nil
}

func (d *JsonrpcData) isPublic(url, method string) bool {
	if url == "system" && (method == "ping" || method == "version") {
		return true
	}
	if url == "auth" && (method == "login" || method == "admin_login" || method == "logout") {
		return true
	}
	return false
}

func (d *JsonrpcData) handleUser(
	ctx context.Context,
	method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	l := d.log.WithContext(ctx)

	pm := map[string]any{}
	if params != nil {
		pm = params.AsMap()
	}

	var opUID int
	var opUname string
	var opRole biz.Role
	if c, ok := biz.GetClaimsFromContext(ctx); ok && c != nil {
		opUID, opUname, opRole = c.UserID, c.Username, c.Role
	}

	l.Infof("[user] handle start method=%s id=%s operator_uid=%d operator_uname=%s operator_role=%d",
		method, id, opUID, opUname, opRole,
	)

	if _, res := d.requireAdmin(ctx); res != nil {
		l.Warnf("[user] requireAdmin denied method=%s id=%s operator_uid=%d code=%d msg=%s",
			method, id, opUID, res.Code, res.Message,
		)
		return id, res, nil
	}

	switch method {
	case "create":
		username := getString(pm, "username")
		password := getString(pm, "password")
		u, err := d.userAdminUC.Create(ctx, username, password)
		if err != nil {
			return id, d.mapUserAdminError(ctx, err), nil
		}
		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: "创建组织账号成功",
			Data: newDataStruct(map[string]any{
				"id":         u.ID,
				"username":   u.Username,
				"disabled":   u.Disabled,
				"created_at": u.CreatedAt.Unix(),
			}),
		}, nil

	case "reset_password":
		userID := getInt(pm, "user_id", 0)
		password := getString(pm, "password")
		if err := d.userAdminUC.ResetPassword(ctx, userID, password); err != nil {
			return id, d.mapUserAdminError(ctx, err), nil
		}
		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: "重置组织账号密码成功",
			Data: newDataStruct(map[string]any{
				"user_id": userID,
			}),
		}, nil

	case "list":
		limit := getInt(pm, "limit", 30)
		offset := getInt(pm, "offset", 0)
		search := strings.TrimSpace(getString(pm, "search"))

		l.Infof("[user] list start id=%s operator_uid=%d limit=%d offset=%d search=%q",
			id, opUID, limit, offset, search,
		)

		list, total, err := d.userAdminUC.List(ctx, limit, offset, search)
		if err != nil {
			l.Errorf("[user] list failed id=%s operator_uid=%d limit=%d offset=%d search=%q err=%v",
				id, opUID, limit, offset, search, err,
			)
			return id, &v1.JsonrpcResult{Code: errcode.UserListFailed.Code, Message: errcode.UserListFailed.Message}, nil
		}

		arr := make([]any, 0, len(list))
		for _, u := range list {
			lastLogin := int64(0)
			if u.LastLoginAt != nil {
				lastLogin = u.LastLoginAt.Unix()
			}
			arr = append(arr, map[string]any{
				"id":            u.ID,
				"username":      u.Username,
				"disabled":      u.Disabled,
				"last_login_at": lastLogin,
				"created_at":    u.CreatedAt.Unix(),
			})
		}

		l.Infof("[user] list success id=%s operator_uid=%d count=%d total=%d search=%q",
			id, opUID, len(list), total, search,
		)

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: "获取账号列表成功",
			Data: newDataStruct(map[string]any{
				"users":  arr,
				"total":  total,
				"limit":  limit,
				"offset": offset,
				"search": search,
			}),
		}, nil

	case "set_disabled":
		userID := getInt(pm, "user_id", 0)
		if userID <= 0 {
			l.Warnf("[user] set_disabled bad param id=%s operator_uid=%d user_id=%d", id, opUID, userID)
			return id, &v1.JsonrpcResult{
				Code:    errcode.UserSetDisabledInvalid.Code,
				Message: errcode.UserSetDisabledInvalid.Message,
			}, nil
		}

		disabled := getBool(pm, "disabled", false)

		l.Infof("[user] set_disabled start id=%s operator_uid=%d target_uid=%d disabled=%v",
			id, opUID, userID, disabled,
		)

		if err := d.userAdminUC.SetDisabled(ctx, userID, disabled); err != nil {
			l.Errorf("[user] set_disabled failed id=%s operator_uid=%d target_uid=%d disabled=%v err=%v",
				id, opUID, userID, disabled, err,
			)
			return id, d.mapUserAdminError(ctx, err), nil
		}

		msg := "启用成功"
		if disabled {
			msg = "禁用成功"
		}

		l.Infof("[user] set_disabled success id=%s operator_uid=%d target_uid=%d disabled=%v",
			id, opUID, userID, disabled,
		)

		return id, &v1.JsonrpcResult{
			Code:    errcode.OK.Code,
			Message: msg,
			Data: newDataStruct(map[string]any{
				"success":  true,
				"user_id":  userID,
				"disabled": disabled,
			}),
		}, nil

	default:
		l.Warnf("[user] unknown method=%s id=%s operator_uid=%d", method, id, opUID)
		return id, &v1.JsonrpcResult{
			Code:    errcode.UnknownMethod.Code,
			Message: fmt.Sprintf("未知用户接口 method=%s", method),
		}, nil
	}
}

func (d *JsonrpcData) mapUserAdminError(ctx context.Context, err error) *v1.JsonrpcResult {
	l := d.log.WithContext(ctx)

	switch err {
	case biz.ErrUserNotFound:
		return &v1.JsonrpcResult{Code: errcode.AuthUserNotFound.Code, Message: errcode.AuthUserNotFound.Message}
	case biz.ErrBadParam:
		return &v1.JsonrpcResult{Code: errcode.UserInvalidParam.Code, Message: errcode.UserInvalidParam.Message}
	case biz.ErrForbidden:
		return &v1.JsonrpcResult{Code: errcode.AdminRequired.Code, Message: errcode.AdminRequired.Message}
	case biz.ErrNoPermission:
		return &v1.JsonrpcResult{Code: errcode.PermissionDenied.Code, Message: errcode.PermissionDenied.Message}
	default:
		l.Errorf("[user] internal err=%v", err)
		return &v1.JsonrpcResult{Code: errcode.Internal.Code, Message: errcode.Internal.Message}
	}
}
