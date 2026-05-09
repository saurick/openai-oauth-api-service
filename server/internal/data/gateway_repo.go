package data

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"server/internal/biz"
	"server/internal/data/model/ent"
	"server/internal/data/model/ent/gatewayalertevent"
	"server/internal/data/model/ent/gatewayalertrule"
	"server/internal/data/model/ent/gatewayapikey"
	"server/internal/data/model/ent/gatewaymodel"
	"server/internal/data/model/ent/gatewaymodelprice"
	"server/internal/data/model/ent/gatewaypolicy"
	"server/internal/data/model/ent/gatewayusagelog"
	"server/internal/data/model/ent/predicate"

	"github.com/go-kratos/kratos/v2/log"
)

type gatewayRepo struct {
	data *Data
	log  *log.Helper
}

func NewGatewayRepo(data *Data, logger log.Logger) *gatewayRepo {
	return &gatewayRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data.api")),
	}
}

var _ biz.GatewayRepo = (*gatewayRepo)(nil)

func (r *gatewayRepo) CreateAPIKey(ctx context.Context, input biz.CreateGatewayAPIKeyInput, secret biz.GatewayAPIKeySecret) (*biz.GatewayAPIKey, error) {
	create := r.data.postgres.GatewayAPIKey.Create().
		SetName(input.Name).
		SetKeyHash(secret.KeyHash).
		SetPlainKey(secret.PlainKey).
		SetKeyPrefix(secret.KeyPrefix).
		SetKeyLast4(secret.KeyLast4).
		SetQuotaRequests(input.QuotaRequests).
		SetQuotaTotalTokens(input.QuotaTotalTokens).
		SetQuotaDailyTokens(input.QuotaDailyTokens).
		SetQuotaWeeklyTokens(input.QuotaWeeklyTokens).
		SetAllowedModels(normalizeStringList(input.AllowedModels))
	if input.OwnerUserID > 0 {
		create.SetOwnerUserID(input.OwnerUserID)
	}
	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayAPIKey(item), nil
}

func (r *gatewayRepo) ListAPIKeys(ctx context.Context, limit, offset int, search string) ([]*biz.GatewayAPIKey, int, error) {
	q := r.data.postgres.GatewayAPIKey.Query()
	if search != "" {
		predicates := []predicate.GatewayAPIKey{
			gatewayapikey.NameContainsFold(search),
			gatewayapikey.PlainKeyContainsFold(search),
			gatewayapikey.KeyPrefixContainsFold(search),
			gatewayapikey.KeyLast4ContainsFold(search),
		}
		if id, err := strconv.Atoi(search); err == nil && id > 0 {
			predicates = append(predicates, gatewayapikey.IDEQ(id))
		}
		q = q.Where(gatewayapikey.Or(predicates...))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := q.Order(ent.Desc(gatewayapikey.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*biz.GatewayAPIKey, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayAPIKey(item))
	}
	return out, total, nil
}

func (r *gatewayRepo) ListAPIKeysByOwner(ctx context.Context, ownerUserID, limit, offset int) ([]*biz.GatewayAPIKey, int, error) {
	q := r.data.postgres.GatewayAPIKey.Query().
		Where(gatewayapikey.OwnerUserIDEQ(ownerUserID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := q.Order(ent.Desc(gatewayapikey.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*biz.GatewayAPIKey, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayAPIKey(item))
	}
	return out, total, nil
}

func (r *gatewayRepo) UpdateAPIKey(ctx context.Context, input biz.UpdateGatewayAPIKeyInput) (*biz.GatewayAPIKey, error) {
	update := r.data.postgres.GatewayAPIKey.UpdateOneID(input.ID).
		SetName(input.Name).
		SetQuotaRequests(input.QuotaRequests).
		SetQuotaTotalTokens(input.QuotaTotalTokens).
		SetQuotaDailyTokens(input.QuotaDailyTokens).
		SetQuotaWeeklyTokens(input.QuotaWeeklyTokens).
		SetAllowedModels(normalizeStringList(input.AllowedModels)).
		SetDisabled(input.Disabled)
	if input.OwnerUserID > 0 {
		update.SetOwnerUserID(input.OwnerUserID)
	} else {
		update.ClearOwnerUserID()
	}
	item, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayAPIKey(item), nil
}

func (r *gatewayRepo) DeleteAPIKey(ctx context.Context, id int) error {
	if _, err := r.data.postgres.GatewayPolicy.Delete().
		Where(gatewaypolicy.APIKeyIDEQ(id)).
		Exec(ctx); err != nil {
		return err
	}
	return r.data.postgres.GatewayAPIKey.DeleteOneID(id).Exec(ctx)
}

func (r *gatewayRepo) DeleteAPIKeys(ctx context.Context, ids []int) (int, error) {
	if _, err := r.data.postgres.GatewayPolicy.Delete().
		Where(gatewaypolicy.APIKeyIDIn(ids...)).
		Exec(ctx); err != nil {
		return 0, err
	}
	return r.data.postgres.GatewayAPIKey.Delete().
		Where(gatewayapikey.IDIn(ids...)).
		Exec(ctx)
}

func (r *gatewayRepo) SetAPIKeyDisabled(ctx context.Context, id int, disabled bool) error {
	return r.data.postgres.GatewayAPIKey.UpdateOneID(id).
		SetDisabled(disabled).
		Exec(ctx)
}

func (r *gatewayRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*biz.GatewayAPIKey, error) {
	item, err := r.data.postgres.GatewayAPIKey.Query().
		Where(gatewayapikey.KeyHashEQ(keyHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrGatewayAPIKeyNotFound
		}
		return nil, err
	}
	return mapGatewayAPIKey(item), nil
}

func (r *gatewayRepo) TouchAPIKeyUsed(ctx context.Context, id int, usedAt time.Time) error {
	return r.data.postgres.GatewayAPIKey.UpdateOneID(id).
		SetLastUsedAt(usedAt).
		Exec(ctx)
}

func (r *gatewayRepo) CreateUsageLog(ctx context.Context, item *biz.GatewayUsageLog) error {
	if item == nil {
		return biz.ErrBadParam
	}

	create := r.data.postgres.GatewayUsageLog.Create().
		SetAPIKeyPrefix(item.APIKeyPrefix).
		SetRequestID(item.RequestID).
		SetMethod(item.Method).
		SetPath(item.Path).
		SetEndpoint(item.Endpoint).
		SetModel(item.Model).
		SetStatusCode(item.StatusCode).
		SetSuccess(item.Success).
		SetStream(item.Stream).
		SetInputTokens(item.InputTokens).
		SetOutputTokens(item.OutputTokens).
		SetTotalTokens(item.TotalTokens).
		SetCachedTokens(item.CachedTokens).
		SetReasoningTokens(item.ReasoningTokens).
		SetRequestBytes(item.RequestBytes).
		SetResponseBytes(item.ResponseBytes).
		SetDurationMs(item.DurationMS).
		SetErrorType(item.ErrorType)
	if item.APIKeyID > 0 {
		create.SetAPIKeyID(item.APIKeyID)
	}
	if !item.CreatedAt.IsZero() {
		create.SetCreatedAt(item.CreatedAt)
	}

	return create.Exec(ctx)
}

func (r *gatewayRepo) ListUsageLogs(ctx context.Context, filter biz.GatewayUsageFilter) ([]*biz.GatewayUsageLog, int, error) {
	q := r.applyUsageFilter(ctx, r.data.postgres.GatewayUsageLog.Query(), filter)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := q.Order(ent.Desc(gatewayusagelog.FieldCreatedAt)).
		Limit(filter.Limit).
		Offset(filter.Offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*biz.GatewayUsageLog, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayUsageLog(item))
	}
	r.attachUsageCosts(ctx, out)
	return out, total, nil
}

func (r *gatewayRepo) attachUsageCosts(ctx context.Context, list []*biz.GatewayUsageLog) {
	if len(list) == 0 {
		return
	}
	prices, err := r.listModelPriceMap(ctx)
	if err != nil || len(prices) == 0 {
		return
	}
	for _, item := range list {
		price, ok := prices[item.Model]
		if !ok || price == nil {
			continue
		}
		item.EstimatedCostUSD = estimateCostForTokens(item.InputTokens, item.CachedTokens, item.OutputTokens, price)
	}
}

func (r *gatewayRepo) SummarizeUsage(ctx context.Context, filter biz.GatewayUsageFilter) (*biz.GatewayUsageSummary, error) {
	where, args := buildUsageWhereClause(filter)
	query := `SELECT
COUNT(*),
COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN success THEN 0 ELSE 1 END), 0),
COALESCE(SUM(total_tokens), 0),
COALESCE(SUM(input_tokens), 0),
COALESCE(SUM(output_tokens), 0),
COALESCE(SUM(cached_tokens), 0),
COALESCE(SUM(reasoning_tokens), 0),
COALESCE(SUM(request_bytes), 0),
COALESCE(SUM(response_bytes), 0),
COALESCE(AVG(duration_ms)::bigint, 0)
FROM gateway_usage_logs` + where

	var out biz.GatewayUsageSummary
	if err := r.data.sqldb.QueryRowContext(ctx, query, args...).Scan(
		&out.TotalRequests,
		&out.SuccessRequests,
		&out.FailedRequests,
		&out.TotalTokens,
		&out.InputTokens,
		&out.OutputTokens,
		&out.CachedTokens,
		&out.ReasoningTokens,
		&out.TotalBytesIn,
		&out.TotalBytesOut,
		&out.AverageDurationMS,
	); err != nil {
		return nil, err
	}
	out.EstimatedCostUSD = r.estimateUsageCost(ctx, filter)
	return &out, nil
}

func (r *gatewayRepo) ListUsageBuckets(ctx context.Context, filter biz.GatewayUsageFilter, groupBy string) ([]*biz.GatewayUsageBucket, error) {
	if groupBy != "day" {
		return nil, biz.ErrBadParam
	}

	where, args := buildUsageWhereClause(filter)
	query := `SELECT
DATE_TRUNC('day', created_at) AS bucket_start,
COUNT(*),
COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN success THEN 0 ELSE 1 END), 0),
COALESCE(SUM(total_tokens), 0),
COALESCE(SUM(input_tokens), 0),
COALESCE(SUM(output_tokens), 0),
COALESCE(SUM(cached_tokens), 0),
COALESCE(SUM(reasoning_tokens), 0),
COALESCE(SUM(request_bytes), 0),
COALESCE(SUM(response_bytes), 0),
COALESCE(AVG(duration_ms)::bigint, 0)
FROM gateway_usage_logs` + where + `
GROUP BY bucket_start
ORDER BY bucket_start ASC`

	rows, err := r.data.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.log.WithContext(ctx).Warnf("close usage bucket rows failed err=%v", closeErr)
		}
	}()

	out := make([]*biz.GatewayUsageBucket, 0)
	for rows.Next() {
		var item biz.GatewayUsageBucket
		if err := rows.Scan(
			&item.BucketStart,
			&item.TotalRequests,
			&item.SuccessRequests,
			&item.FailedRequests,
			&item.TotalTokens,
			&item.InputTokens,
			&item.OutputTokens,
			&item.CachedTokens,
			&item.ReasoningTokens,
			&item.TotalBytesIn,
			&item.TotalBytesOut,
			&item.AverageDurationMS,
		); err != nil {
			return nil, err
		}
		costFilter := filter
		costFilter.StartTime = item.BucketStart
		costFilter.EndTime = item.BucketStart.Add(24 * time.Hour)
		item.EstimatedCostUSD = r.estimateUsageCost(ctx, costFilter)
		out = append(out, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *gatewayRepo) ListUsageKeySummaries(ctx context.Context, filter biz.GatewayUsageFilter, limit int) ([]*biz.GatewayUsageKeySummary, error) {
	where, args := buildUsageWhereClauseWithPrefix(filter, "l.")
	args = append(args, limit)
	query := `SELECT
COALESCE(l.api_key_id, 0),
COALESCE(MAX(k.key_prefix), MAX(NULLIF(l.api_key_prefix, '')), ''),
COALESCE(MAX(k.name), ''),
COALESCE(BOOL_OR(k.disabled), false),
COUNT(*),
COALESCE(SUM(CASE WHEN l.success THEN 1 ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN l.success THEN 0 ELSE 1 END), 0),
COALESCE(SUM(l.total_tokens), 0),
COALESCE(SUM(l.input_tokens), 0),
COALESCE(SUM(l.output_tokens), 0),
COALESCE(SUM(l.cached_tokens), 0),
COALESCE(SUM(l.reasoning_tokens), 0),
COALESCE(AVG(l.duration_ms)::bigint, 0)
FROM gateway_usage_logs l
LEFT JOIN gateway_api_keys k ON k.id = l.api_key_id` + where + `
GROUP BY COALESCE(l.api_key_id, 0)
ORDER BY COALESCE(SUM(l.total_tokens), 0) DESC, COUNT(*) DESC
LIMIT $` + fmt.Sprint(len(args))

	rows, err := r.data.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.log.WithContext(ctx).Warnf("close usage key summary rows failed err=%v", closeErr)
		}
	}()

	out := make([]*biz.GatewayUsageKeySummary, 0)
	for rows.Next() {
		var item biz.GatewayUsageKeySummary
		if err := rows.Scan(
			&item.APIKeyID,
			&item.APIKeyPrefix,
			&item.APIKeyName,
			&item.Disabled,
			&item.TotalRequests,
			&item.SuccessRequests,
			&item.FailedRequests,
			&item.TotalTokens,
			&item.InputTokens,
			&item.OutputTokens,
			&item.CachedTokens,
			&item.ReasoningTokens,
			&item.AverageDurationMS,
		); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.attachUsageKeySummaryCosts(ctx, filter, out)
	return out, nil
}

func (r *gatewayRepo) GetPolicyForKeyModel(ctx context.Context, apiKeyID int, modelID string) (*biz.GatewayPolicy, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID != "" {
		item, err := r.data.postgres.GatewayPolicy.Query().
			Where(gatewaypolicy.APIKeyIDEQ(apiKeyID), gatewaypolicy.ModelIDEQ(modelID)).
			Only(ctx)
		if err == nil {
			return mapGatewayPolicy(item), nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	item, err := r.data.postgres.GatewayPolicy.Query().
		Where(gatewaypolicy.APIKeyIDEQ(apiKeyID), gatewaypolicy.ModelIDEQ("*")).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return mapGatewayPolicy(item), nil
}

func (r *gatewayRepo) ListPolicies(ctx context.Context, apiKeyID int) ([]*biz.GatewayPolicy, error) {
	q := r.data.postgres.GatewayPolicy.Query()
	if apiKeyID > 0 {
		q = q.Where(gatewaypolicy.APIKeyIDEQ(apiKeyID))
	}
	list, err := q.Order(ent.Asc(gatewaypolicy.FieldAPIKeyID), ent.Asc(gatewaypolicy.FieldModelID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.GatewayPolicy, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayPolicy(item))
	}
	return out, nil
}

func (r *gatewayRepo) UpsertPolicy(ctx context.Context, item biz.GatewayPolicy) (*biz.GatewayPolicy, error) {
	existing, err := r.data.postgres.GatewayPolicy.Query().
		Where(gatewaypolicy.APIKeyIDEQ(item.APIKeyID), gatewaypolicy.ModelIDEQ(item.ModelID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if existing == nil {
		created, err := r.data.postgres.GatewayPolicy.Create().
			SetAPIKeyID(item.APIKeyID).
			SetModelID(item.ModelID).
			SetRpm(item.RPM).
			SetTpm(item.TPM).
			SetDailyRequests(item.DailyRequests).
			SetMonthlyRequests(item.MonthlyRequests).
			SetDailyTokens(item.DailyTokens).
			SetMonthlyTokens(item.MonthlyTokens).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return mapGatewayPolicy(created), nil
	}
	updated, err := r.data.postgres.GatewayPolicy.UpdateOneID(existing.ID).
		SetRpm(item.RPM).
		SetTpm(item.TPM).
		SetDailyRequests(item.DailyRequests).
		SetMonthlyRequests(item.MonthlyRequests).
		SetDailyTokens(item.DailyTokens).
		SetMonthlyTokens(item.MonthlyTokens).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayPolicy(updated), nil
}

func (r *gatewayRepo) DeletePolicy(ctx context.Context, id int) error {
	return r.data.postgres.GatewayPolicy.DeleteOneID(id).Exec(ctx)
}

func (r *gatewayRepo) ListModelPrices(ctx context.Context) ([]*biz.GatewayModelPrice, error) {
	list, err := r.data.postgres.GatewayModelPrice.Query().
		Order(ent.Asc(gatewaymodelprice.FieldModelID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.GatewayModelPrice, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayModelPrice(item))
	}
	return out, nil
}

func (r *gatewayRepo) UpsertModelPrice(ctx context.Context, item biz.GatewayModelPrice) (*biz.GatewayModelPrice, error) {
	existing, err := r.data.postgres.GatewayModelPrice.Query().
		Where(gatewaymodelprice.ModelIDEQ(item.ModelID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if existing == nil {
		created, err := r.data.postgres.GatewayModelPrice.Create().
			SetModelID(item.ModelID).
			SetInputUsdPerMillion(item.InputUSDPerMillion).
			SetCachedInputUsdPerMillion(item.CachedInputUSDPerMillion).
			SetOutputUsdPerMillion(item.OutputUSDPerMillion).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return mapGatewayModelPrice(created), nil
	}
	updated, err := r.data.postgres.GatewayModelPrice.UpdateOneID(existing.ID).
		SetInputUsdPerMillion(item.InputUSDPerMillion).
		SetCachedInputUsdPerMillion(item.CachedInputUSDPerMillion).
		SetOutputUsdPerMillion(item.OutputUSDPerMillion).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayModelPrice(updated), nil
}

func (r *gatewayRepo) DeleteModelPrice(ctx context.Context, id int) error {
	return r.data.postgres.GatewayModelPrice.DeleteOneID(id).Exec(ctx)
}

func (r *gatewayRepo) CreateAuditLog(ctx context.Context, item biz.GatewayAuditLog) error {
	return r.data.postgres.GatewayAuditLog.Create().
		SetActorID(item.ActorID).
		SetActorName(item.ActorName).
		SetActorRole(item.ActorRole).
		SetAction(item.Action).
		SetTargetType(item.TargetType).
		SetTargetID(item.TargetID).
		SetMetadata(sanitizeAuditMetadata(item.Metadata)).
		Exec(ctx)
}

func (r *gatewayRepo) ListAlertRules(ctx context.Context) ([]*biz.GatewayAlertRule, error) {
	list, err := r.data.postgres.GatewayAlertRule.Query().
		Order(ent.Desc(gatewayalertrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.GatewayAlertRule, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayAlertRule(item))
	}
	return out, nil
}

func (r *gatewayRepo) UpsertAlertRule(ctx context.Context, item biz.GatewayAlertRule) (*biz.GatewayAlertRule, error) {
	if item.ID <= 0 {
		created, err := r.data.postgres.GatewayAlertRule.Create().
			SetName(item.Name).
			SetMetric(item.Metric).
			SetOperator(item.Operator).
			SetThreshold(item.Threshold).
			SetWindowSeconds(item.WindowSeconds).
			SetEnabled(item.Enabled).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return mapGatewayAlertRule(created), nil
	}
	updated, err := r.data.postgres.GatewayAlertRule.UpdateOneID(item.ID).
		SetName(item.Name).
		SetMetric(item.Metric).
		SetOperator(item.Operator).
		SetThreshold(item.Threshold).
		SetWindowSeconds(item.WindowSeconds).
		SetEnabled(item.Enabled).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayAlertRule(updated), nil
}

func (r *gatewayRepo) DeleteAlertRule(ctx context.Context, id int) error {
	if _, err := r.data.postgres.GatewayAlertEvent.Delete().
		Where(gatewayalertevent.RuleIDEQ(id)).
		Exec(ctx); err != nil {
		return err
	}
	return r.data.postgres.GatewayAlertRule.DeleteOneID(id).Exec(ctx)
}

func (r *gatewayRepo) SetAlertRuleEnabled(ctx context.Context, id int, enabled bool) error {
	return r.data.postgres.GatewayAlertRule.UpdateOneID(id).
		SetEnabled(enabled).
		Exec(ctx)
}

func (r *gatewayRepo) ListAlertEvents(ctx context.Context, status string, limit, offset int) ([]*biz.GatewayAlertEvent, int, error) {
	q := r.data.postgres.GatewayAlertEvent.Query()
	if status != "" {
		q = q.Where(gatewayalertevent.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := q.Order(ent.Desc(gatewayalertevent.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.GatewayAlertEvent, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayAlertEvent(item))
	}
	return out, total, nil
}

func (r *gatewayRepo) AckAlertEvent(ctx context.Context, id, ackBy int, ackAt time.Time) error {
	return r.data.postgres.GatewayAlertEvent.UpdateOneID(id).
		SetStatus("acknowledged").
		SetAckBy(ackBy).
		SetAckAt(ackAt).
		Exec(ctx)
}

func (r *gatewayRepo) CreateAlertEvent(ctx context.Context, item biz.GatewayAlertEvent) error {
	create := r.data.postgres.GatewayAlertEvent.Create().
		SetRuleName(item.RuleName).
		SetMetric(item.Metric).
		SetValue(item.Value).
		SetThreshold(item.Threshold).
		SetStatus(item.Status)
	if item.RuleID > 0 {
		create.SetRuleID(item.RuleID)
	}
	return create.Exec(ctx)
}

func (r *gatewayRepo) ListModels(ctx context.Context, limit, offset int, enabledOnly bool, search string) ([]*biz.GatewayModel, int, error) {
	q := r.data.postgres.GatewayModel.Query()
	if enabledOnly {
		q = q.Where(gatewaymodel.EnabledEQ(true))
	}
	if search != "" {
		q = q.Where(gatewaymodel.ModelIDContainsFold(search))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := q.Order(ent.Asc(gatewaymodel.FieldModelID)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*biz.GatewayModel, 0, len(list))
	for _, item := range list {
		out = append(out, mapGatewayModel(item))
	}
	return out, total, nil
}

func (r *gatewayRepo) UpsertModel(ctx context.Context, model biz.GatewayModel) (*biz.GatewayModel, error) {
	existing, err := r.data.postgres.GatewayModel.Query().
		Where(gatewaymodel.ModelIDEQ(model.ModelID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}

	now := time.Now()
	if existing == nil {
		item, err := r.data.postgres.GatewayModel.Create().
			SetModelID(model.ModelID).
			SetOwnedBy(model.OwnedBy).
			SetCreatedUnix(model.CreatedUnix).
			SetEnabled(model.Enabled).
			SetSource(model.Source).
			SetLastSeenAt(now).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return mapGatewayModel(item), nil
	}

	item, err := r.data.postgres.GatewayModel.UpdateOneID(existing.ID).
		SetOwnedBy(model.OwnedBy).
		SetCreatedUnix(model.CreatedUnix).
		SetEnabled(model.Enabled).
		SetSource(model.Source).
		SetLastSeenAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayModel(item), nil
}

func (r *gatewayRepo) UpsertSyncedModel(ctx context.Context, model biz.GatewayModel) (*biz.GatewayModel, error) {
	existing, err := r.data.postgres.GatewayModel.Query().
		Where(gatewaymodel.ModelIDEQ(model.ModelID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}

	now := time.Now()
	if existing == nil {
		item, err := r.data.postgres.GatewayModel.Create().
			SetModelID(model.ModelID).
			SetOwnedBy(model.OwnedBy).
			SetCreatedUnix(model.CreatedUnix).
			SetEnabled(false).
			SetSource("upstream").
			SetLastSeenAt(now).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return mapGatewayModel(item), nil
	}

	item, err := r.data.postgres.GatewayModel.UpdateOneID(existing.ID).
		SetOwnedBy(model.OwnedBy).
		SetCreatedUnix(model.CreatedUnix).
		SetSource("upstream").
		SetLastSeenAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return mapGatewayModel(item), nil
}

func (r *gatewayRepo) SetModelEnabled(ctx context.Context, id int, enabled bool) error {
	return r.data.postgres.GatewayModel.UpdateOneID(id).
		SetEnabled(enabled).
		Exec(ctx)
}

func (r *gatewayRepo) DeleteModel(ctx context.Context, id int) error {
	model, err := r.data.postgres.GatewayModel.Get(ctx, id)
	if err != nil {
		return err
	}
	modelID := model.ModelID

	if _, err := r.data.postgres.GatewayPolicy.Delete().
		Where(gatewaypolicy.ModelIDEQ(modelID)).
		Exec(ctx); err != nil {
		return err
	}
	if _, err := r.data.postgres.GatewayModelPrice.Delete().
		Where(gatewaymodelprice.ModelIDEQ(modelID)).
		Exec(ctx); err != nil {
		return err
	}
	keys, err := r.data.postgres.GatewayAPIKey.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		models := removeString(normalizeStringList(key.AllowedModels), modelID)
		if len(models) == len(normalizeStringList(key.AllowedModels)) {
			continue
		}
		if err := r.data.postgres.GatewayAPIKey.UpdateOneID(key.ID).
			SetAllowedModels(models).
			Exec(ctx); err != nil {
			return err
		}
	}
	return r.data.postgres.GatewayModel.DeleteOneID(id).Exec(ctx)
}

func (r *gatewayRepo) GetModelByID(ctx context.Context, modelID string) (*biz.GatewayModel, error) {
	item, err := r.data.postgres.GatewayModel.Query().
		Where(gatewaymodel.ModelIDEQ(modelID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return mapGatewayModel(item), nil
}

func (r *gatewayRepo) EnsureDefaultModels(ctx context.Context) error {
	defaults := make([]biz.GatewayModel, 0, len(biz.CodexModelIDs))
	for _, modelID := range biz.CodexModelIDs {
		defaults = append(defaults, biz.GatewayModel{
			ModelID: modelID,
			OwnedBy: "openai",
			Enabled: true,
			Source:  "seed",
		})
	}
	allowedModelIDs := make([]string, 0, len(defaults))
	allowedModelSet := make(map[string]struct{}, len(defaults))
	for _, item := range defaults {
		allowedModelIDs = append(allowedModelIDs, item.ModelID)
		allowedModelSet[item.ModelID] = struct{}{}
		exists, err := r.data.postgres.GatewayModel.Query().
			Where(gatewaymodel.ModelIDEQ(item.ModelID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := r.UpsertModel(ctx, item); err != nil {
			return err
		}
	}
	if err := r.pruneGatewayModels(ctx, allowedModelIDs, allowedModelSet); err != nil {
		return err
	}
	return nil
}

func (r *gatewayRepo) pruneGatewayModels(ctx context.Context, allowedModelIDs []string, allowedModelSet map[string]struct{}) error {
	list, err := r.data.postgres.GatewayModel.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		if _, ok := allowedModelSet[item.ModelID]; ok {
			continue
		}
		if err := r.DeleteModel(ctx, item.ID); err != nil && !ent.IsNotFound(err) {
			return err
		}
	}
	if _, err := r.data.postgres.GatewayModelPrice.Delete().
		Where(gatewaymodelprice.ModelIDNotIn(allowedModelIDs...)).
		Exec(ctx); err != nil {
		return err
	}
	if _, err := r.data.postgres.GatewayPolicy.Delete().
		Where(gatewaypolicy.ModelIDNEQ("*"), gatewaypolicy.ModelIDNotIn(allowedModelIDs...)).
		Exec(ctx); err != nil {
		return err
	}
	keys, err := r.data.postgres.GatewayAPIKey.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		current := normalizeStringList(key.AllowedModels)
		next := filterAllowedStrings(current, allowedModelSet)
		if sameStringList(current, next) {
			continue
		}
		if err := r.data.postgres.GatewayAPIKey.UpdateOneID(key.ID).
			SetAllowedModels(next).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *gatewayRepo) applyUsageFilter(ctx context.Context, q *ent.GatewayUsageLogQuery, filter biz.GatewayUsageFilter) *ent.GatewayUsageLogQuery {
	if filter.KeyID > 0 {
		q = q.Where(gatewayusagelog.APIKeyIDEQ(filter.KeyID))
	}
	if filter.OwnerUserID > 0 {
		ids, err := r.data.postgres.GatewayAPIKey.Query().
			Where(gatewayapikey.OwnerUserIDEQ(filter.OwnerUserID)).
			IDs(ctx)
		if err == nil {
			if len(ids) == 0 {
				q = q.Where(gatewayusagelog.APIKeyIDEQ(-1))
			} else {
				q = q.Where(gatewayusagelog.APIKeyIDIn(ids...))
			}
		}
	}
	if filter.Model != "" {
		q = q.Where(gatewayusagelog.ModelEQ(filter.Model))
	}
	if filter.Endpoint != "" {
		q = q.Where(gatewayusagelog.EndpointEQ(filter.Endpoint))
	}
	if filter.SuccessSet {
		q = q.Where(gatewayusagelog.SuccessEQ(filter.Success))
	}
	if !filter.StartTime.IsZero() {
		q = q.Where(gatewayusagelog.CreatedAtGTE(filter.StartTime))
	}
	if !filter.EndTime.IsZero() {
		q = q.Where(gatewayusagelog.CreatedAtLT(filter.EndTime))
	}
	return q
}

func buildUsageWhereClause(filter biz.GatewayUsageFilter) (string, []any) {
	return buildUsageWhereClauseWithPrefix(filter, "")
}

func buildUsageWhereClauseWithPrefix(filter biz.GatewayUsageFilter, columnPrefix string) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)
	col := func(name string) string {
		return columnPrefix + name
	}

	add := func(format string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(format, len(args)))
	}

	if filter.KeyID > 0 {
		add(col("api_key_id")+" = $%d", filter.KeyID)
	}
	if filter.OwnerUserID > 0 {
		add(col("api_key_id")+" IN (SELECT id FROM gateway_api_keys WHERE owner_user_id = $%d)", filter.OwnerUserID)
	}
	if filter.Model != "" {
		add(col("model")+" = $%d", filter.Model)
	}
	if filter.Endpoint != "" {
		add(col("endpoint")+" = $%d", filter.Endpoint)
	}
	if filter.SuccessSet {
		add(col("success")+" = $%d", filter.Success)
	}
	if !filter.StartTime.IsZero() {
		add(col("created_at")+" >= $%d", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		add(col("created_at")+" < $%d", filter.EndTime)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *gatewayRepo) estimateUsageCost(ctx context.Context, filter biz.GatewayUsageFilter) *float64 {
	prices, err := r.listModelPriceMap(ctx)
	if err != nil || len(prices) == 0 {
		return nil
	}
	where, args := buildUsageWhereClause(filter)
	query := `SELECT
model,
COALESCE(SUM(input_tokens), 0),
COALESCE(SUM(cached_tokens), 0),
COALESCE(SUM(output_tokens), 0)
FROM gateway_usage_logs` + where + `
GROUP BY model`
	rows, err := r.data.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.log.WithContext(ctx).Warnf("close usage cost rows failed err=%v", closeErr)
		}
	}()

	total := 0.0
	seen := false
	for rows.Next() {
		var model string
		var inputTokens, cachedTokens, outputTokens int64
		if err := rows.Scan(&model, &inputTokens, &cachedTokens, &outputTokens); err != nil {
			return nil
		}
		seen = true
		if inputTokens <= 0 && cachedTokens <= 0 && outputTokens <= 0 {
			continue
		}
		price, ok := prices[model]
		if !ok || price == nil {
			return nil
		}
		cost := estimateCostForTokens(inputTokens, cachedTokens, outputTokens, price)
		if cost == nil {
			return nil
		}
		total += *cost
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	if !seen {
		zero := 0.0
		return &zero
	}
	return &total
}

func (r *gatewayRepo) attachUsageKeySummaryCosts(ctx context.Context, filter biz.GatewayUsageFilter, list []*biz.GatewayUsageKeySummary) {
	if len(list) == 0 {
		return
	}
	prices, err := r.listModelPriceMap(ctx)
	if err != nil || len(prices) == 0 {
		return
	}
	where, args := buildUsageWhereClauseWithPrefix(filter, "l.")
	query := `SELECT
COALESCE(l.api_key_id, 0),
l.model,
COALESCE(SUM(l.input_tokens), 0),
COALESCE(SUM(l.cached_tokens), 0),
COALESCE(SUM(l.output_tokens), 0)
FROM gateway_usage_logs l` + where + `
GROUP BY COALESCE(l.api_key_id, 0), l.model`
	rows, err := r.data.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.log.WithContext(ctx).Warnf("close usage key summary cost rows failed err=%v", closeErr)
		}
	}()

	totals := make(map[int]float64, len(list))
	seen := make(map[int]bool, len(list))
	missingPrice := make(map[int]bool, len(list))
	for rows.Next() {
		var apiKeyID int
		var model string
		var inputTokens, cachedTokens, outputTokens int64
		if err := rows.Scan(&apiKeyID, &model, &inputTokens, &cachedTokens, &outputTokens); err != nil {
			return
		}
		seen[apiKeyID] = true
		if inputTokens <= 0 && cachedTokens <= 0 && outputTokens <= 0 {
			continue
		}
		price, ok := prices[model]
		if !ok || price == nil {
			missingPrice[apiKeyID] = true
			continue
		}
		cost := estimateCostForTokens(inputTokens, cachedTokens, outputTokens, price)
		if cost == nil {
			missingPrice[apiKeyID] = true
			continue
		}
		totals[apiKeyID] += *cost
	}
	if err := rows.Err(); err != nil {
		return
	}
	for _, item := range list {
		if item == nil || !seen[item.APIKeyID] || missingPrice[item.APIKeyID] {
			continue
		}
		cost := totals[item.APIKeyID]
		item.EstimatedCostUSD = &cost
	}
}

func (r *gatewayRepo) listModelPriceMap(ctx context.Context) (map[string]*biz.GatewayModelPrice, error) {
	out := biz.OfficialModelPriceMap()
	list, err := r.ListModelPrices(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item != nil {
			out[item.ModelID] = item
		}
	}
	return out, nil
}

func estimateCostForTokens(inputTokens, cachedTokens, outputTokens int64, price *biz.GatewayModelPrice) *float64 {
	if price == nil {
		return nil
	}
	cached := math.Min(float64(maxInt64(cachedTokens, 0)), float64(maxInt64(inputTokens, 0)))
	nonCached := math.Max(float64(inputTokens)-cached, 0)
	output := math.Max(float64(outputTokens), 0)
	cost := (nonCached*price.InputUSDPerMillion + cached*price.CachedInputUSDPerMillion + output*price.OutputUSDPerMillion) / 1_000_000
	return &cost
}

func maxInt64(v, min int64) int64 {
	if v < min {
		return min
	}
	return v
}

func sanitizeAuditMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "authorization") ||
			strings.Contains(lower, "prompt") ||
			strings.Contains(lower, "response") ||
			strings.Contains(lower, "body") {
			continue
		}
		out[key] = value
	}
	return out
}

func mapGatewayAPIKey(item *ent.GatewayAPIKey) *biz.GatewayAPIKey {
	if item == nil {
		return nil
	}
	ownerUserID := 0
	if item.OwnerUserID != nil {
		ownerUserID = *item.OwnerUserID
	}
	quotaWeeklyTokens := item.QuotaWeeklyTokens
	if quotaWeeklyTokens <= 0 && item.QuotaDailyTokens <= 0 && item.QuotaTotalTokens > 0 {
		quotaWeeklyTokens = item.QuotaTotalTokens
	}
	return &biz.GatewayAPIKey{
		ID:                item.ID,
		OwnerUserID:       ownerUserID,
		Name:              item.Name,
		PlainKey:          item.PlainKey,
		KeyPrefix:         item.KeyPrefix,
		KeyLast4:          item.KeyLast4,
		Disabled:          item.Disabled,
		QuotaRequests:     item.QuotaRequests,
		QuotaTotalTokens:  item.QuotaTotalTokens,
		QuotaDailyTokens:  item.QuotaDailyTokens,
		QuotaWeeklyTokens: quotaWeeklyTokens,
		AllowedModels:     normalizeStringList(item.AllowedModels),
		LastUsedAt:        item.LastUsedAt,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func mapGatewayPolicy(item *ent.GatewayPolicy) *biz.GatewayPolicy {
	if item == nil {
		return nil
	}
	return &biz.GatewayPolicy{
		ID:              item.ID,
		APIKeyID:        item.APIKeyID,
		ModelID:         item.ModelID,
		RPM:             item.Rpm,
		TPM:             item.Tpm,
		DailyRequests:   item.DailyRequests,
		MonthlyRequests: item.MonthlyRequests,
		DailyTokens:     item.DailyTokens,
		MonthlyTokens:   item.MonthlyTokens,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func mapGatewayModelPrice(item *ent.GatewayModelPrice) *biz.GatewayModelPrice {
	if item == nil {
		return nil
	}
	return &biz.GatewayModelPrice{
		ID:                       item.ID,
		ModelID:                  item.ModelID,
		InputUSDPerMillion:       item.InputUsdPerMillion,
		CachedInputUSDPerMillion: item.CachedInputUsdPerMillion,
		OutputUSDPerMillion:      item.OutputUsdPerMillion,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
	}
}

func mapGatewayAlertRule(item *ent.GatewayAlertRule) *biz.GatewayAlertRule {
	if item == nil {
		return nil
	}
	return &biz.GatewayAlertRule{
		ID:            item.ID,
		Name:          item.Name,
		Metric:        item.Metric,
		Operator:      item.Operator,
		Threshold:     item.Threshold,
		WindowSeconds: item.WindowSeconds,
		Enabled:       item.Enabled,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func mapGatewayAlertEvent(item *ent.GatewayAlertEvent) *biz.GatewayAlertEvent {
	if item == nil {
		return nil
	}
	ruleID := 0
	if item.RuleID != nil {
		ruleID = *item.RuleID
	}
	ackBy := 0
	if item.AckBy != nil {
		ackBy = *item.AckBy
	}
	return &biz.GatewayAlertEvent{
		ID:        item.ID,
		RuleID:    ruleID,
		RuleName:  item.RuleName,
		Metric:    item.Metric,
		Value:     item.Value,
		Threshold: item.Threshold,
		Status:    item.Status,
		AckBy:     ackBy,
		AckAt:     item.AckAt,
		CreatedAt: item.CreatedAt,
	}
}

func mapGatewayModel(item *ent.GatewayModel) *biz.GatewayModel {
	if item == nil {
		return nil
	}
	return &biz.GatewayModel{
		ID:          item.ID,
		ModelID:     item.ModelID,
		OwnedBy:     item.OwnedBy,
		CreatedUnix: item.CreatedUnix,
		Enabled:     item.Enabled,
		Source:      item.Source,
		LastSeenAt:  item.LastSeenAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapGatewayUsageLog(item *ent.GatewayUsageLog) *biz.GatewayUsageLog {
	if item == nil {
		return nil
	}
	apiKeyID := 0
	if item.APIKeyID != nil {
		apiKeyID = *item.APIKeyID
	}
	return &biz.GatewayUsageLog{
		ID:              item.ID,
		APIKeyID:        apiKeyID,
		APIKeyPrefix:    item.APIKeyPrefix,
		RequestID:       item.RequestID,
		Method:          item.Method,
		Path:            item.Path,
		Endpoint:        item.Endpoint,
		Model:           item.Model,
		StatusCode:      item.StatusCode,
		Success:         item.Success,
		Stream:          item.Stream,
		InputTokens:     item.InputTokens,
		OutputTokens:    item.OutputTokens,
		TotalTokens:     item.TotalTokens,
		CachedTokens:    item.CachedTokens,
		ReasoningTokens: item.ReasoningTokens,
		RequestBytes:    item.RequestBytes,
		ResponseBytes:   item.ResponseBytes,
		DurationMS:      item.DurationMs,
		ErrorType:       item.ErrorType,
		CreatedAt:       item.CreatedAt,
	}
}

func normalizeStringList(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func removeString(in []string, target string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func filterAllowedStrings(in []string, allowed map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := allowed[item]; ok {
			out = append(out, item)
		}
	}
	return out
}

func sameStringList(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
