package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func TestNewGatewayAPIKeySecretStoresOnlyDerivedParts(t *testing.T) {
	secret, err := NewGatewayAPIKeySecret()
	if err != nil {
		t.Fatalf("NewGatewayAPIKeySecret() error = %v", err)
	}
	if !strings.HasPrefix(secret.PlainKey, GatewayAPIKeyPrefix) {
		t.Fatalf("plain key prefix = %q, want %q", secret.PlainKey, GatewayAPIKeyPrefix)
	}
	if len(secret.KeyHash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(secret.KeyHash))
	}
	if secret.KeyPrefix == "" || secret.KeyLast4 == "" {
		t.Fatalf("expected prefix and last4, got %+v", secret)
	}

	rebuilt := BuildGatewayAPIKeySecret(secret.PlainKey)
	if rebuilt.KeyHash != secret.KeyHash {
		t.Fatalf("rebuilt hash mismatch")
	}
	if rebuilt.KeyPrefix != secret.KeyPrefix || rebuilt.KeyLast4 != secret.KeyLast4 {
		t.Fatalf("rebuilt preview mismatch")
	}
}

func TestOfficialModelPriceMapContainsStandardTokenRates(t *testing.T) {
	prices := OfficialModelPriceMap()
	price := prices["gpt-5.5"]
	if price == nil {
		t.Fatalf("expected gpt-5.5 official price")
	}
	if price.InputUSDPerMillion != 5 || price.CachedInputUSDPerMillion != 0.5 || price.OutputUSDPerMillion != 30 {
		t.Fatalf("unexpected gpt-5.5 price: %+v", price)
	}

	price.InputUSDPerMillion = 999
	if OfficialModelPriceMap()["gpt-5.5"].InputUSDPerMillion != 5 {
		t.Fatalf("official price map should return independent copies")
	}
	if !IsOfficialCodexModelID("gpt-5.3-codex-spark") {
		t.Fatalf("expected spark research preview to be allowed")
	}
}

func TestGatewayUsecaseCreateAPIKeyAllowsBlankRemark(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	created, err := uc.CreateAPIKey(context.Background(), CreateGatewayAPIKeyInput{})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if created == nil || created.PlainKey == "" {
		t.Fatalf("expected created key with plain key, got %+v", created)
	}
	if repo.createdInput.Name == "" {
		t.Fatalf("expected fallback remark to be generated")
	}
	if !strings.HasPrefix(repo.createdInput.Name, "key ") {
		t.Fatalf("fallback remark = %q, want key prefix", repo.createdInput.Name)
	}
}

func TestGatewayUsecaseCheckPolicyRateAndQuota(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	repo := &gatewayPolicyTestRepo{
		policy: &GatewayPolicy{
			APIKeyID:      9,
			ModelID:       "*",
			RPM:           2,
			DailyRequests: 10,
		},
		summaryByStart: map[time.Time]*GatewayUsageSummary{
			now.Add(-time.Minute): {TotalRequests: 2},
		},
	}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	err := uc.CheckPolicy(context.Background(), &GatewayAPIKey{ID: 9}, DefaultCodexModelID, now)
	if err != ErrGatewayRateLimited {
		t.Fatalf("CheckPolicy err = %v, want ErrGatewayRateLimited", err)
	}

	repo.summaryByStart[now.Add(-time.Minute)] = &GatewayUsageSummary{TotalRequests: 1}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	repo.summaryByStart[dayStart] = &GatewayUsageSummary{TotalRequests: 10}
	err = uc.CheckPolicy(context.Background(), &GatewayAPIKey{ID: 9}, DefaultCodexModelID, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckPolicy err = %v, want ErrGatewayQuotaExceeded", err)
	}
}

func TestGatewayUsecaseCheckAPIKeyTokenQuota(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := weekStart(dayStart)
	repo := &gatewayPolicyTestRepo{
		summaryByStart: map[time.Time]*GatewayUsageSummary{
			dayStart: {
				TotalTokens:  100,
				InputTokens:  80,
				CachedTokens: 30,
				OutputTokens: 20,
			},
			weekStart: {
				TotalTokens:  500,
				InputTokens:  380,
				CachedTokens: 180,
				OutputTokens: 120,
			},
		},
	}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	err := uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaDailyTokens: 100}, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckAPIKeyTokenQuota err = %v, want ErrGatewayQuotaExceeded", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaDailyTokens: 101}, now)
	if err != nil {
		t.Fatalf("CheckAPIKeyTokenQuota err = %v, want nil", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaWeeklyTokens: 500}, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckAPIKeyTokenQuota weekly err = %v, want ErrGatewayQuotaExceeded", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaWeeklyTokens: 501}, now)
	if err != nil {
		t.Fatalf("CheckAPIKeyTokenQuota weekly err = %v, want nil", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaTotalTokens: 500}, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckAPIKeyTokenQuota legacy total err = %v, want ErrGatewayQuotaExceeded", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaDailyInputTokens: 80}, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckAPIKeyTokenQuota daily input err = %v, want ErrGatewayQuotaExceeded", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaWeeklyOutputTokens: 121}, now)
	if err != nil {
		t.Fatalf("CheckAPIKeyTokenQuota weekly output err = %v, want nil", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9, QuotaWeeklyBillableInputTokens: 200}, now)
	if err != ErrGatewayQuotaExceeded {
		t.Fatalf("CheckAPIKeyTokenQuota weekly billable input err = %v, want ErrGatewayQuotaExceeded", err)
	}

	err = uc.CheckAPIKeyTokenQuota(context.Background(), &GatewayAPIKey{ID: 9}, now)
	if err != nil {
		t.Fatalf("CheckAPIKeyTokenQuota unlimited err = %v, want nil", err)
	}
}

func TestGatewayUsecaseDeleteAPIKeysFiltersInvalidAndDuplicateIDs(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	deleted, err := uc.DeleteAPIKeys(context.Background(), []int{3, 0, -1, 3, 8})
	if err != nil {
		t.Fatalf("DeleteAPIKeys() error = %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}
	want := []int{3, 8}
	if len(repo.deletedIDs) != len(want) {
		t.Fatalf("deletedIDs = %v, want %v", repo.deletedIDs, want)
	}
	for i := range want {
		if repo.deletedIDs[i] != want[i] {
			t.Fatalf("deletedIDs = %v, want %v", repo.deletedIDs, want)
		}
	}
}

func TestGatewayUsecaseDeleteAPIKeysRejectsEmptyIDs(t *testing.T) {
	uc := NewGatewayUsecase(&gatewayPolicyTestRepo{}, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.DeleteAPIKeys(context.Background(), []int{0, -1}); err != ErrBadParam {
		t.Fatalf("DeleteAPIKeys() err = %v, want ErrBadParam", err)
	}
}

func TestGatewayUsecaseRejectsNonCodexModels(t *testing.T) {
	uc := NewGatewayUsecase(&gatewayPolicyTestRepo{}, log.NewStdLogger(testWriter{}), nil)

	if err := uc.ValidateModelAccess(context.Background(), &GatewayAPIKey{}, "gpt-5.5-pro"); err != ErrGatewayModelDisabled {
		t.Fatalf("ValidateModelAccess() err = %v, want ErrGatewayModelDisabled", err)
	}
	if _, err := uc.UpsertModel(context.Background(), GatewayModel{ModelID: "gpt-5.5-pro"}); err != ErrBadParam {
		t.Fatalf("UpsertModel() err = %v, want ErrBadParam", err)
	}
	if err := uc.ValidateModelAccess(context.Background(), &GatewayAPIKey{}, DefaultCodexModelID); err != nil {
		t.Fatalf("ValidateModelAccess() codex err = %v, want nil", err)
	}
}

func TestGatewayUsecaseCodexUpstreamStrategyPersistsModeAndFallback(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)
	ctx := context.Background()

	strategy, mode, fallbackEnabled, err := uc.SetCodexUpstreamStrategy(ctx, GatewayUpstreamStrategyBackendWithFallback)
	if err != nil {
		t.Fatalf("SetCodexUpstreamStrategy() error = %v", err)
	}
	if strategy != GatewayUpstreamStrategyBackendWithFallback || mode != GatewayUpstreamModeCodexBackend || !fallbackEnabled {
		t.Fatalf("unexpected fallback strategy: strategy=%q mode=%q fallback=%v", strategy, mode, fallbackEnabled)
	}

	strategy, mode, fallbackEnabled, err = uc.GetCodexUpstreamStrategy(ctx)
	if err != nil {
		t.Fatalf("GetCodexUpstreamStrategy() error = %v", err)
	}
	if strategy != GatewayUpstreamStrategyBackendWithFallback || mode != GatewayUpstreamModeCodexBackend || !fallbackEnabled {
		t.Fatalf("persisted fallback strategy mismatch: strategy=%q mode=%q fallback=%v", strategy, mode, fallbackEnabled)
	}

	strategy, mode, fallbackEnabled, err = uc.SetCodexUpstreamStrategy(ctx, GatewayUpstreamStrategyCodexCLI)
	if err != nil {
		t.Fatalf("SetCodexUpstreamStrategy(codex_cli) error = %v", err)
	}
	if strategy != GatewayUpstreamStrategyCodexCLI || mode != GatewayUpstreamModeCodexCLI || fallbackEnabled {
		t.Fatalf("unexpected cli strategy: strategy=%q mode=%q fallback=%v", strategy, mode, fallbackEnabled)
	}
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

type gatewayPolicyTestRepo struct {
	policy         *GatewayPolicy
	summaryByStart map[time.Time]*GatewayUsageSummary
	settings       map[string]string
	createdInput   CreateGatewayAPIKeyInput
	deletedIDs     []int
}

func (r *gatewayPolicyTestRepo) CreateAPIKey(_ context.Context, input CreateGatewayAPIKeyInput, secret GatewayAPIKeySecret) (*GatewayAPIKey, error) {
	r.createdInput = input
	return &GatewayAPIKey{
		ID:        1,
		Name:      input.Name,
		KeyPrefix: secret.KeyPrefix,
		KeyLast4:  secret.KeyLast4,
	}, nil
}
func (r *gatewayPolicyTestRepo) ListAPIKeys(context.Context, int, int, string) ([]*GatewayAPIKey, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) ListAPIKeysByOwner(context.Context, int, int, int) ([]*GatewayAPIKey, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) UpdateAPIKey(context.Context, UpdateGatewayAPIKeyInput) (*GatewayAPIKey, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) DeleteAPIKey(context.Context, int) error { return nil }
func (r *gatewayPolicyTestRepo) DeleteAPIKeys(_ context.Context, ids []int) (int, error) {
	r.deletedIDs = append([]int(nil), ids...)
	return len(ids), nil
}
func (r *gatewayPolicyTestRepo) SetAPIKeyDisabled(context.Context, int, bool) error { return nil }
func (r *gatewayPolicyTestRepo) GetAPIKeyByHash(context.Context, string) (*GatewayAPIKey, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) TouchAPIKeyUsed(context.Context, int, time.Time) error {
	return nil
}
func (r *gatewayPolicyTestRepo) CreateUsageLog(context.Context, *GatewayUsageLog) error { return nil }
func (r *gatewayPolicyTestRepo) ListUsageLogs(context.Context, GatewayUsageFilter) ([]*GatewayUsageLog, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) SummarizeUsage(_ context.Context, filter GatewayUsageFilter) (*GatewayUsageSummary, error) {
	if item := r.summaryByStart[filter.StartTime]; item != nil {
		return item, nil
	}
	return &GatewayUsageSummary{}, nil
}
func (r *gatewayPolicyTestRepo) ListUsageBuckets(context.Context, GatewayUsageFilter, string) ([]*GatewayUsageBucket, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) ListUsageKeySummaries(context.Context, GatewayUsageFilter, int) ([]*GatewayUsageKeySummary, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) ListUsageSessionSummaries(context.Context, GatewayUsageFilter, int, int) ([]*GatewayUsageSessionSummary, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) GetGatewaySetting(_ context.Context, key string) (string, error) {
	if r.settings == nil {
		return "", nil
	}
	return r.settings[key], nil
}
func (r *gatewayPolicyTestRepo) SetGatewaySetting(_ context.Context, key string, value string) error {
	if r.settings == nil {
		r.settings = map[string]string{}
	}
	r.settings[key] = value
	return nil
}
func (r *gatewayPolicyTestRepo) GetPolicyForKeyModel(context.Context, int, string) (*GatewayPolicy, error) {
	return r.policy, nil
}
func (r *gatewayPolicyTestRepo) ListPolicies(context.Context, int) ([]*GatewayPolicy, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) UpsertPolicy(context.Context, GatewayPolicy) (*GatewayPolicy, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) DeletePolicy(context.Context, int) error { return nil }
func (r *gatewayPolicyTestRepo) ListModelPrices(context.Context) ([]*GatewayModelPrice, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) UpsertModelPrice(context.Context, GatewayModelPrice) (*GatewayModelPrice, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) DeleteModelPrice(context.Context, int) error { return nil }
func (r *gatewayPolicyTestRepo) CreateAuditLog(context.Context, GatewayAuditLog) error {
	return nil
}
func (r *gatewayPolicyTestRepo) ListAlertRules(context.Context) ([]*GatewayAlertRule, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) UpsertAlertRule(context.Context, GatewayAlertRule) (*GatewayAlertRule, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) DeleteAlertRule(context.Context, int) error { return nil }
func (r *gatewayPolicyTestRepo) SetAlertRuleEnabled(context.Context, int, bool) error {
	return nil
}
func (r *gatewayPolicyTestRepo) ListAlertEvents(context.Context, string, int, int) ([]*GatewayAlertEvent, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) AckAlertEvent(context.Context, int, int, time.Time) error {
	return nil
}
func (r *gatewayPolicyTestRepo) CreateAlertEvent(context.Context, GatewayAlertEvent) error {
	return nil
}
func (r *gatewayPolicyTestRepo) ListModels(context.Context, int, int, bool, string) ([]*GatewayModel, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) UpsertModel(context.Context, GatewayModel) (*GatewayModel, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) UpsertSyncedModel(context.Context, GatewayModel) (*GatewayModel, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) SetModelEnabled(context.Context, int, bool) error { return nil }
func (r *gatewayPolicyTestRepo) DeleteModel(context.Context, int) error           { return nil }
func (r *gatewayPolicyTestRepo) GetModelByID(context.Context, string) (*GatewayModel, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) EnsureDefaultModels(context.Context) error { return nil }

func TestNormalizeGatewayBearer(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "bearer", in: "Bearer ogw_abc", want: "ogw_abc"},
		{name: "lowercase", in: "bearer ogw_def", want: "ogw_def"},
		{name: "plain", in: "ogw_plain", want: "ogw_plain"},
		{name: "blank", in: "  ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeGatewayBearer(tt.in); got != tt.want {
				t.Fatalf("NormalizeGatewayBearer(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
