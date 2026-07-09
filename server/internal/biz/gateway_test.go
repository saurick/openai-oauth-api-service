package biz

import (
	"context"
	"io"
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

func TestNewGatewayAPIKeySecretWithRemarkPrefixesPlainKey(t *testing.T) {
	secret, err := NewGatewayAPIKeySecretWithRemark("team1")
	if err != nil {
		t.Fatalf("NewGatewayAPIKeySecretWithRemark() error = %v", err)
	}
	if !strings.HasPrefix(secret.PlainKey, GatewayAPIKeyPrefix+"team1_") {
		t.Fatalf("plain key = %q, want remark after gateway prefix", secret.PlainKey)
	}

	if _, err := NewGatewayAPIKeySecretWithRemark("team-1"); err != ErrGatewayAPIKeyRemarkInvalid {
		t.Fatalf("NewGatewayAPIKeySecretWithRemark() err = %v, want ErrGatewayAPIKeyRemarkInvalid", err)
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

func TestDetectGatewayClientType(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "explicit codex",
			values: []string{"codex"},
			want:   GatewayClientTypeCodex,
		},
		{
			name:   "explicit opencode wins before user agent",
			values: []string{"opencode", "codex-cli/0.133.0"},
			want:   GatewayClientTypeOpenCode,
		},
		{
			name:   "open code user agent",
			values: []string{"", "OpenCode/1.14.48"},
			want:   GatewayClientTypeOpenCode,
		},
		{
			name:   "codex user agent",
			values: []string{"", "codex-cli/0.133.0"},
			want:   GatewayClientTypeCodex,
		},
		{
			name:   "unknown",
			values: []string{"curl/8.0"},
			want:   GatewayClientTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectGatewayClientType(tt.values...); got != tt.want {
				t.Fatalf("DetectGatewayClientType(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestNormalizeLimitCapsAtPaginationMaximum(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: 30},
		{name: "keeps requested page size", limit: 500, want: 500},
		{name: "caps at maximum page size", limit: 1200, want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLimit(tt.limit); got != tt.want {
				t.Fatalf("normalizeLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
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
	if !strings.HasPrefix(repo.createdInput.Name, "key") {
		t.Fatalf("fallback remark = %q, want key prefix", repo.createdInput.Name)
	}
	if strings.Contains(repo.createdInput.Name, " ") {
		t.Fatalf("fallback remark = %q, want alphanumeric remark", repo.createdInput.Name)
	}
	if !strings.HasPrefix(created.PlainKey, GatewayAPIKeyPrefix+repo.createdInput.Name+"_") {
		t.Fatalf("plain key = %q, want fallback remark prefix %q", created.PlainKey, repo.createdInput.Name)
	}
}

func TestGatewayUsecaseCreateAPIKeyUsesRemarkPrefix(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	created, err := uc.CreateAPIKey(context.Background(), CreateGatewayAPIKeyInput{Name: "client7"})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if repo.createdInput.Name != "client7" {
		t.Fatalf("created name = %q, want client7", repo.createdInput.Name)
	}
	if !strings.HasPrefix(created.PlainKey, GatewayAPIKeyPrefix+"client7_") {
		t.Fatalf("plain key = %q, want remark after gateway prefix", created.PlainKey)
	}
}

func TestGatewayUsecaseCreateAPIKeyRejectsInvalidRemark(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.CreateAPIKey(context.Background(), CreateGatewayAPIKeyInput{Name: "client-7"}); err != ErrGatewayAPIKeyRemarkInvalid {
		t.Fatalf("CreateAPIKey() err = %v, want ErrGatewayAPIKeyRemarkInvalid", err)
	}
}

func TestGatewayUsecaseCreateAPIKeyNormalizesDefaultReasoningEffort(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.CreateAPIKey(context.Background(), CreateGatewayAPIKeyInput{
		Name:                   "client7",
		DefaultReasoningEffort: "LOW",
	}); err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if repo.createdInput.DefaultReasoningEffort != GatewayReasoningEffortLow {
		t.Fatalf("default reasoning = %q, want low", repo.createdInput.DefaultReasoningEffort)
	}
}

func TestGatewayUsecaseCreateAPIKeyRejectsInvalidDefaultReasoningEffort(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.CreateAPIKey(context.Background(), CreateGatewayAPIKeyInput{Name: "client7", DefaultReasoningEffort: "extreme"}); err != ErrGatewayReasoningInvalid {
		t.Fatalf("CreateAPIKey() err = %v, want ErrGatewayReasoningInvalid", err)
	}
}

func TestGatewayUsecaseUpdateAPIKeyRejectsInvalidRemark(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.UpdateAPIKey(context.Background(), UpdateGatewayAPIKeyInput{ID: 1, Name: "client-7"}); err != ErrGatewayAPIKeyRemarkInvalid {
		t.Fatalf("UpdateAPIKey() err = %v, want ErrGatewayAPIKeyRemarkInvalid", err)
	}
}

func TestGatewayUsecaseUpdateAPIKeyDoesNotRewriteSecret(t *testing.T) {
	repo := &gatewayPolicyTestRepo{currentKey: &GatewayAPIKey{ID: 1, Name: "alreadynew", PlainKey: "ogw_old_random_part_with_under_score"}}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	updated, err := uc.UpdateAPIKey(context.Background(), UpdateGatewayAPIKeyInput{ID: 1, Name: "new"})
	if err != nil {
		t.Fatalf("UpdateAPIKey() error = %v", err)
	}
	if updated.Name != "new" {
		t.Fatalf("updated name = %q, want new", updated.Name)
	}
	if repo.resetID != 0 || repo.resetSecret != (GatewayAPIKeySecret{}) {
		t.Fatalf("secret should not be reset on ordinary update: id=%d secret=%+v", repo.resetID, repo.resetSecret)
	}
}

func TestGatewayUsecaseResetAPIKeySecretRewritesOnlySecret(t *testing.T) {
	repo := &gatewayPolicyTestRepo{
		currentKey: &GatewayAPIKey{
			ID:                     7,
			Name:                   "client7",
			Disabled:               true,
			UpstreamStrategy:       GatewayUpstreamStrategyCodexCLI,
			DefaultReasoningEffort: GatewayReasoningEffortHigh,
			QuotaDailyTokens:       12,
			QuotaWeeklyTokens:      34,
			AllowedModels:          []string{"gpt-5.5"},
		},
	}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	created, err := uc.ResetAPIKeySecret(context.Background(), 7)
	if err != nil {
		t.Fatalf("ResetAPIKeySecret() error = %v", err)
	}
	if repo.resetID != 7 {
		t.Fatalf("reset id = %d, want 7", repo.resetID)
	}
	if repo.resetSecret == (GatewayAPIKeySecret{}) {
		t.Fatalf("expected reset secret to be generated")
	}
	if !strings.HasPrefix(created.PlainKey, GatewayAPIKeyPrefix+"client7_") {
		t.Fatalf("plain key = %q, want current remark prefix", created.PlainKey)
	}
	if created.Name != "client7" || !created.Disabled || created.UpstreamStrategy != GatewayUpstreamStrategyCodexCLI || created.DefaultReasoningEffort != GatewayReasoningEffortHigh {
		t.Fatalf("reset should preserve non-secret fields, got %+v", created.GatewayAPIKey)
	}
	if created.QuotaDailyTokens != 12 || created.QuotaWeeklyTokens != 34 || len(created.AllowedModels) != 1 {
		t.Fatalf("reset should preserve policy fields, got %+v", created.GatewayAPIKey)
	}
}

func TestRebuildGatewayAPIKeySecretWithRemarkReplacesFirstSegment(t *testing.T) {
	secret, err := RebuildGatewayAPIKeySecretWithRemark("ogw_123123111_uHuQ6wGpD7bJ", "ggg")
	if err != nil {
		t.Fatalf("RebuildGatewayAPIKeySecretWithRemark() error = %v", err)
	}
	if secret.PlainKey != "ogw_ggg_uHuQ6wGpD7bJ" {
		t.Fatalf("plain key = %q, want first segment replaced", secret.PlainKey)
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

func TestGatewayUsecaseDisableAllAPIKeysDelegatesToRepo(t *testing.T) {
	repo := &gatewayPolicyTestRepo{disableAllCount: 4}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	updated, err := uc.DisableAllAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("DisableAllAPIKeys() error = %v", err)
	}
	if updated != 4 {
		t.Fatalf("updated = %d, want 4", updated)
	}
	if !repo.disableAllCalled {
		t.Fatalf("expected repo DisableAllAPIKeys to be called")
	}
}

func TestGatewayUsecaseEnableAllAPIKeysDelegatesToRepo(t *testing.T) {
	repo := &gatewayPolicyTestRepo{enableAllCount: 3}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)

	updated, err := uc.EnableAllAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("EnableAllAPIKeys() error = %v", err)
	}
	if updated != 3 {
		t.Fatalf("updated = %d, want 3", updated)
	}
	if !repo.enableAllCalled {
		t.Fatalf("expected repo EnableAllAPIKeys to be called")
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

func TestGatewayUsecaseEffectiveCodexUpstreamStrategyUsesAPIKeyOverride(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)
	ctx := context.Background()

	if _, _, _, err := uc.SetCodexUpstreamStrategy(ctx, GatewayUpstreamStrategyBackendOnly); err != nil {
		t.Fatalf("SetCodexUpstreamStrategy() error = %v", err)
	}
	strategy, mode, fallbackEnabled, err := uc.GetEffectiveCodexUpstreamStrategy(ctx, &GatewayAPIKey{
		ID:               9,
		UpstreamStrategy: GatewayUpstreamStrategyBackendWithFallback,
	})
	if err != nil {
		t.Fatalf("GetEffectiveCodexUpstreamStrategy() error = %v", err)
	}
	if strategy != GatewayUpstreamStrategyBackendWithFallback || mode != GatewayUpstreamModeCodexBackend || !fallbackEnabled {
		t.Fatalf("unexpected key override strategy: strategy=%q mode=%q fallback=%v", strategy, mode, fallbackEnabled)
	}

	strategy, mode, fallbackEnabled, err = uc.GetEffectiveCodexUpstreamStrategy(ctx, &GatewayAPIKey{ID: 9})
	if err != nil {
		t.Fatalf("GetEffectiveCodexUpstreamStrategy(inherit) error = %v", err)
	}
	if strategy != GatewayUpstreamStrategyBackendOnly || mode != GatewayUpstreamModeCodexBackend || fallbackEnabled {
		t.Fatalf("unexpected inherited strategy: strategy=%q mode=%q fallback=%v", strategy, mode, fallbackEnabled)
	}
}

func TestGatewayUsecaseEffectiveReasoningEffortUsesKeyGlobalThenRequest(t *testing.T) {
	repo := &gatewayPolicyTestRepo{}
	uc := NewGatewayUsecase(repo, log.NewStdLogger(testWriter{}), nil)
	ctx := context.Background()

	if got, err := uc.GetEffectiveReasoningEffort(ctx, &GatewayAPIKey{DefaultReasoningEffort: GatewayReasoningEffortLow}, GatewayReasoningEffortHigh); err != nil || got != GatewayReasoningEffortLow {
		t.Fatalf("key override effort = %q err=%v, want low", got, err)
	}
	if got, err := uc.GetEffectiveReasoningEffort(ctx, &GatewayAPIKey{DefaultReasoningEffort: GatewayReasoningEffortLow}, ""); err != nil || got != GatewayReasoningEffortLow {
		t.Fatalf("key effort = %q err=%v, want low", got, err)
	}
	if _, err := uc.SetGatewayDefaultReasoningEffort(ctx, GatewayReasoningEffortMed); err != nil {
		t.Fatalf("SetGatewayDefaultReasoningEffort() error = %v", err)
	}
	if got, err := uc.GetEffectiveReasoningEffort(ctx, &GatewayAPIKey{}, GatewayReasoningEffortHigh); err != nil || got != GatewayReasoningEffortMed {
		t.Fatalf("global effort = %q err=%v, want medium", got, err)
	}
	if got, err := uc.GetEffectiveReasoningEffort(ctx, &GatewayAPIKey{DefaultReasoningEffort: GatewayReasoningEffortNone}, GatewayReasoningEffortHigh); err != nil || got != GatewayReasoningEffortHigh {
		t.Fatalf("key none effort = %q err=%v, want requested high", got, err)
	}
	if _, err := uc.SetGatewayDefaultReasoningEffort(ctx, ""); err != nil {
		t.Fatalf("clear SetGatewayDefaultReasoningEffort() error = %v", err)
	}
	if got, err := uc.GetEffectiveReasoningEffort(ctx, &GatewayAPIKey{}, GatewayReasoningEffortHigh); err != nil || got != GatewayReasoningEffortHigh {
		t.Fatalf("request effort = %q err=%v, want high", got, err)
	}
}

func TestGatewayUsecaseSetGatewayDefaultReasoningEffortRejectsInvalidValue(t *testing.T) {
	uc := NewGatewayUsecase(&gatewayPolicyTestRepo{}, log.NewStdLogger(testWriter{}), nil)

	if _, err := uc.SetGatewayDefaultReasoningEffort(context.Background(), "extreme"); err != ErrGatewayReasoningInvalid {
		t.Fatalf("SetGatewayDefaultReasoningEffort() err = %v, want ErrGatewayReasoningInvalid", err)
	}
}

func TestGatewayUsecaseUpdateModelContextPolicyValidation(t *testing.T) {
	uc := NewGatewayUsecase(&gatewayPolicyTestRepo{}, log.NewStdLogger(io.Discard), nil)
	if _, err := uc.UpdateModelContextPolicy(context.Background(), 1, GatewayModelContextPolicy{
		ContextCompactTokens: 300_000,
		ContextHardTokens:    390_000,
		ContextCompactBytes:  1_200_000,
		ContextHardBytes:     1_600_000,
		ContextKeepItems:     8,
	}); err != nil {
		t.Fatalf("UpdateModelContextPolicy(valid) error = %v", err)
	}
	if _, err := uc.UpdateModelContextPolicy(context.Background(), 1, GatewayModelContextPolicy{
		ContextCompactTokens: 390_000,
		ContextHardTokens:    300_000,
	}); err != ErrGatewayModelContextInvalid {
		t.Fatalf("UpdateModelContextPolicy(invalid tokens) error = %v, want ErrGatewayModelContextInvalid", err)
	}
	if _, err := uc.UpdateModelContextPolicy(context.Background(), 1, GatewayModelContextPolicy{
		ContextKeepItems: 1,
	}); err != ErrGatewayModelContextInvalid {
		t.Fatalf("UpdateModelContextPolicy(invalid keep items) error = %v, want ErrGatewayModelContextInvalid", err)
	}
}

func TestEffectiveGatewayModelContextPolicyUsesModelOverride(t *testing.T) {
	fallback := GatewayModelContextPolicy{
		ContextCompactTokens: 180_000,
		ContextHardTokens:    255_000,
		ContextCompactBytes:  850_000,
		ContextHardBytes:     1_050_000,
		ContextKeepItems:     8,
	}
	policy := EffectiveGatewayModelContextPolicy(&GatewayModel{
		ModelID:              "gpt-5.3-codex",
		ContextWindowTokens:  400_000,
		ContextCompactTokens: 300_000,
		ContextHardTokens:    380_000,
		ContextCompactBytes:  1_600_000,
		ContextHardBytes:     2_000_000,
		ContextKeepItems:     10,
	}, fallback)
	if policy.ContextCompactTokens != 300_000 || policy.ContextHardTokens != 380_000 || policy.ContextKeepItems != 10 {
		t.Fatalf("policy did not use model override: %+v", policy)
	}
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

type gatewayPolicyTestRepo struct {
	policy           *GatewayPolicy
	summaryByStart   map[time.Time]*GatewayUsageSummary
	settings         map[string]string
	createdInput     CreateGatewayAPIKeyInput
	updatedInput     UpdateGatewayAPIKeyInput
	currentKey       *GatewayAPIKey
	resetID          int
	resetSecret      GatewayAPIKeySecret
	deletedIDs       []int
	disableAllCount  int
	disableAllCalled bool
	enableAllCount   int
	enableAllCalled  bool
}

func (r *gatewayPolicyTestRepo) CreateAPIKey(_ context.Context, input CreateGatewayAPIKeyInput, secret GatewayAPIKeySecret) (*GatewayAPIKey, error) {
	r.createdInput = input
	return &GatewayAPIKey{
		ID:                     1,
		Name:                   input.Name,
		KeyPrefix:              secret.KeyPrefix,
		KeyLast4:               secret.KeyLast4,
		DefaultReasoningEffort: input.DefaultReasoningEffort,
	}, nil
}
func (r *gatewayPolicyTestRepo) ListAPIKeys(context.Context, int, int, string) ([]*GatewayAPIKey, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) ListAPIKeysByOwner(context.Context, int, int, int) ([]*GatewayAPIKey, int, error) {
	return nil, 0, nil
}
func (r *gatewayPolicyTestRepo) UpdateAPIKey(_ context.Context, input UpdateGatewayAPIKeyInput) (*GatewayAPIKey, error) {
	r.updatedInput = input
	return &GatewayAPIKey{ID: input.ID, Name: input.Name, DefaultReasoningEffort: input.DefaultReasoningEffort}, nil
}
func (r *gatewayPolicyTestRepo) ResetAPIKeySecret(_ context.Context, id int, secret GatewayAPIKeySecret) (*GatewayAPIKey, error) {
	r.resetID = id
	r.resetSecret = secret
	current := &GatewayAPIKey{ID: id, Name: "old"}
	if r.currentKey != nil {
		copy := *r.currentKey
		current = &copy
	}
	current.PlainKey = secret.PlainKey
	current.KeyPrefix = secret.KeyPrefix
	current.KeyLast4 = secret.KeyLast4
	return current, nil
}
func (r *gatewayPolicyTestRepo) GetAPIKeyByID(context.Context, int) (*GatewayAPIKey, error) {
	if r.currentKey != nil {
		return r.currentKey, nil
	}
	return &GatewayAPIKey{ID: 1, Name: "old", PlainKey: "ogw_old_random_part"}, nil
}
func (r *gatewayPolicyTestRepo) DeleteAPIKey(context.Context, int) error { return nil }
func (r *gatewayPolicyTestRepo) DeleteAPIKeys(_ context.Context, ids []int) (int, error) {
	r.deletedIDs = append([]int(nil), ids...)
	return len(ids), nil
}
func (r *gatewayPolicyTestRepo) SetAPIKeyDisabled(context.Context, int, bool) error { return nil }
func (r *gatewayPolicyTestRepo) DisableAllAPIKeys(context.Context) (int, error) {
	r.disableAllCalled = true
	return r.disableAllCount, nil
}
func (r *gatewayPolicyTestRepo) EnableAllAPIKeys(context.Context) (int, error) {
	r.enableAllCalled = true
	return r.enableAllCount, nil
}
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
func (r *gatewayPolicyTestRepo) GetContextSummary(context.Context, string) (*GatewayContextSummary, error) {
	return nil, nil
}
func (r *gatewayPolicyTestRepo) UpsertContextSummary(context.Context, GatewayContextSummary) error {
	return nil
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
func (r *gatewayPolicyTestRepo) UpdateModelContextPolicy(_ context.Context, id int, policy GatewayModelContextPolicy) (*GatewayModel, error) {
	return &GatewayModel{
		ID:                   id,
		ModelID:              DefaultCodexModelID,
		ContextWindowTokens:  policy.ContextWindowTokens,
		ContextCompactTokens: policy.ContextCompactTokens,
		ContextHardTokens:    policy.ContextHardTokens,
		ContextCompactBytes:  policy.ContextCompactBytes,
		ContextHardBytes:     policy.ContextHardBytes,
		ContextKeepItems:     policy.ContextKeepItems,
	}, nil
}
func (r *gatewayPolicyTestRepo) DeleteModel(context.Context, int) error { return nil }
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
