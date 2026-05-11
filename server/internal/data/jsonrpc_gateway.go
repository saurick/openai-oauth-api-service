package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "server/api/jsonrpc/v1"
	"server/internal/biz"
	"server/internal/errcode"

	"google.golang.org/protobuf/types/known/structpb"
)

func (d *JsonrpcData) handleGateway(
	ctx context.Context,
	method, id string,
	params *structpb.Struct,
) (string, *v1.JsonrpcResult, error) {
	l := d.log.WithContext(ctx)
	pm := map[string]any{}
	if params != nil {
		pm = params.AsMap()
	}

	if strings.HasPrefix(method, "user_") {
		return d.handleGatewayUser(ctx, method, id, pm)
	}

	if _, res := d.requireAdmin(ctx); res != nil {
		l.Warnf("[gateway] requireAdmin denied method=%s id=%s code=%d msg=%s", method, id, res.Code, res.Message)
		return id, res, nil
	}

	switch method {
	case "summary":
		filter := gatewayUsageFilterFromParams(pm)
		summary, err := d.gatewayUC.SummarizeUsage(ctx, filter)
		if err != nil {
			l.Errorf("[gateway] summary failed err=%v", err)
			return id, &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}, nil
		}
		return id, okResult("获取 API 用量概览成功", map[string]any{
			"summary": mapGatewaySummary(summary),
		}), nil

	case "gateway_upstream_get":
		mode, err := d.gatewayUC.GetCodexUpstreamMode(ctx)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		return id, okResult("获取 Codex 上游模式成功", mapGatewayUpstreamModeForRPC(mode)), nil

	case "gateway_upstream_set":
		mode, err := d.gatewayUC.SetCodexUpstreamMode(ctx, getString(pm, "mode"))
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.gateway_upstream_set", "gateway_setting", biz.GatewaySettingCodexUpstreamMode, map[string]any{
			"mode": mode,
		})
		return id, okResult("切换 Codex 上游模式成功", mapGatewayUpstreamModeForRPC(mode)), nil

	case "key_list":
		limit := getInt(pm, "limit", 30)
		offset := getInt(pm, "offset", 0)
		search := strings.TrimSpace(getString(pm, "search"))
		list, total, err := d.gatewayUC.ListAPIKeys(ctx, limit, offset, search)
		if err != nil {
			l.Errorf("[gateway] key_list failed err=%v", err)
			return id, &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}, nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayAPIKeyForRPC(item, true))
		}
		return id, okResult("获取 API key 列表成功", map[string]any{
			"items":  items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
			"search": search,
		}), nil

	case "key_create":
		item, err := d.gatewayUC.CreateAPIKey(ctx, biz.CreateGatewayAPIKeyInput{
			Name:                           getString(pm, "name"),
			OwnerUserID:                    getInt(pm, "owner_user_id", 0),
			QuotaRequests:                  getInt64(pm, "quota_requests", 0),
			QuotaDailyTokens:               getInt64(pm, "quota_daily_tokens", 0),
			QuotaWeeklyTokens:              getInt64WithFallback(pm, "quota_weekly_tokens", "quota_total_tokens"),
			QuotaDailyInputTokens:          getInt64(pm, "quota_daily_input_tokens", 0),
			QuotaWeeklyInputTokens:         getInt64(pm, "quota_weekly_input_tokens", 0),
			QuotaDailyOutputTokens:         getInt64(pm, "quota_daily_output_tokens", 0),
			QuotaWeeklyOutputTokens:        getInt64(pm, "quota_weekly_output_tokens", 0),
			QuotaDailyBillableInputTokens:  getInt64(pm, "quota_daily_billable_input_tokens", 0),
			QuotaWeeklyBillableInputTokens: getInt64(pm, "quota_weekly_billable_input_tokens", 0),
			AllowedModels:                  getStringList(pm, "allowed_models"),
		})
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.key_create", "api_key", fmt.Sprint(item.ID), map[string]any{
			"name":          item.Name,
			"owner_user_id": item.OwnerUserID,
		})
		data := mapGatewayAPIKeyForRPC(&item.GatewayAPIKey, true)
		data["plain_key"] = item.PlainKey
		return id, okResult("创建 API key 成功，明文 key 只返回一次", data), nil

	case "key_update":
		keyID := getInt(pm, "key_id", 0)
		item, err := d.gatewayUC.UpdateAPIKey(ctx, biz.UpdateGatewayAPIKeyInput{
			ID:                             keyID,
			Name:                           getString(pm, "name"),
			OwnerUserID:                    getInt(pm, "owner_user_id", 0),
			QuotaRequests:                  getInt64(pm, "quota_requests", 0),
			QuotaDailyTokens:               getInt64(pm, "quota_daily_tokens", 0),
			QuotaWeeklyTokens:              getInt64WithFallback(pm, "quota_weekly_tokens", "quota_total_tokens"),
			QuotaDailyInputTokens:          getInt64(pm, "quota_daily_input_tokens", 0),
			QuotaWeeklyInputTokens:         getInt64(pm, "quota_weekly_input_tokens", 0),
			QuotaDailyOutputTokens:         getInt64(pm, "quota_daily_output_tokens", 0),
			QuotaWeeklyOutputTokens:        getInt64(pm, "quota_weekly_output_tokens", 0),
			QuotaDailyBillableInputTokens:  getInt64(pm, "quota_daily_billable_input_tokens", 0),
			QuotaWeeklyBillableInputTokens: getInt64(pm, "quota_weekly_billable_input_tokens", 0),
			AllowedModels:                  getStringList(pm, "allowed_models"),
			Disabled:                       getBool(pm, "disabled", false),
		})
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.key_update", "api_key", fmt.Sprint(item.ID), map[string]any{
			"owner_user_id": item.OwnerUserID,
			"disabled":      item.Disabled,
		})
		return id, okResult("更新 API key 成功", mapGatewayAPIKeyForRPC(item, true)), nil

	case "key_delete":
		keyID := getInt(pm, "key_id", 0)
		if err := d.gatewayUC.DeleteAPIKey(ctx, keyID); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.key_delete", "api_key", fmt.Sprint(keyID), nil)
		return id, okResult("删除 API key 成功", map[string]any{"key_id": keyID}), nil

	case "key_delete_batch":
		keyIDs := getIntList(pm, "key_ids")
		deleted, err := d.gatewayUC.DeleteAPIKeys(ctx, keyIDs)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.key_delete_batch", "api_key", "batch", map[string]any{
			"key_ids": keyIDs,
			"deleted": deleted,
		})
		return id, okResult("批量删除 API key 成功", map[string]any{"key_ids": keyIDs, "deleted": deleted}), nil

	case "key_set_disabled":
		keyID := getInt(pm, "key_id", 0)
		disabled := getBool(pm, "disabled", false)
		if err := d.gatewayUC.SetAPIKeyDisabled(ctx, keyID, disabled); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		msg := "启用 API key 成功"
		d.auditGateway(ctx, "api.key_set_disabled", "api_key", fmt.Sprint(keyID), map[string]any{
			"disabled": disabled,
		})
		if disabled {
			msg = "禁用 API key 成功"
		}
		return id, okResult(msg, map[string]any{
			"key_id":   keyID,
			"disabled": disabled,
		}), nil

	case "usage_list":
		filter := gatewayUsageFilterFromParams(pm)
		list, total, err := d.gatewayUC.ListUsageLogs(ctx, filter)
		if err != nil {
			l.Errorf("[api] usage_list failed err=%v", err)
			return id, &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}, nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayUsageForRPC(item))
		}
		summary, err := d.gatewayUC.SummarizeUsage(ctx, filter)
		if err != nil {
			l.Errorf("[api] usage summary in list failed err=%v", err)
			return id, &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}, nil
		}
		return id, okResult("获取 usage 记录成功", map[string]any{
			"items":   items,
			"total":   total,
			"limit":   filter.Limit,
			"offset":  filter.Offset,
			"summary": mapGatewaySummary(summary),
		}), nil

	case "usage_buckets":
		filter := gatewayUsageFilterFromParams(pm)
		groupBy := strings.TrimSpace(getString(pm, "group_by"))
		buckets, err := d.gatewayUC.ListUsageBuckets(ctx, filter, groupBy)
		if err != nil {
			l.Errorf("[api] usage_buckets failed err=%v", err)
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(buckets))
		for _, item := range buckets {
			items = append(items, mapGatewayUsageBucketForRPC(item))
		}
		return id, okResult("获取 usage 聚合成功", map[string]any{
			"items":    items,
			"group_by": groupBy,
		}), nil

	case "usage_key_summaries":
		filter := gatewayUsageFilterFromParams(pm)
		limit := getInt(pm, "limit", 30)
		items, err := d.gatewayUC.ListUsageKeySummaries(ctx, filter, limit)
		if err != nil {
			l.Errorf("[api] usage_key_summaries failed err=%v", err)
			return id, d.mapGatewayError(ctx, err), nil
		}
		out := make([]any, 0, len(items))
		for _, item := range items {
			out = append(out, mapGatewayUsageKeySummaryForRPC(item))
		}
		return id, okResult("获取 key usage 聚合成功", map[string]any{
			"items": out,
			"limit": limit,
		}), nil

	case "usage_session_summaries":
		filter := gatewayUsageFilterFromParams(pm)
		limit := getInt(pm, "limit", 30)
		offset := getInt(pm, "offset", 0)
		items, total, err := d.gatewayUC.ListUsageSessionSummaries(ctx, filter, limit, offset)
		if err != nil {
			l.Errorf("[api] usage_session_summaries failed err=%v", err)
			return id, d.mapGatewayError(ctx, err), nil
		}
		out := make([]any, 0, len(items))
		for _, item := range items {
			out = append(out, mapGatewayUsageSessionSummaryForRPC(item))
		}
		return id, okResult("获取会话 usage 聚合成功", map[string]any{
			"items":  out,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		}), nil

	case "model_list":
		limit := getInt(pm, "limit", 100)
		offset := getInt(pm, "offset", 0)
		enabledOnly := getBool(pm, "enabled_only", false)
		search := strings.TrimSpace(getString(pm, "search"))
		list, total, err := d.gatewayUC.ListModels(ctx, limit, offset, enabledOnly, search)
		if err != nil {
			l.Errorf("[api] model_list failed err=%v", err)
			return id, &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}, nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayModelForRPC(item))
		}
		return id, okResult("获取模型列表成功", map[string]any{
			"items":        items,
			"total":        total,
			"limit":        limit,
			"offset":       offset,
			"search":       search,
			"enabled_only": enabledOnly,
		}), nil

	case "official_model_price_list":
		items := make([]any, 0, len(biz.OfficialModelPrices))
		for _, item := range biz.OfficialModelPrices {
			items = append(items, mapGatewayPriceForRPC(item))
		}
		return id, okResult("获取官方模型价格成功", map[string]any{"items": items}), nil

	case "model_upsert":
		return id, d.mapGatewayError(ctx, biz.ErrGatewayModelCatalogFixed), nil

	case "model_set_enabled":
		modelID := getInt(pm, "id", 0)
		enabled := getBool(pm, "enabled", true)
		if err := d.gatewayUC.SetModelEnabled(ctx, modelID, enabled); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.model_set_enabled", "api_model", fmt.Sprint(modelID), map[string]any{
			"enabled": enabled,
		})
		return id, okResult("更新模型状态成功", map[string]any{
			"id":      modelID,
			"enabled": enabled,
		}), nil

	case "model_delete":
		return id, d.mapGatewayError(ctx, biz.ErrGatewayModelCatalogFixed), nil

	case "policy_list":
		apiKeyID := getInt(pm, "api_key_id", 0)
		list, err := d.gatewayUC.ListPolicies(ctx, apiKeyID)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayPolicyForRPC(item))
		}
		return id, okResult("获取策略列表成功", map[string]any{"items": items}), nil

	case "policy_upsert":
		item, err := d.gatewayUC.UpsertPolicy(ctx, biz.GatewayPolicy{
			APIKeyID:        getInt(pm, "api_key_id", 0),
			ModelID:         getString(pm, "model_id"),
			RPM:             getInt64(pm, "rpm", 0),
			TPM:             getInt64(pm, "tpm", 0),
			DailyRequests:   getInt64(pm, "daily_requests", 0),
			MonthlyRequests: getInt64(pm, "monthly_requests", 0),
			DailyTokens:     getInt64(pm, "daily_tokens", 0),
			MonthlyTokens:   getInt64(pm, "monthly_tokens", 0),
		})
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.policy_upsert", "api_policy", fmt.Sprint(item.ID), map[string]any{
			"api_key_id": item.APIKeyID,
			"model_id":   item.ModelID,
		})
		return id, okResult("保存策略成功", mapGatewayPolicyForRPC(item)), nil

	case "policy_delete":
		policyID := getInt(pm, "policy_id", 0)
		if err := d.gatewayUC.DeletePolicy(ctx, policyID); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.policy_delete", "api_policy", fmt.Sprint(policyID), nil)
		return id, okResult("删除策略成功", map[string]any{"policy_id": policyID}), nil

	case "price_list":
		list, err := d.gatewayUC.ListModelPrices(ctx)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayPriceForRPC(item))
		}
		return id, okResult("获取价格列表成功", map[string]any{"items": items}), nil

	case "price_upsert":
		item, err := d.gatewayUC.UpsertModelPrice(ctx, biz.GatewayModelPrice{
			ModelID:                  getString(pm, "model_id"),
			InputUSDPerMillion:       getFloat(pm, "input_usd_per_million", 0),
			CachedInputUSDPerMillion: getFloat(pm, "cached_input_usd_per_million", 0),
			OutputUSDPerMillion:      getFloat(pm, "output_usd_per_million", 0),
		})
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.price_upsert", "api_model_price", item.ModelID, nil)
		return id, okResult("保存模型价格成功", mapGatewayPriceForRPC(item)), nil

	case "price_delete":
		priceID := getInt(pm, "price_id", 0)
		if err := d.gatewayUC.DeleteModelPrice(ctx, priceID); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.price_delete", "api_model_price", fmt.Sprint(priceID), nil)
		return id, okResult("删除模型价格成功", map[string]any{"price_id": priceID}), nil

	case "alert_rule_list":
		list, err := d.gatewayUC.ListAlertRules(ctx)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayAlertRuleForRPC(item))
		}
		return id, okResult("获取告警规则成功", map[string]any{"items": items}), nil

	case "alert_rule_upsert":
		enabled := true
		if _, ok := pm["enabled"]; ok {
			enabled = getBool(pm, "enabled", true)
		}
		item, err := d.gatewayUC.UpsertAlertRule(ctx, biz.GatewayAlertRule{
			ID:            getInt(pm, "id", 0),
			Name:          getString(pm, "name"),
			Metric:        getString(pm, "metric"),
			Operator:      getString(pm, "operator"),
			Threshold:     getFloat(pm, "threshold", 0),
			WindowSeconds: getInt64(pm, "window_seconds", 300),
			Enabled:       enabled,
		})
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.alert_rule_upsert", "api_alert_rule", fmt.Sprint(item.ID), map[string]any{
			"metric": item.Metric,
		})
		return id, okResult("保存告警规则成功", mapGatewayAlertRuleForRPC(item)), nil

	case "alert_rule_set_enabled":
		ruleID := getInt(pm, "rule_id", 0)
		enabled := getBool(pm, "enabled", true)
		if err := d.gatewayUC.SetAlertRuleEnabled(ctx, ruleID, enabled); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.alert_rule_set_enabled", "api_alert_rule", fmt.Sprint(ruleID), map[string]any{"enabled": enabled})
		return id, okResult("更新告警规则状态成功", map[string]any{"rule_id": ruleID, "enabled": enabled}), nil

	case "alert_rule_delete":
		ruleID := getInt(pm, "rule_id", 0)
		if err := d.gatewayUC.DeleteAlertRule(ctx, ruleID); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.alert_rule_delete", "api_alert_rule", fmt.Sprint(ruleID), nil)
		return id, okResult("删除告警规则成功", map[string]any{"rule_id": ruleID}), nil

	case "alert_event_list":
		status := strings.TrimSpace(getString(pm, "status"))
		limit := getInt(pm, "limit", 30)
		offset := getInt(pm, "offset", 0)
		list, total, err := d.gatewayUC.ListAlertEvents(ctx, status, limit, offset)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayAlertEventForRPC(item))
		}
		return id, okResult("获取告警事件成功", map[string]any{"items": items, "total": total, "limit": limit, "offset": offset}), nil

	case "alert_event_ack":
		eventID := getInt(pm, "event_id", 0)
		claims, _ := biz.GetClaimsFromContext(ctx)
		ackBy := 0
		if claims != nil {
			ackBy = claims.UserID
		}
		if err := d.gatewayUC.AckAlertEvent(ctx, eventID, ackBy); err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		d.auditGateway(ctx, "api.alert_event_ack", "api_alert_event", fmt.Sprint(eventID), nil)
		return id, okResult("确认告警事件成功", map[string]any{"event_id": eventID}), nil

	default:
		return id, &v1.JsonrpcResult{
			Code:    errcode.UnknownMethod.Code,
			Message: fmt.Sprintf("未知 API 转发接口 method=%s", method),
		}, nil
	}
}

func (d *JsonrpcData) mapGatewayError(ctx context.Context, err error) *v1.JsonrpcResult {
	d.log.WithContext(ctx).Warnf("[api] operation error=%v", err)
	switch {
	case errors.Is(err, biz.ErrBadParam):
		return &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: errcode.InvalidParam.Message}
	case errors.Is(err, biz.ErrGatewayAPIKeyNotFound):
		return &v1.JsonrpcResult{Code: errcode.APIKeyInvalid.Code, Message: errcode.APIKeyInvalid.Message}
	case errors.Is(err, biz.ErrGatewayAPIKeyDisabled):
		return &v1.JsonrpcResult{Code: errcode.APIKeyDisabled.Code, Message: errcode.APIKeyDisabled.Message}
	case errors.Is(err, biz.ErrGatewayModelDisabled):
		return &v1.JsonrpcResult{Code: errcode.APIModelDisabled.Code, Message: errcode.APIModelDisabled.Message}
	case errors.Is(err, biz.ErrGatewayModelNotAllowed):
		return &v1.JsonrpcResult{Code: errcode.APIModelNotAllowed.Code, Message: errcode.APIModelNotAllowed.Message}
	case errors.Is(err, biz.ErrGatewayModelCatalogFixed):
		return &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "模型列表为固定官方目录，不支持手工新增或删除"}
	case errors.Is(err, biz.ErrGatewayRateLimited):
		return &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: "API key 限流超限"}
	case errors.Is(err, biz.ErrGatewayQuotaExceeded):
		return &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: "API key 配额超限"}
	case errors.Is(err, biz.ErrGatewayPolicyInvalid):
		return &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "模型策略参数错误"}
	case errors.Is(err, biz.ErrGatewayExportRange):
		return &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "导出时间范围过大"}
	case errors.Is(err, biz.ErrGatewayUpstreamModeInvalid):
		return &v1.JsonrpcResult{Code: errcode.InvalidParam.Code, Message: "Codex 上游模式参数错误"}
	default:
		return &v1.JsonrpcResult{Code: errcode.APIOperationFailed.Code, Message: errcode.APIOperationFailed.Message}
	}
}

func (d *JsonrpcData) handleGatewayUser(ctx context.Context, method, id string, pm map[string]any) (string, *v1.JsonrpcResult, error) {
	claims, res := d.requireLogin(ctx)
	if res != nil {
		return id, res, nil
	}
	if claims == nil || claims.Role != biz.RoleUser {
		return id, &v1.JsonrpcResult{Code: errcode.AuthRequired.Code, Message: errcode.AuthRequired.Message}, nil
	}

	switch method {
	case "user_key_list":
		limit := getInt(pm, "limit", 30)
		offset := getInt(pm, "offset", 0)
		list, total, err := d.gatewayUC.ListAPIKeysByOwner(ctx, claims.UserID, limit, offset)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayAPIKeyForRPC(item, false))
		}
		return id, okResult("获取我的 API key 成功", map[string]any{"items": items, "total": total, "limit": limit, "offset": offset}), nil

	case "user_usage_summary":
		filter := gatewayUsageFilterFromParams(pm)
		filter.OwnerUserID = claims.UserID
		summary, err := d.gatewayUC.SummarizeUsage(ctx, filter)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		return id, okResult("获取我的 usage 概览成功", map[string]any{"summary": mapGatewaySummary(summary)}), nil

	case "user_usage_list":
		filter := gatewayUsageFilterFromParams(pm)
		filter.OwnerUserID = claims.UserID
		list, total, err := d.gatewayUC.ListUsageLogs(ctx, filter)
		if err != nil {
			return id, d.mapGatewayError(ctx, err), nil
		}
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, mapGatewayUsageForRPC(item))
		}
		return id, okResult("获取我的 usage 记录成功", map[string]any{"items": items, "total": total, "limit": filter.Limit, "offset": filter.Offset}), nil

	default:
		return id, &v1.JsonrpcResult{
			Code:    errcode.UnknownMethod.Code,
			Message: fmt.Sprintf("未知 API 用户接口 method=%s", method),
		}, nil
	}
}

func (d *JsonrpcData) auditGateway(ctx context.Context, action, targetType, targetID string, metadata map[string]any) {
	claims, _ := biz.GetClaimsFromContext(ctx)
	item := biz.GatewayAuditLog{
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   metadata,
	}
	if claims != nil {
		item.ActorID = claims.UserID
		item.ActorName = claims.Username
		if claims.Role == biz.RoleAdmin {
			item.ActorRole = "admin"
		} else {
			item.ActorRole = "user"
		}
	}
	d.gatewayUC.CreateAuditLog(ctx, item)
}

func okResult(message string, data map[string]any) *v1.JsonrpcResult {
	return &v1.JsonrpcResult{
		Code:    errcode.OK.Code,
		Message: message,
		Data:    newDataStruct(data),
	}
}

func gatewayUsageFilterFromParams(pm map[string]any) biz.GatewayUsageFilter {
	now := time.Now()
	start := time.Unix(getInt64(pm, "start_time", now.Add(-24*time.Hour).Unix()), 0)
	endRaw := getInt64(pm, "end_time", 0)
	var end time.Time
	if endRaw > 0 {
		end = time.Unix(endRaw, 0)
	}
	return biz.GatewayUsageFilter{
		Limit:        getInt(pm, "limit", 30),
		Offset:       getInt(pm, "offset", 0),
		KeyID:        getInt(pm, "key_id", 0),
		SessionID:    getString(pm, "session_id"),
		Model:        getString(pm, "model"),
		Endpoint:     getString(pm, "endpoint"),
		UpstreamMode: getString(pm, "upstream_mode"),
		StartTime:    start,
		EndTime:      end,
		SuccessSet: func() bool {
			_, ok := pm["success"]
			return ok
		}(),
		Success: getBool(pm, "success", false),
	}
}

func getStringList(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok || raw == nil {
		return []string{}
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return []string{}
		}
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	default:
		return []string{}
	}
}

func getIntList(m map[string]any, key string) []int {
	raw, ok := m[key]
	if !ok || raw == nil {
		return []int{}
	}
	switch v := raw.(type) {
	case []int:
		return v
	case []any:
		out := make([]int, 0, len(v))
		for _, item := range v {
			n := intFromAny(item)
			if n > 0 {
				out = append(out, n)
			}
		}
		return out
	case string:
		parts := strings.Split(v, ",")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			n := intFromAny(strings.TrimSpace(part))
			if n > 0 {
				out = append(out, n)
			}
		}
		return out
	default:
		return []int{}
	}
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return n
		}
	}
	return 0
}

func mapGatewayAPIKeyForRPC(item *biz.GatewayAPIKey, includePlainKey bool) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	data := map[string]any{
		"id":                                 item.ID,
		"owner_user_id":                      item.OwnerUserID,
		"name":                               item.Name,
		"key_prefix":                         item.KeyPrefix,
		"key_last4":                          item.KeyLast4,
		"disabled":                           item.Disabled,
		"quota_requests":                     item.QuotaRequests,
		"quota_total_tokens":                 item.QuotaTotalTokens,
		"quota_daily_tokens":                 item.QuotaDailyTokens,
		"quota_weekly_tokens":                item.QuotaWeeklyTokens,
		"quota_daily_input_tokens":           item.QuotaDailyInputTokens,
		"quota_weekly_input_tokens":          item.QuotaWeeklyInputTokens,
		"quota_daily_output_tokens":          item.QuotaDailyOutputTokens,
		"quota_weekly_output_tokens":         item.QuotaWeeklyOutputTokens,
		"quota_daily_billable_input_tokens":  item.QuotaDailyBillableInputTokens,
		"quota_weekly_billable_input_tokens": item.QuotaWeeklyBillableInputTokens,
		"allowed_models":                     stringListForRPC(item.AllowedModels),
		"created_at":                         item.CreatedAt.Unix(),
		"updated_at":                         item.UpdatedAt.Unix(),
	}
	if includePlainKey {
		data["plain_key"] = item.PlainKey
	}
	if item.LastUsedAt != nil {
		data["last_used_at"] = item.LastUsedAt.Unix()
	}
	return data
}

func stringListForRPC(items []string) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func mapGatewayModelForRPC(item *biz.GatewayModel) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	data := map[string]any{
		"id":           item.ID,
		"model_id":     item.ModelID,
		"owned_by":     item.OwnedBy,
		"created_unix": item.CreatedUnix,
		"enabled":      item.Enabled,
		"source":       item.Source,
		"created_at":   item.CreatedAt.Unix(),
		"updated_at":   item.UpdatedAt.Unix(),
	}
	if item.LastSeenAt != nil {
		data["last_seen_at"] = item.LastSeenAt.Unix()
	}
	return data
}

func mapGatewayUsageForRPC(item *biz.GatewayUsageLog) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	billableInputTokens := gatewayBillableInputTokens(item.InputTokens, item.CachedTokens)
	data := map[string]any{
		"id":                       item.ID,
		"api_key_id":               item.APIKeyID,
		"api_key_prefix":           item.APIKeyPrefix,
		"api_key_name":             item.APIKeyName,
		"session_id":               item.SessionID,
		"request_id":               item.RequestID,
		"method":                   item.Method,
		"path":                     item.Path,
		"endpoint":                 item.Endpoint,
		"model":                    item.Model,
		"status_code":              item.StatusCode,
		"success":                  item.Success,
		"stream":                   item.Stream,
		"input_tokens":             item.InputTokens,
		"output_tokens":            item.OutputTokens,
		"total_tokens":             item.TotalTokens,
		"cached_tokens":            item.CachedTokens,
		"cached_input_tokens":      item.CachedTokens,
		"billable_input_tokens":    billableInputTokens,
		"reasoning_tokens":         item.ReasoningTokens,
		"request_bytes":            item.RequestBytes,
		"response_bytes":           item.ResponseBytes,
		"duration_ms":              item.DurationMS,
		"upstream_configured_mode": item.UpstreamConfiguredMode,
		"upstream_mode":            item.UpstreamMode,
		"upstream_fallback":        item.UpstreamFallback,
		"upstream_error_type":      item.UpstreamErrorType,
		"error_type":               item.ErrorType,
		"created_at":               item.CreatedAt.Unix(),
	}
	if item.EstimatedCostUSD != nil {
		data["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		data["estimated_cost_usd"] = nil
	}
	return data
}

func mapGatewayUpstreamModeForRPC(mode string) map[string]any {
	mode = biz.NormalizeGatewayUpstreamMode(mode)
	if mode == "" {
		mode = biz.DefaultGatewayUpstreamMode()
	}
	return map[string]any{
		"mode":         mode,
		"default_mode": biz.DefaultGatewayUpstreamMode(),
		"options": []any{
			map[string]any{
				"label": "Backend 优先",
				"value": biz.GatewayUpstreamModeCodexBackend,
			},
			map[string]any{
				"label": "强制 CLI",
				"value": biz.GatewayUpstreamModeCodexCLI,
			},
		},
	}
}

func mapGatewaySummary(item *biz.GatewayUsageSummary) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	billableInputTokens := gatewayBillableInputTokens(item.InputTokens, item.CachedTokens)
	data := map[string]any{
		"total_requests":        item.TotalRequests,
		"success_requests":      item.SuccessRequests,
		"failed_requests":       item.FailedRequests,
		"total_tokens":          item.TotalTokens,
		"input_tokens":          item.InputTokens,
		"output_tokens":         item.OutputTokens,
		"cached_tokens":         item.CachedTokens,
		"cached_input_tokens":   item.CachedTokens,
		"billable_input_tokens": billableInputTokens,
		"reasoning_tokens":      item.ReasoningTokens,
		"total_bytes_in":        item.TotalBytesIn,
		"total_bytes_out":       item.TotalBytesOut,
		"average_duration_ms":   item.AverageDurationMS,
		"backend_requests":      item.BackendRequests,
		"cli_requests":          item.CLIRequests,
		"fallback_requests":     item.FallbackRequests,
	}
	if item.EstimatedCostUSD != nil {
		data["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		data["estimated_cost_usd"] = nil
	}
	return data
}

func mapGatewayUsageBucketForRPC(item *biz.GatewayUsageBucket) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	billableInputTokens := gatewayBillableInputTokens(item.InputTokens, item.CachedTokens)
	data := map[string]any{
		"bucket_start":          item.BucketStart.Unix(),
		"model":                 item.Model,
		"total_requests":        item.TotalRequests,
		"success_requests":      item.SuccessRequests,
		"failed_requests":       item.FailedRequests,
		"total_tokens":          item.TotalTokens,
		"input_tokens":          item.InputTokens,
		"output_tokens":         item.OutputTokens,
		"cached_tokens":         item.CachedTokens,
		"cached_input_tokens":   item.CachedTokens,
		"billable_input_tokens": billableInputTokens,
		"reasoning_tokens":      item.ReasoningTokens,
		"total_bytes_in":        item.TotalBytesIn,
		"total_bytes_out":       item.TotalBytesOut,
		"average_duration_ms":   item.AverageDurationMS,
		"backend_requests":      item.BackendRequests,
		"cli_requests":          item.CLIRequests,
		"fallback_requests":     item.FallbackRequests,
	}
	if item.EstimatedCostUSD != nil {
		data["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		data["estimated_cost_usd"] = nil
	}
	return data
}

func mapGatewayUsageKeySummaryForRPC(item *biz.GatewayUsageKeySummary) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	billableInputTokens := gatewayBillableInputTokens(item.InputTokens, item.CachedTokens)
	data := map[string]any{
		"api_key_id":            item.APIKeyID,
		"api_key_prefix":        item.APIKeyPrefix,
		"api_key_name":          item.APIKeyName,
		"disabled":              item.Disabled,
		"total_requests":        item.TotalRequests,
		"success_requests":      item.SuccessRequests,
		"failed_requests":       item.FailedRequests,
		"total_tokens":          item.TotalTokens,
		"input_tokens":          item.InputTokens,
		"output_tokens":         item.OutputTokens,
		"cached_tokens":         item.CachedTokens,
		"cached_input_tokens":   item.CachedTokens,
		"billable_input_tokens": billableInputTokens,
		"reasoning_tokens":      item.ReasoningTokens,
		"average_duration_ms":   item.AverageDurationMS,
		"backend_requests":      item.BackendRequests,
		"cli_requests":          item.CLIRequests,
		"fallback_requests":     item.FallbackRequests,
	}
	if item.EstimatedCostUSD != nil {
		data["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		data["estimated_cost_usd"] = nil
	}
	return data
}

func mapGatewayUsageSessionSummaryForRPC(item *biz.GatewayUsageSessionSummary) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	billableInputTokens := gatewayBillableInputTokens(item.InputTokens, item.CachedTokens)
	data := map[string]any{
		"session_id":            item.SessionID,
		"api_key_id":            item.APIKeyID,
		"api_key_prefix":        item.APIKeyPrefix,
		"api_key_name":          item.APIKeyName,
		"total_requests":        item.TotalRequests,
		"success_requests":      item.SuccessRequests,
		"failed_requests":       item.FailedRequests,
		"total_tokens":          item.TotalTokens,
		"input_tokens":          item.InputTokens,
		"output_tokens":         item.OutputTokens,
		"cached_tokens":         item.CachedTokens,
		"cached_input_tokens":   item.CachedTokens,
		"billable_input_tokens": billableInputTokens,
		"reasoning_tokens":      item.ReasoningTokens,
		"average_duration_ms":   item.AverageDurationMS,
		"backend_requests":      item.BackendRequests,
		"cli_requests":          item.CLIRequests,
		"fallback_requests":     item.FallbackRequests,
		"first_seen_at":         item.FirstSeenAt.Unix(),
		"last_seen_at":          item.LastSeenAt.Unix(),
	}
	if item.EstimatedCostUSD != nil {
		data["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		data["estimated_cost_usd"] = nil
	}
	return data
}

func gatewayBillableInputTokens(inputTokens, cachedTokens int64) int64 {
	if inputTokens <= 0 {
		return 0
	}
	if cachedTokens <= 0 {
		return inputTokens
	}
	if cachedTokens >= inputTokens {
		return 0
	}
	return inputTokens - cachedTokens
}

func mapGatewayPolicyForRPC(item *biz.GatewayPolicy) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":               item.ID,
		"api_key_id":       item.APIKeyID,
		"model_id":         item.ModelID,
		"rpm":              item.RPM,
		"tpm":              item.TPM,
		"daily_requests":   item.DailyRequests,
		"monthly_requests": item.MonthlyRequests,
		"daily_tokens":     item.DailyTokens,
		"monthly_tokens":   item.MonthlyTokens,
		"created_at":       item.CreatedAt.Unix(),
		"updated_at":       item.UpdatedAt.Unix(),
	}
}

func mapGatewayPriceForRPC(item *biz.GatewayModelPrice) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	createdAt := int64(0)
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.Unix()
	}
	updatedAt := int64(0)
	if !item.UpdatedAt.IsZero() {
		updatedAt = item.UpdatedAt.Unix()
	}
	return map[string]any{
		"id":                            item.ID,
		"model_id":                      item.ModelID,
		"input_usd_per_million":         item.InputUSDPerMillion,
		"cached_input_usd_per_million":  item.CachedInputUSDPerMillion,
		"output_usd_per_million":        item.OutputUSDPerMillion,
		"reasoning_tokens_billed_as":    "output_tokens",
		"reasoning_tokens_double_count": false,
		"created_at":                    createdAt,
		"updated_at":                    updatedAt,
	}
}

func mapGatewayAlertRuleForRPC(item *biz.GatewayAlertRule) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":             item.ID,
		"name":           item.Name,
		"metric":         item.Metric,
		"operator":       item.Operator,
		"threshold":      item.Threshold,
		"window_seconds": item.WindowSeconds,
		"enabled":        item.Enabled,
		"created_at":     item.CreatedAt.Unix(),
		"updated_at":     item.UpdatedAt.Unix(),
	}
}

func mapGatewayAlertEventForRPC(item *biz.GatewayAlertEvent) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	data := map[string]any{
		"id":         item.ID,
		"rule_id":    item.RuleID,
		"rule_name":  item.RuleName,
		"metric":     item.Metric,
		"value":      item.Value,
		"threshold":  item.Threshold,
		"status":     item.Status,
		"ack_by":     item.AckBy,
		"created_at": item.CreatedAt.Unix(),
	}
	if item.AckAt != nil {
		data["ack_at"] = item.AckAt.Unix()
	}
	return data
}
