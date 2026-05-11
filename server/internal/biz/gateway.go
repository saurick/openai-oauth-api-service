package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/sdk/trace"
)

const GatewayAPIKeyPrefix = "ogw_"

const (
	GatewayUpstreamModeCodexBackend = "codex_backend"
	GatewayUpstreamModeCodexCLI     = "codex_cli"
	GatewaySettingCodexUpstreamMode = "codex_upstream_mode"
)

var (
	ErrGatewayAPIKeyNotFound      = errors.New("gateway api key not found")
	ErrGatewayAPIKeyDisabled      = errors.New("gateway api key disabled")
	ErrGatewayModelDisabled       = errors.New("gateway model disabled")
	ErrGatewayModelNotAllowed     = errors.New("gateway model not allowed")
	ErrGatewayModelCatalogFixed   = errors.New("gateway model catalog fixed")
	ErrGatewayRateLimited         = errors.New("gateway rate limited")
	ErrGatewayQuotaExceeded       = errors.New("gateway quota exceeded")
	ErrGatewayPolicyInvalid       = errors.New("gateway policy invalid")
	ErrGatewayExportRange         = errors.New("gateway export range too large")
	ErrGatewayUpstreamModeInvalid = errors.New("gateway upstream mode invalid")
)

type GatewayAPIKey struct {
	ID                             int
	OwnerUserID                    int
	Name                           string
	PlainKey                       string
	KeyPrefix                      string
	KeyLast4                       string
	Disabled                       bool
	QuotaRequests                  int64
	QuotaTotalTokens               int64
	QuotaDailyTokens               int64
	QuotaWeeklyTokens              int64
	QuotaDailyInputTokens          int64
	QuotaWeeklyInputTokens         int64
	QuotaDailyOutputTokens         int64
	QuotaWeeklyOutputTokens        int64
	QuotaDailyBillableInputTokens  int64
	QuotaWeeklyBillableInputTokens int64
	AllowedModels                  []string
	LastUsedAt                     *time.Time
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

type CreatedGatewayAPIKey struct {
	GatewayAPIKey
	PlainKey string
}

type GatewayModel struct {
	ID          int
	ModelID     string
	OwnedBy     string
	CreatedUnix int64
	Enabled     bool
	Source      string
	LastSeenAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GatewayUsageLog struct {
	ID                     int
	APIKeyID               int
	APIKeyPrefix           string
	APIKeyName             string
	SessionID              string
	RequestID              string
	Method                 string
	Path                   string
	Endpoint               string
	Model                  string
	StatusCode             int
	Success                bool
	Stream                 bool
	InputTokens            int64
	OutputTokens           int64
	TotalTokens            int64
	CachedTokens           int64
	ReasoningTokens        int64
	RequestBytes           int64
	ResponseBytes          int64
	DurationMS             int64
	UpstreamConfiguredMode string
	UpstreamMode           string
	UpstreamFallback       bool
	UpstreamErrorType      string
	ErrorType              string
	CreatedAt              time.Time
	EstimatedCostUSD       *float64
}

type GatewayUsageFilter struct {
	Limit        int
	Offset       int
	KeyID        int
	OwnerUserID  int
	SessionID    string
	Model        string
	Endpoint     string
	UpstreamMode string
	SuccessSet   bool
	Success      bool
	StartTime    time.Time
	EndTime      time.Time
}

type GatewayUsageSummary struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalTokens       int64
	InputTokens       int64
	OutputTokens      int64
	CachedTokens      int64
	ReasoningTokens   int64
	TotalBytesIn      int64
	TotalBytesOut     int64
	AverageDurationMS int64
	BackendRequests   int64
	CLIRequests       int64
	FallbackRequests  int64
	EstimatedCostUSD  *float64
}

type GatewayUsageBucket struct {
	BucketStart       time.Time
	Model             string
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalTokens       int64
	InputTokens       int64
	OutputTokens      int64
	CachedTokens      int64
	ReasoningTokens   int64
	TotalBytesIn      int64
	TotalBytesOut     int64
	AverageDurationMS int64
	BackendRequests   int64
	CLIRequests       int64
	FallbackRequests  int64
	EstimatedCostUSD  *float64
}

type GatewayUsageKeySummary struct {
	APIKeyID          int
	APIKeyPrefix      string
	APIKeyName        string
	Disabled          bool
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalTokens       int64
	InputTokens       int64
	OutputTokens      int64
	CachedTokens      int64
	ReasoningTokens   int64
	AverageDurationMS int64
	BackendRequests   int64
	CLIRequests       int64
	FallbackRequests  int64
	EstimatedCostUSD  *float64
}

type GatewayUsageSessionSummary struct {
	SessionID         string
	APIKeyID          int
	APIKeyPrefix      string
	APIKeyName        string
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalTokens       int64
	InputTokens       int64
	OutputTokens      int64
	CachedTokens      int64
	ReasoningTokens   int64
	AverageDurationMS int64
	BackendRequests   int64
	CLIRequests       int64
	FallbackRequests  int64
	FirstSeenAt       time.Time
	LastSeenAt        time.Time
	EstimatedCostUSD  *float64
}

type CreateGatewayAPIKeyInput struct {
	Name                           string
	OwnerUserID                    int
	QuotaRequests                  int64
	QuotaTotalTokens               int64
	QuotaDailyTokens               int64
	QuotaWeeklyTokens              int64
	QuotaDailyInputTokens          int64
	QuotaWeeklyInputTokens         int64
	QuotaDailyOutputTokens         int64
	QuotaWeeklyOutputTokens        int64
	QuotaDailyBillableInputTokens  int64
	QuotaWeeklyBillableInputTokens int64
	AllowedModels                  []string
}

type UpdateGatewayAPIKeyInput struct {
	ID                             int
	Name                           string
	OwnerUserID                    int
	QuotaRequests                  int64
	QuotaTotalTokens               int64
	QuotaDailyTokens               int64
	QuotaWeeklyTokens              int64
	QuotaDailyInputTokens          int64
	QuotaWeeklyInputTokens         int64
	QuotaDailyOutputTokens         int64
	QuotaWeeklyOutputTokens        int64
	QuotaDailyBillableInputTokens  int64
	QuotaWeeklyBillableInputTokens int64
	AllowedModels                  []string
	Disabled                       bool
}

type GatewayPolicy struct {
	ID              int
	APIKeyID        int
	ModelID         string
	RPM             int64
	TPM             int64
	DailyRequests   int64
	MonthlyRequests int64
	DailyTokens     int64
	MonthlyTokens   int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type GatewayModelPrice struct {
	ID                       int
	ModelID                  string
	InputUSDPerMillion       float64
	CachedInputUSDPerMillion float64
	OutputUSDPerMillion      float64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type GatewayAuditLog struct {
	ID         int
	ActorID    int
	ActorName  string
	ActorRole  string
	Action     string
	TargetType string
	TargetID   string
	Metadata   map[string]any
	CreatedAt  time.Time
}

type GatewayAlertRule struct {
	ID            int
	Name          string
	Metric        string
	Operator      string
	Threshold     float64
	WindowSeconds int64
	Enabled       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type GatewayAlertEvent struct {
	ID        int
	RuleID    int
	RuleName  string
	Metric    string
	Value     float64
	Threshold float64
	Status    string
	AckBy     int
	AckAt     *time.Time
	CreatedAt time.Time
}

type GatewayAPIKeySecret struct {
	PlainKey  string
	KeyHash   string
	KeyPrefix string
	KeyLast4  string
}

type GatewayRepo interface {
	CreateAPIKey(ctx context.Context, input CreateGatewayAPIKeyInput, secret GatewayAPIKeySecret) (*GatewayAPIKey, error)
	ListAPIKeys(ctx context.Context, limit, offset int, search string) ([]*GatewayAPIKey, int, error)
	ListAPIKeysByOwner(ctx context.Context, ownerUserID, limit, offset int) ([]*GatewayAPIKey, int, error)
	UpdateAPIKey(ctx context.Context, input UpdateGatewayAPIKeyInput) (*GatewayAPIKey, error)
	DeleteAPIKey(ctx context.Context, id int) error
	DeleteAPIKeys(ctx context.Context, ids []int) (int, error)
	SetAPIKeyDisabled(ctx context.Context, id int, disabled bool) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*GatewayAPIKey, error)
	TouchAPIKeyUsed(ctx context.Context, id int, usedAt time.Time) error

	CreateUsageLog(ctx context.Context, item *GatewayUsageLog) error
	ListUsageLogs(ctx context.Context, filter GatewayUsageFilter) ([]*GatewayUsageLog, int, error)
	SummarizeUsage(ctx context.Context, filter GatewayUsageFilter) (*GatewayUsageSummary, error)
	ListUsageBuckets(ctx context.Context, filter GatewayUsageFilter, groupBy string) ([]*GatewayUsageBucket, error)
	ListUsageKeySummaries(ctx context.Context, filter GatewayUsageFilter, limit int) ([]*GatewayUsageKeySummary, error)
	ListUsageSessionSummaries(ctx context.Context, filter GatewayUsageFilter, limit, offset int) ([]*GatewayUsageSessionSummary, int, error)
	GetGatewaySetting(ctx context.Context, key string) (string, error)
	SetGatewaySetting(ctx context.Context, key, value string) error

	GetPolicyForKeyModel(ctx context.Context, apiKeyID int, modelID string) (*GatewayPolicy, error)
	ListPolicies(ctx context.Context, apiKeyID int) ([]*GatewayPolicy, error)
	UpsertPolicy(ctx context.Context, item GatewayPolicy) (*GatewayPolicy, error)
	DeletePolicy(ctx context.Context, id int) error

	ListModelPrices(ctx context.Context) ([]*GatewayModelPrice, error)
	UpsertModelPrice(ctx context.Context, item GatewayModelPrice) (*GatewayModelPrice, error)
	DeleteModelPrice(ctx context.Context, id int) error

	CreateAuditLog(ctx context.Context, item GatewayAuditLog) error
	ListAlertRules(ctx context.Context) ([]*GatewayAlertRule, error)
	UpsertAlertRule(ctx context.Context, item GatewayAlertRule) (*GatewayAlertRule, error)
	DeleteAlertRule(ctx context.Context, id int) error
	SetAlertRuleEnabled(ctx context.Context, id int, enabled bool) error
	ListAlertEvents(ctx context.Context, status string, limit, offset int) ([]*GatewayAlertEvent, int, error)
	AckAlertEvent(ctx context.Context, id, ackBy int, ackAt time.Time) error
	CreateAlertEvent(ctx context.Context, item GatewayAlertEvent) error

	ListModels(ctx context.Context, limit, offset int, enabledOnly bool, search string) ([]*GatewayModel, int, error)
	UpsertModel(ctx context.Context, model GatewayModel) (*GatewayModel, error)
	UpsertSyncedModel(ctx context.Context, model GatewayModel) (*GatewayModel, error)
	SetModelEnabled(ctx context.Context, id int, enabled bool) error
	DeleteModel(ctx context.Context, id int) error
	GetModelByID(ctx context.Context, modelID string) (*GatewayModel, error)
	EnsureDefaultModels(ctx context.Context) error
}

type GatewayUsecase struct {
	repo GatewayRepo
	log  *log.Helper
	tp   *trace.TracerProvider
}

func NewGatewayUsecase(repo GatewayRepo, logger log.Logger, tp *trace.TracerProvider) *GatewayUsecase {
	return &GatewayUsecase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz.gateway")),
		tp:   tp,
	}
}

func NewGatewayAPIKeySecret() (GatewayAPIKeySecret, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return GatewayAPIKeySecret{}, err
	}

	plain := GatewayAPIKeyPrefix + base64.RawURLEncoding.EncodeToString(raw)
	return BuildGatewayAPIKeySecret(plain), nil
}

func BuildGatewayAPIKeySecret(plain string) GatewayAPIKeySecret {
	plain = strings.TrimSpace(plain)
	sum := sha256.Sum256([]byte(plain))
	last4 := plain
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	prefix := plain
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	return GatewayAPIKeySecret{
		PlainKey:  plain,
		KeyHash:   hex.EncodeToString(sum[:]),
		KeyPrefix: prefix,
		KeyLast4:  last4,
	}
}

func NormalizeGatewayBearer(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return auth
}

func DefaultGatewayUpstreamMode() string {
	return GatewayUpstreamModeCodexBackend
}

func NormalizeGatewayUpstreamMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case GatewayUpstreamModeCodexCLI:
		return GatewayUpstreamModeCodexCLI
	case GatewayUpstreamModeCodexBackend:
		return GatewayUpstreamModeCodexBackend
	default:
		return ""
	}
}

func (uc *GatewayUsecase) CreateAPIKey(ctx context.Context, input CreateGatewayAPIKeyInput) (*CreatedGatewayAPIKey, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.QuotaTotalTokens = normalizeNonNegativeInt64(input.QuotaTotalTokens)
	input.QuotaDailyTokens = normalizeNonNegativeInt64(input.QuotaDailyTokens)
	input.QuotaWeeklyTokens = normalizeNonNegativeInt64(input.QuotaWeeklyTokens)
	input.QuotaDailyInputTokens = normalizeNonNegativeInt64(input.QuotaDailyInputTokens)
	input.QuotaWeeklyInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyInputTokens)
	input.QuotaDailyOutputTokens = normalizeNonNegativeInt64(input.QuotaDailyOutputTokens)
	input.QuotaWeeklyOutputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyOutputTokens)
	input.QuotaDailyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaDailyBillableInputTokens)
	input.QuotaWeeklyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyBillableInputTokens)
	allowedModels, err := normalizeAllowedCodexModels(input.AllowedModels)
	if err != nil {
		return nil, err
	}
	input.AllowedModels = allowedModels

	secret, err := NewGatewayAPIKeySecret()
	if err != nil {
		return nil, err
	}
	if input.Name == "" {
		input.Name = "key " + secret.KeyLast4
	}

	item, err := uc.repo.CreateAPIKey(ctx, input, secret)
	if err != nil {
		return nil, err
	}
	return &CreatedGatewayAPIKey{GatewayAPIKey: *item, PlainKey: secret.PlainKey}, nil
}

func (uc *GatewayUsecase) ListAPIKeys(ctx context.Context, limit, offset int, search string) ([]*GatewayAPIKey, int, error) {
	return uc.repo.ListAPIKeys(ctx, normalizeLimit(limit), normalizeOffset(offset), strings.TrimSpace(search))
}

func (uc *GatewayUsecase) ListAPIKeysByOwner(ctx context.Context, ownerUserID, limit, offset int) ([]*GatewayAPIKey, int, error) {
	if ownerUserID <= 0 {
		return []*GatewayAPIKey{}, 0, nil
	}
	return uc.repo.ListAPIKeysByOwner(ctx, ownerUserID, normalizeLimit(limit), normalizeOffset(offset))
}

func (uc *GatewayUsecase) UpdateAPIKey(ctx context.Context, input UpdateGatewayAPIKeyInput) (*GatewayAPIKey, error) {
	if input.ID <= 0 {
		return nil, ErrBadParam
	}
	input.Name = strings.TrimSpace(input.Name)
	input.QuotaTotalTokens = normalizeNonNegativeInt64(input.QuotaTotalTokens)
	input.QuotaDailyTokens = normalizeNonNegativeInt64(input.QuotaDailyTokens)
	input.QuotaWeeklyTokens = normalizeNonNegativeInt64(input.QuotaWeeklyTokens)
	input.QuotaDailyInputTokens = normalizeNonNegativeInt64(input.QuotaDailyInputTokens)
	input.QuotaWeeklyInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyInputTokens)
	input.QuotaDailyOutputTokens = normalizeNonNegativeInt64(input.QuotaDailyOutputTokens)
	input.QuotaWeeklyOutputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyOutputTokens)
	input.QuotaDailyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaDailyBillableInputTokens)
	input.QuotaWeeklyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyBillableInputTokens)
	allowedModels, err := normalizeAllowedCodexModels(input.AllowedModels)
	if err != nil {
		return nil, err
	}
	input.AllowedModels = allowedModels
	if input.Name == "" {
		input.Name = "key " + strconv.Itoa(input.ID)
	}
	return uc.repo.UpdateAPIKey(ctx, input)
}

func (uc *GatewayUsecase) DeleteAPIKey(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.DeleteAPIKey(ctx, id)
}

func (uc *GatewayUsecase) DeleteAPIKeys(ctx context.Context, ids []int) (int, error) {
	seen := map[int]struct{}{}
	cleaned := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleaned = append(cleaned, id)
	}
	if len(cleaned) == 0 {
		return 0, ErrBadParam
	}
	return uc.repo.DeleteAPIKeys(ctx, cleaned)
}

func (uc *GatewayUsecase) SetAPIKeyDisabled(ctx context.Context, id int, disabled bool) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.SetAPIKeyDisabled(ctx, id, disabled)
}

func (uc *GatewayUsecase) AuthenticateAPIKey(ctx context.Context, plain string) (*GatewayAPIKey, error) {
	secret := BuildGatewayAPIKeySecret(plain)
	if secret.PlainKey == "" {
		return nil, ErrGatewayAPIKeyNotFound
	}
	key, err := uc.repo.GetAPIKeyByHash(ctx, secret.KeyHash)
	if err != nil {
		return nil, err
	}
	if key.Disabled {
		return nil, ErrGatewayAPIKeyDisabled
	}
	return key, nil
}

func (uc *GatewayUsecase) ValidateModelAccess(ctx context.Context, key *GatewayAPIKey, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}
	if !IsOfficialCodexModelID(modelID) {
		return ErrGatewayModelDisabled
	}

	model, err := uc.repo.GetModelByID(ctx, modelID)
	if err != nil {
		if errors.Is(err, ErrGatewayModelDisabled) {
			return err
		}
		return err
	}
	if model != nil && !model.Enabled {
		return ErrGatewayModelDisabled
	}

	if key == nil || len(key.AllowedModels) == 0 {
		return nil
	}
	for _, allowed := range key.AllowedModels {
		if strings.TrimSpace(allowed) == modelID {
			return nil
		}
	}
	return ErrGatewayModelNotAllowed
}

func (uc *GatewayUsecase) CheckAPIKeyTokenQuota(ctx context.Context, key *GatewayAPIKey, now time.Time) error {
	if key == nil || key.ID <= 0 {
		return ErrGatewayAPIKeyNotFound
	}
	dailyLimits := gatewayAPIKeyTokenQuotaLimits{
		TotalTokens:         key.QuotaDailyTokens,
		InputTokens:         key.QuotaDailyInputTokens,
		OutputTokens:        key.QuotaDailyOutputTokens,
		BillableInputTokens: key.QuotaDailyBillableInputTokens,
	}
	weeklyLimits := gatewayAPIKeyTokenQuotaLimits{
		TotalTokens:         effectiveAPIKeyWeeklyTokens(key),
		InputTokens:         key.QuotaWeeklyInputTokens,
		OutputTokens:        key.QuotaWeeklyOutputTokens,
		BillableInputTokens: key.QuotaWeeklyBillableInputTokens,
	}
	if !dailyLimits.enabled() && !weeklyLimits.enabled() {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}

	checkTokenWindow := func(limits gatewayAPIKeyTokenQuotaLimits, start time.Time) error {
		if !limits.enabled() {
			return nil
		}
		summary, err := uc.repo.SummarizeUsage(ctx, GatewayUsageFilter{
			KeyID: key.ID, StartTime: start, EndTime: now,
		})
		if err != nil {
			return err
		}
		if limits.exceeded(summary) {
			return ErrGatewayQuotaExceeded
		}
		return nil
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := checkTokenWindow(dailyLimits, dayStart); err != nil {
		return err
	}
	return checkTokenWindow(weeklyLimits, weekStart(dayStart))
}

type gatewayAPIKeyTokenQuotaLimits struct {
	TotalTokens         int64
	InputTokens         int64
	OutputTokens        int64
	BillableInputTokens int64
}

func (limits gatewayAPIKeyTokenQuotaLimits) enabled() bool {
	return limits.TotalTokens > 0 ||
		limits.InputTokens > 0 ||
		limits.OutputTokens > 0 ||
		limits.BillableInputTokens > 0
}

func (limits gatewayAPIKeyTokenQuotaLimits) exceeded(summary *GatewayUsageSummary) bool {
	if summary == nil {
		return false
	}
	if limits.TotalTokens > 0 && summary.TotalTokens >= limits.TotalTokens {
		return true
	}
	if limits.InputTokens > 0 && summary.InputTokens >= limits.InputTokens {
		return true
	}
	if limits.OutputTokens > 0 && summary.OutputTokens >= limits.OutputTokens {
		return true
	}
	if limits.BillableInputTokens > 0 &&
		usageBillableInputTokens(summary.InputTokens, summary.CachedTokens) >= limits.BillableInputTokens {
		return true
	}
	return false
}

func effectiveAPIKeyWeeklyTokens(key *GatewayAPIKey) int64 {
	if key == nil {
		return 0
	}
	if key.QuotaWeeklyTokens > 0 {
		return key.QuotaWeeklyTokens
	}
	if key.QuotaDailyTokens <= 0 && key.QuotaTotalTokens > 0 {
		return key.QuotaTotalTokens
	}
	return 0
}

func weekStart(dayStart time.Time) time.Time {
	weekday := int(dayStart.Weekday())
	daysSinceMonday := (weekday + 6) % 7
	return dayStart.AddDate(0, 0, -daysSinceMonday)
}

func usageBillableInputTokens(inputTokens, cachedTokens int64) int64 {
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

func (uc *GatewayUsecase) CheckPolicy(ctx context.Context, key *GatewayAPIKey, modelID string, now time.Time) error {
	if key == nil || key.ID <= 0 {
		return ErrGatewayAPIKeyNotFound
	}
	if now.IsZero() {
		now = time.Now()
	}
	policy, err := uc.repo.GetPolicyForKeyModel(ctx, key.ID, modelID)
	if err != nil {
		return err
	}
	if policy == nil {
		return nil
	}

	checkWindow := func(limit int64, start time.Time, tokens bool) error {
		if limit <= 0 {
			return nil
		}
		summary, err := uc.repo.SummarizeUsage(ctx, GatewayUsageFilter{
			KeyID: key.ID, Model: strings.TrimSpace(modelID), StartTime: start, EndTime: now,
		})
		if err != nil {
			return err
		}
		if tokens {
			if summary.TotalTokens >= limit {
				return ErrGatewayQuotaExceeded
			}
			return nil
		}
		if summary.TotalRequests >= limit {
			return ErrGatewayQuotaExceeded
		}
		return nil
	}

	if err := checkWindow(policy.RPM, now.Add(-time.Minute), false); err != nil {
		return ErrGatewayRateLimited
	}
	if err := checkWindow(policy.TPM, now.Add(-time.Minute), true); err != nil {
		return ErrGatewayRateLimited
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := checkWindow(policy.DailyRequests, dayStart, false); err != nil {
		return err
	}
	if err := checkWindow(policy.MonthlyRequests, monthStart, false); err != nil {
		return err
	}
	if err := checkWindow(policy.DailyTokens, dayStart, true); err != nil {
		return err
	}
	return checkWindow(policy.MonthlyTokens, monthStart, true)
}

func (uc *GatewayUsecase) TouchAPIKeyUsed(ctx context.Context, id int, usedAt time.Time) error {
	if id <= 0 {
		return nil
	}
	return uc.repo.TouchAPIKeyUsed(ctx, id, usedAt)
}

func (uc *GatewayUsecase) CreateUsageLog(ctx context.Context, item *GatewayUsageLog) error {
	if item == nil {
		return ErrBadParam
	}
	item.UpstreamConfiguredMode = NormalizeGatewayUpstreamMode(item.UpstreamConfiguredMode)
	item.UpstreamMode = NormalizeGatewayUpstreamMode(item.UpstreamMode)
	if err := uc.repo.CreateUsageLog(ctx, item); err != nil {
		return err
	}
	uc.evaluateAlertRules(ctx, item)
	return nil
}

func normalizeUsageFilter(filter GatewayUsageFilter) GatewayUsageFilter {
	filter.SessionID = strings.TrimSpace(filter.SessionID)
	filter.Model = strings.TrimSpace(filter.Model)
	filter.Endpoint = strings.TrimSpace(filter.Endpoint)
	filter.UpstreamMode = NormalizeGatewayUpstreamMode(filter.UpstreamMode)
	return filter
}

func (uc *GatewayUsecase) ListUsageLogs(ctx context.Context, filter GatewayUsageFilter) ([]*GatewayUsageLog, int, error) {
	filter.Limit = normalizeLimit(filter.Limit)
	filter.Offset = normalizeOffset(filter.Offset)
	filter = normalizeUsageFilter(filter)
	return uc.repo.ListUsageLogs(ctx, filter)
}

func (uc *GatewayUsecase) SummarizeUsage(ctx context.Context, filter GatewayUsageFilter) (*GatewayUsageSummary, error) {
	filter = normalizeUsageFilter(filter)
	return uc.repo.SummarizeUsage(ctx, filter)
}

func (uc *GatewayUsecase) ListUsageBuckets(ctx context.Context, filter GatewayUsageFilter, groupBy string) ([]*GatewayUsageBucket, error) {
	filter = normalizeUsageFilter(filter)
	groupBy = strings.TrimSpace(strings.ToLower(groupBy))
	if groupBy == "" {
		groupBy = "day"
	}
	if groupBy != "day" && groupBy != "day_model" {
		return nil, ErrBadParam
	}
	return uc.repo.ListUsageBuckets(ctx, filter, groupBy)
}

func (uc *GatewayUsecase) ListUsageKeySummaries(ctx context.Context, filter GatewayUsageFilter, limit int) ([]*GatewayUsageKeySummary, error) {
	filter = normalizeUsageFilter(filter)
	return uc.repo.ListUsageKeySummaries(ctx, filter, normalizeLimit(limit))
}

func (uc *GatewayUsecase) ListUsageSessionSummaries(ctx context.Context, filter GatewayUsageFilter, limit, offset int) ([]*GatewayUsageSessionSummary, int, error) {
	filter = normalizeUsageFilter(filter)
	return uc.repo.ListUsageSessionSummaries(ctx, filter, normalizeLimit(limit), normalizeOffset(offset))
}

func (uc *GatewayUsecase) GetCodexUpstreamMode(ctx context.Context) (string, error) {
	value, err := uc.repo.GetGatewaySetting(ctx, GatewaySettingCodexUpstreamMode)
	if err != nil {
		return "", err
	}
	mode := NormalizeGatewayUpstreamMode(value)
	if mode == "" {
		return DefaultGatewayUpstreamMode(), nil
	}
	return mode, nil
}

func (uc *GatewayUsecase) SetCodexUpstreamMode(ctx context.Context, mode string) (string, error) {
	normalized := NormalizeGatewayUpstreamMode(mode)
	if normalized == "" {
		return "", ErrGatewayUpstreamModeInvalid
	}
	if err := uc.repo.SetGatewaySetting(ctx, GatewaySettingCodexUpstreamMode, normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func (uc *GatewayUsecase) ListPolicies(ctx context.Context, apiKeyID int) ([]*GatewayPolicy, error) {
	return uc.repo.ListPolicies(ctx, apiKeyID)
}

func (uc *GatewayUsecase) UpsertPolicy(ctx context.Context, item GatewayPolicy) (*GatewayPolicy, error) {
	item.ModelID = strings.TrimSpace(item.ModelID)
	if item.APIKeyID <= 0 || item.ModelID == "" {
		return nil, ErrGatewayPolicyInvalid
	}
	if item.ModelID != "*" && !IsOfficialCodexModelID(item.ModelID) {
		return nil, ErrGatewayPolicyInvalid
	}
	return uc.repo.UpsertPolicy(ctx, item)
}

func (uc *GatewayUsecase) DeletePolicy(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.DeletePolicy(ctx, id)
}

func (uc *GatewayUsecase) ListModelPrices(ctx context.Context) ([]*GatewayModelPrice, error) {
	return uc.repo.ListModelPrices(ctx)
}

func (uc *GatewayUsecase) UpsertModelPrice(ctx context.Context, item GatewayModelPrice) (*GatewayModelPrice, error) {
	item.ModelID = strings.TrimSpace(item.ModelID)
	if item.ModelID == "" {
		return nil, ErrBadParam
	}
	if !IsOfficialCodexModelID(item.ModelID) {
		return nil, ErrBadParam
	}
	return uc.repo.UpsertModelPrice(ctx, item)
}

func (uc *GatewayUsecase) DeleteModelPrice(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.DeleteModelPrice(ctx, id)
}

func (uc *GatewayUsecase) CreateAuditLog(ctx context.Context, item GatewayAuditLog) {
	item.Action = strings.TrimSpace(item.Action)
	if item.Action == "" {
		return
	}
	if err := uc.repo.CreateAuditLog(ctx, item); err != nil {
		uc.log.WithContext(ctx).Warnf("create gateway audit log failed err=%v", err)
	}
}

func (uc *GatewayUsecase) ListAlertRules(ctx context.Context) ([]*GatewayAlertRule, error) {
	return uc.repo.ListAlertRules(ctx)
}

func (uc *GatewayUsecase) UpsertAlertRule(ctx context.Context, item GatewayAlertRule) (*GatewayAlertRule, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.Metric = strings.TrimSpace(item.Metric)
	if item.Name == "" || item.Metric == "" || item.WindowSeconds <= 0 {
		return nil, ErrBadParam
	}
	if item.Operator == "" {
		item.Operator = ">="
	}
	return uc.repo.UpsertAlertRule(ctx, item)
}

func (uc *GatewayUsecase) DeleteAlertRule(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.DeleteAlertRule(ctx, id)
}

func (uc *GatewayUsecase) SetAlertRuleEnabled(ctx context.Context, id int, enabled bool) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.SetAlertRuleEnabled(ctx, id, enabled)
}

func (uc *GatewayUsecase) ListAlertEvents(ctx context.Context, status string, limit, offset int) ([]*GatewayAlertEvent, int, error) {
	return uc.repo.ListAlertEvents(ctx, strings.TrimSpace(status), normalizeLimit(limit), normalizeOffset(offset))
}

func (uc *GatewayUsecase) AckAlertEvent(ctx context.Context, id, ackBy int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.AckAlertEvent(ctx, id, ackBy, time.Now())
}

func (uc *GatewayUsecase) evaluateAlertRules(ctx context.Context, item *GatewayUsageLog) {
	rules, err := uc.repo.ListAlertRules(ctx)
	if err != nil || len(rules) == 0 {
		return
	}
	now := time.Now()
	for _, rule := range rules {
		if rule == nil || !rule.Enabled {
			continue
		}
		start := now.Add(-time.Duration(rule.WindowSeconds) * time.Second)
		summary, err := uc.repo.SummarizeUsage(ctx, GatewayUsageFilter{StartTime: start, EndTime: now})
		if err != nil {
			continue
		}
		value := alertMetricValue(rule.Metric, summary)
		if !compareAlertValue(value, rule.Operator, rule.Threshold) {
			continue
		}
		_ = uc.repo.CreateAlertEvent(ctx, GatewayAlertEvent{
			RuleID: rule.ID, RuleName: rule.Name, Metric: rule.Metric,
			Value: value, Threshold: rule.Threshold, Status: "open",
		})
	}
}

func (uc *GatewayUsecase) ListModels(ctx context.Context, limit, offset int, enabledOnly bool, search string) ([]*GatewayModel, int, error) {
	return uc.repo.ListModels(ctx, normalizeLimit(limit), normalizeOffset(offset), enabledOnly, strings.TrimSpace(search))
}

func (uc *GatewayUsecase) UpsertModel(ctx context.Context, model GatewayModel) (*GatewayModel, error) {
	model.ModelID = strings.TrimSpace(model.ModelID)
	model.OwnedBy = strings.TrimSpace(model.OwnedBy)
	model.Source = strings.TrimSpace(model.Source)
	if model.ModelID == "" {
		return nil, ErrBadParam
	}
	if !IsOfficialCodexModelID(model.ModelID) {
		return nil, ErrBadParam
	}
	if model.Source == "" {
		model.Source = "manual"
	}
	return uc.repo.UpsertModel(ctx, model)
}

func (uc *GatewayUsecase) UpsertSyncedModel(ctx context.Context, model GatewayModel) (*GatewayModel, error) {
	model.ModelID = strings.TrimSpace(model.ModelID)
	model.OwnedBy = strings.TrimSpace(model.OwnedBy)
	if model.ModelID == "" {
		return nil, ErrBadParam
	}
	if !IsOfficialCodexModelID(model.ModelID) {
		return nil, ErrBadParam
	}
	model.Source = "upstream"
	return uc.repo.UpsertSyncedModel(ctx, model)
}

func (uc *GatewayUsecase) SetModelEnabled(ctx context.Context, id int, enabled bool) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.SetModelEnabled(ctx, id, enabled)
}

func (uc *GatewayUsecase) DeleteModel(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrBadParam
	}
	return uc.repo.DeleteModel(ctx, id)
}

func (uc *GatewayUsecase) EnsureDefaultModels(ctx context.Context) error {
	return uc.repo.EnsureDefaultModels(ctx)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 30
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func normalizeNonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func normalizeAllowedCodexModels(in []string) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !IsOfficialCodexModelID(item) {
			return nil, ErrBadParam
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func alertMetricValue(metric string, summary *GatewayUsageSummary) float64 {
	if summary == nil {
		return 0
	}
	switch strings.TrimSpace(metric) {
	case "failed_requests":
		return float64(summary.FailedRequests)
	case "total_tokens":
		return float64(summary.TotalTokens)
	case "average_duration_ms":
		return float64(summary.AverageDurationMS)
	default:
		return 0
	}
}

func compareAlertValue(value float64, op string, threshold float64) bool {
	switch strings.TrimSpace(op) {
	case ">":
		return value > threshold
	case "==":
		return value == threshold
	case "<=":
		return value <= threshold
	case "<":
		return value < threshold
	default:
		return value >= threshold
	}
}
