package data

import (
	"testing"
	"time"

	"server/internal/biz"
)

func TestMapGatewayAPIKeyForRPCIsStructCompatible(t *testing.T) {
	now := time.Unix(100, 0)
	item := &biz.GatewayAPIKey{
		ID:                             1,
		Name:                           "smoke",
		PlainKey:                       "ogw_plain",
		KeyPrefix:                      "ogw_test",
		KeyLast4:                       "abcd",
		AllowedModels:                  []string{"gpt-5.5", "gpt-5.3-codex"},
		QuotaRequests:                  100,
		QuotaTotalTokens:               200,
		QuotaDailyTokens:               300,
		QuotaWeeklyTokens:              900,
		QuotaDailyInputTokens:          301,
		QuotaWeeklyInputTokens:         901,
		QuotaDailyOutputTokens:         302,
		QuotaWeeklyOutputTokens:        902,
		QuotaDailyBillableInputTokens:  303,
		QuotaWeeklyBillableInputTokens: 903,
		CreatedAt:                      now,
		UpdatedAt:                      now,
	}

	data := mapGatewayAPIKeyForRPC(item, true)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway api key rpc data to be structpb-compatible")
	}

	models, ok := s.AsMap()["allowed_models"].([]any)
	if !ok {
		t.Fatalf("expected allowed_models to decode as []any, got %T", s.AsMap()["allowed_models"])
	}
	if len(models) != 2 || models[0] != "gpt-5.5" || models[1] != "gpt-5.3-codex" {
		t.Fatalf("unexpected allowed_models: %#v", models)
	}
	if s.AsMap()["plain_key"] != "ogw_plain" {
		t.Fatalf("plain_key = %#v, want ogw_plain", s.AsMap()["plain_key"])
	}
	if s.AsMap()["quota_daily_tokens"] != float64(300) ||
		s.AsMap()["quota_weekly_tokens"] != float64(900) {
		t.Fatalf("unexpected quota token windows: %#v", s.AsMap())
	}
	if s.AsMap()["quota_daily_input_tokens"] != float64(301) ||
		s.AsMap()["quota_weekly_input_tokens"] != float64(901) ||
		s.AsMap()["quota_daily_output_tokens"] != float64(302) ||
		s.AsMap()["quota_weekly_output_tokens"] != float64(902) ||
		s.AsMap()["quota_daily_billable_input_tokens"] != float64(303) ||
		s.AsMap()["quota_weekly_billable_input_tokens"] != float64(903) {
		t.Fatalf("unexpected detailed quota token windows: %#v", s.AsMap())
	}
}

func TestMapGatewayAPIKeyForRPCCanHidePlainKey(t *testing.T) {
	item := &biz.GatewayAPIKey{ID: 1, Name: "smoke", PlainKey: "ogw_plain"}
	data := mapGatewayAPIKeyForRPC(item, false)
	if _, ok := data["plain_key"]; ok {
		t.Fatalf("plain_key should be hidden when includePlainKey=false")
	}
}

func TestGatewayUsageFilterFromParamsSupportsKeyIDs(t *testing.T) {
	filter := gatewayUsageFilterFromParams(map[string]any{
		"key_id":  1,
		"key_ids": []any{2, float64(3), "4", 0, "bad"},
	})

	if filter.KeyID != 1 {
		t.Fatalf("KeyID = %d, want 1", filter.KeyID)
	}
	if len(filter.KeyIDs) != 3 ||
		filter.KeyIDs[0] != 2 ||
		filter.KeyIDs[1] != 3 ||
		filter.KeyIDs[2] != 4 {
		t.Fatalf("KeyIDs = %#v, want [2 3 4]", filter.KeyIDs)
	}
}

func TestMapGatewayUsageBucketForRPCIsStructCompatible(t *testing.T) {
	start := time.Unix(1778000000, 0)
	item := &biz.GatewayUsageBucket{
		BucketStart:       start,
		Model:             "gpt-5.4",
		TotalRequests:     12,
		SuccessRequests:   10,
		FailedRequests:    2,
		TotalTokens:       300,
		InputTokens:       180,
		OutputTokens:      90,
		CachedTokens:      40,
		ReasoningTokens:   20,
		TotalBytesIn:      1024,
		TotalBytesOut:     2048,
		AverageDurationMS: 818,
	}

	data := mapGatewayUsageBucketForRPC(item)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway usage bucket rpc data to be structpb-compatible")
	}

	got := s.AsMap()
	if got["bucket_start"] != float64(start.Unix()) {
		t.Fatalf("bucket_start = %#v, want %d", got["bucket_start"], start.Unix())
	}
	if got["model"] != "gpt-5.4" {
		t.Fatalf("model = %#v, want gpt-5.4", got["model"])
	}
	if got["cached_tokens"] != float64(40) || got["reasoning_tokens"] != float64(20) {
		t.Fatalf("unexpected token fields: %#v", got)
	}
}

func TestMapGatewayUsageKeySummaryForRPCIsStructCompatible(t *testing.T) {
	cost := 0.42
	item := &biz.GatewayUsageKeySummary{
		APIKeyID:          3,
		APIKeyPrefix:      "ogw_test",
		APIKeyName:        "production",
		TotalRequests:     7,
		SuccessRequests:   6,
		FailedRequests:    1,
		TotalTokens:       900,
		InputTokens:       600,
		OutputTokens:      250,
		CachedTokens:      120,
		ReasoningTokens:   50,
		AverageDurationMS: 321,
		EstimatedCostUSD:  &cost,
	}

	data := mapGatewayUsageKeySummaryForRPC(item)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway usage key summary rpc data to be structpb-compatible")
	}

	got := s.AsMap()
	if got["api_key_id"] != float64(3) || got["api_key_name"] != "production" {
		t.Fatalf("unexpected key fields: %#v", got)
	}
	if got["estimated_cost_usd"] != cost {
		t.Fatalf("estimated_cost_usd = %#v, want %v", got["estimated_cost_usd"], cost)
	}
}

func TestMapGatewayUsageForRPCIncludesKeyName(t *testing.T) {
	item := &biz.GatewayUsageLog{
		ID:              10,
		APIKeyID:        3,
		APIKeyPrefix:    "ogw_test",
		APIKeyName:      "production",
		ReasoningEffort: "high",
		CreatedAt:       time.Unix(1778000000, 0),
		InputTokens:     100,
		CachedTokens:    40,
		OutputTokens:    20,
		TotalTokens:     120,
	}

	data := mapGatewayUsageForRPC(item)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway usage rpc data to be structpb-compatible")
	}

	got := s.AsMap()
	if got["api_key_id"] != float64(3) ||
		got["api_key_prefix"] != "ogw_test" ||
		got["api_key_name"] != "production" ||
		got["reasoning_effort"] != "high" {
		t.Fatalf("unexpected key fields: %#v", got)
	}
}

func TestMapGatewayUsageSessionSummaryForRPCIsStructCompatible(t *testing.T) {
	cost := 1.23
	first := time.Unix(1778000000, 0)
	last := first.Add(5 * time.Minute)
	item := &biz.GatewayUsageSessionSummary{
		SessionID:         "session-123",
		APIKeyID:          3,
		APIKeyPrefix:      "ogw_test",
		APIKeyName:        "production",
		TotalRequests:     4,
		SuccessRequests:   3,
		FailedRequests:    1,
		TotalTokens:       1200,
		InputTokens:       800,
		OutputTokens:      300,
		CachedTokens:      100,
		ReasoningTokens:   50,
		AverageDurationMS: 456,
		FirstSeenAt:       first,
		LastSeenAt:        last,
		EstimatedCostUSD:  &cost,
	}

	data := mapGatewayUsageSessionSummaryForRPC(item)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway usage session summary rpc data to be structpb-compatible")
	}

	got := s.AsMap()
	if got["session_id"] != "session-123" ||
		got["api_key_id"] != float64(3) ||
		got["api_key_name"] != "production" {
		t.Fatalf("unexpected session fields: %#v", got)
	}
	if got["first_seen_at"] != float64(first.Unix()) ||
		got["last_seen_at"] != float64(last.Unix()) {
		t.Fatalf("unexpected time fields: %#v", got)
	}
}

func TestEstimateCostForTokensUsesPriceSource(t *testing.T) {
	cost := estimateCostForTokens(1000, 400, 200, &biz.GatewayModelPrice{
		InputUSDPerMillion:       5,
		CachedInputUSDPerMillion: 0.5,
		OutputUSDPerMillion:      30,
	})
	if cost == nil {
		t.Fatalf("expected cost")
	}
	want := (600*5 + 400*0.5 + 200*30) / 1_000_000.0
	if *cost != want {
		t.Fatalf("cost = %v, want %v", *cost, want)
	}
}
