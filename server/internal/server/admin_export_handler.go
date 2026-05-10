package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"server/internal/biz"
	"server/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	httpx "github.com/go-kratos/kratos/v2/transport/http"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const defaultGatewayExportMaxDays = 90

func registerAdminExportRoutes(
	srv *httpx.Server,
	logger log.Logger,
	tp *sdktrace.TracerProvider,
	gatewayUC *biz.GatewayUsecase,
	dataCfg *conf.Data,
) {
	helper := log.NewHelper(log.With(logger, "logger.name", "server.http.admin_export"))
	handler := &adminExportHandler{dataCfg: dataCfg, log: helper, gatewayUC: gatewayUC}

	srv.Handle("/admin/exports/usage.csv", newObservedHTTPHandler(logger, tp, "server.http.admin_export.usage_csv", handler.handleUsageCSV))
	srv.Handle("/admin/exports/usage.json", newObservedHTTPHandler(logger, tp, "server.http.admin_export.usage_json", handler.handleUsageJSON))
}

type adminExportHandler struct {
	dataCfg   *conf.Data
	log       *log.Helper
	gatewayUC *biz.GatewayUsecase
}

func (h *adminExportHandler) handleUsageCSV(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	filter, ok := h.requireAdminAndBuildFilter(ctx, w, r)
	if !ok {
		return
	}
	items, err := h.collectUsage(ctx, filter)
	if err != nil {
		writePlainText(w, stdhttp.StatusInternalServerError, "导出 usage 失败")
		return
	}
	h.auditExport(ctx, "csv", len(items), filter)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gateway-usage.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"created_at", "api_key_id", "api_key_prefix", "endpoint", "model", "status_code", "success",
		"input_tokens", "cached_tokens", "output_tokens", "reasoning_tokens", "total_tokens",
		"estimated_cost_usd", "duration_ms", "error_type",
	})
	for _, item := range items {
		cost := ""
		if item.EstimatedCostUSD != nil {
			cost = fmt.Sprintf("%.8f", *item.EstimatedCostUSD)
		}
		_ = writer.Write([]string{
			item.CreatedAt.Format(time.RFC3339),
			strconv.Itoa(item.APIKeyID),
			item.APIKeyPrefix,
			item.Endpoint,
			item.Model,
			strconv.Itoa(item.StatusCode),
			strconv.FormatBool(item.Success),
			strconv.FormatInt(item.InputTokens, 10),
			strconv.FormatInt(item.CachedTokens, 10),
			strconv.FormatInt(item.OutputTokens, 10),
			strconv.FormatInt(item.ReasoningTokens, 10),
			strconv.FormatInt(item.TotalTokens, 10),
			cost,
			strconv.FormatInt(item.DurationMS, 10),
			item.ErrorType,
		})
	}
	writer.Flush()
}

func (h *adminExportHandler) handleUsageJSON(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) {
	filter, ok := h.requireAdminAndBuildFilter(ctx, w, r)
	if !ok {
		return
	}
	items, err := h.collectUsage(ctx, filter)
	if err != nil {
		writePlainText(w, stdhttp.StatusInternalServerError, "导出 usage 失败")
		return
	}
	h.auditExport(ctx, "json", len(items), filter)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapUsageExportRow(item))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": rows,
		"total": len(rows),
	})
}

func (h *adminExportHandler) requireAdminAndBuildFilter(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) (biz.GatewayUsageFilter, bool) {
	claims, ok := biz.GetClaimsFromContext(ctx)
	if !ok || claims == nil || claims.Role != biz.RoleAdmin {
		writePlainText(w, stdhttp.StatusForbidden, "需要管理员权限")
		return biz.GatewayUsageFilter{}, false
	}

	filter := biz.GatewayUsageFilter{
		Limit:        200,
		KeyID:        queryInt(r, "key_id"),
		Model:        strings.TrimSpace(r.URL.Query().Get("model")),
		Endpoint:     strings.TrimSpace(r.URL.Query().Get("endpoint")),
		UpstreamMode: strings.TrimSpace(r.URL.Query().Get("upstream_mode")),
		StartTime:    queryUnixTime(r, "start_time", time.Now().Add(-24*time.Hour)),
		EndTime:      queryUnixTime(r, "end_time", time.Now()),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("success")); raw != "" {
		filter.SuccessSet = true
		filter.Success = raw == "true" || raw == "1"
	}
	if filter.EndTime.Sub(filter.StartTime) > time.Duration(h.exportMaxDays())*24*time.Hour {
		writePlainText(w, stdhttp.StatusBadRequest, "导出时间范围过大")
		return biz.GatewayUsageFilter{}, false
	}
	return filter, true
}

func (h *adminExportHandler) exportMaxDays() int {
	if h.dataCfg != nil && h.dataCfg.Api != nil && h.dataCfg.Api.ExportMaxDays > 0 {
		return int(h.dataCfg.Api.ExportMaxDays)
	}
	return defaultGatewayExportMaxDays
}

func (h *adminExportHandler) collectUsage(ctx context.Context, filter biz.GatewayUsageFilter) ([]*biz.GatewayUsageLog, error) {
	out := make([]*biz.GatewayUsageLog, 0)
	for {
		filter.Offset = len(out)
		items, total, err := h.gatewayUC.ListUsageLogs(ctx, filter)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= total || len(items) == 0 {
			return out, nil
		}
	}
}

func (h *adminExportHandler) auditExport(ctx context.Context, format string, count int, filter biz.GatewayUsageFilter) {
	claims, _ := biz.GetClaimsFromContext(ctx)
	item := biz.GatewayAuditLog{
		Action:     "api.usage_export",
		TargetType: "gateway_usage",
		TargetID:   format,
		Metadata: map[string]any{
			"format":     format,
			"count":      count,
			"key_id":     filter.KeyID,
			"model":      filter.Model,
			"endpoint":   filter.Endpoint,
			"start_time": filter.StartTime.Unix(),
			"end_time":   filter.EndTime.Unix(),
		},
	}
	if claims != nil {
		item.ActorID = claims.UserID
		item.ActorName = claims.Username
		item.ActorRole = "admin"
	}
	h.gatewayUC.CreateAuditLog(ctx, item)
}

func queryInt(r *stdhttp.Request, key string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(key)))
	return n
}

func queryUnixTime(r *stdhttp.Request, key string, fallback time.Time) time.Time {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Unix(n, 0)
}

func mapUsageExportRow(item *biz.GatewayUsageLog) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	row := map[string]any{
		"api_key_id":               item.APIKeyID,
		"api_key_prefix":           item.APIKeyPrefix,
		"cached_tokens":            item.CachedTokens,
		"created_at":               item.CreatedAt.Format(time.RFC3339),
		"duration_ms":              item.DurationMS,
		"endpoint":                 item.Endpoint,
		"error_type":               item.ErrorType,
		"id":                       item.ID,
		"input_tokens":             item.InputTokens,
		"model":                    item.Model,
		"output_tokens":            item.OutputTokens,
		"path":                     item.Path,
		"reasoning_tokens":         item.ReasoningTokens,
		"request_id":               item.RequestID,
		"status_code":              item.StatusCode,
		"success":                  item.Success,
		"total_tokens":             item.TotalTokens,
		"upstream_configured_mode": item.UpstreamConfiguredMode,
		"upstream_error_type":      item.UpstreamErrorType,
		"upstream_fallback":        item.UpstreamFallback,
		"upstream_mode":            item.UpstreamMode,
	}
	if item.EstimatedCostUSD != nil {
		row["estimated_cost_usd"] = *item.EstimatedCostUSD
	} else {
		row["estimated_cost_usd"] = nil
	}
	return row
}
