package data

import (
	"testing"
	"time"

	"server/internal/biz"
)

func TestMapGatewayAPIKeyForRPCIsStructCompatible(t *testing.T) {
	now := time.Unix(100, 0)
	item := &biz.GatewayAPIKey{
		ID:               1,
		Name:             "smoke",
		KeyPrefix:        "ogw_test",
		KeyLast4:         "abcd",
		AllowedModels:    []string{"gpt-5.4", "gpt-5.4-mini"},
		QuotaRequests:    100,
		QuotaTotalTokens: 200,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	data := mapGatewayAPIKeyForRPC(item)
	s := newDataStruct(data)
	if s == nil {
		t.Fatalf("expected gateway api key rpc data to be structpb-compatible")
	}

	models, ok := s.AsMap()["allowed_models"].([]any)
	if !ok {
		t.Fatalf("expected allowed_models to decode as []any, got %T", s.AsMap()["allowed_models"])
	}
	if len(models) != 2 || models[0] != "gpt-5.4" || models[1] != "gpt-5.4-mini" {
		t.Fatalf("unexpected allowed_models: %#v", models)
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
