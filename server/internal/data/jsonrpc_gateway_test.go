package data

import (
	"testing"
	"time"

	"server/internal/biz"
)

func TestMapGatewayAPIKeyForRPCIsStructCompatible(t *testing.T) {
	now := time.Unix(100, 0)
	item := &biz.GatewayAPIKey{
		ID:                1,
		Name:              "smoke",
		PlainKey:          "ogw_plain",
		KeyPrefix:         "ogw_test",
		KeyLast4:          "abcd",
		AllowedModels:     []string{"gpt-5.5", "gpt-5.3-codex"},
		QuotaRequests:     100,
		QuotaTotalTokens:  200,
		QuotaDailyTokens:  300,
		QuotaWeeklyTokens: 900,
		CreatedAt:         now,
		UpdatedAt:         now,
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
}

func TestMapGatewayAPIKeyForRPCCanHidePlainKey(t *testing.T) {
	item := &biz.GatewayAPIKey{ID: 1, Name: "smoke", PlainKey: "ogw_plain"}
	data := mapGatewayAPIKeyForRPC(item, false)
	if _, ok := data["plain_key"]; ok {
		t.Fatalf("plain_key should be hidden when includePlainKey=false")
	}
}

func TestMapGatewayUsageBucketForRPCIsStructCompatible(t *testing.T) {
	start := time.Unix(1778000000, 0)
	item := &biz.GatewayUsageBucket{
		BucketStart:       start,
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
