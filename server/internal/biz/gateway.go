package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
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
	GatewaySettingCodexFallback     = "codex_upstream_fallback_enabled"
	GatewaySettingDefaultReasoning  = "gateway_default_reasoning_effort"

	GatewayUpstreamStrategyBackendOnly         = "backend_only"
	GatewayUpstreamStrategyBackendWithFallback = "backend_with_cli_fallback"
	GatewayUpstreamStrategyCodexCLI            = "codex_cli"

	GatewayReasoningEffortNone  = "none"
	GatewayReasoningEffortLow   = "low"
	GatewayReasoningEffortMed   = "medium"
	GatewayReasoningEffortHigh  = "high"
	GatewayReasoningEffortXHigh = "xhigh"
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
	ErrGatewayAPIKeyRemarkInvalid = errors.New("gateway api key remark invalid")
	ErrGatewayModelContextInvalid = errors.New("gateway model context invalid")
	ErrGatewayReasoningInvalid    = errors.New("gateway reasoning effort invalid")
)

type GatewayAPIKey struct {
	ID                             int
	OwnerUserID                    int
	Name                           string
	PlainKey                       string
	KeyPrefix                      string
	KeyLast4                       string
	Disabled                       bool
	UpstreamStrategy               string
	DefaultReasoningEffort         string
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
	ID                   int
	ModelID              string
	OwnedBy              string
	CreatedUnix          int64
	Enabled              bool
	Source               string
	ContextWindowTokens  int64
	ContextCompactTokens int64
	ContextHardTokens    int64
	ContextCompactBytes  int64
	ContextHardBytes     int64
	ContextKeepItems     int
	LastSeenAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type GatewayModelContextPolicy struct {
	ContextWindowTokens  int64
	ContextCompactTokens int64
	ContextHardTokens    int64
	ContextCompactBytes  int64
	ContextHardBytes     int64
	ContextKeepItems     int
}

type GatewayUsageDiagnostic struct {
	RequestBytes                    int64  `json:"request_bytes"`
	ResponseBytes                   int64  `json:"response_bytes"`
	BackendOnly                     bool   `json:"backend_only"`
	FallbackEnabled                 bool   `json:"fallback_enabled"`
	FallbackBlocked                 bool   `json:"fallback_blocked"`
	ReasoningEffort                 string `json:"reasoning_effort"`
	UpstreamHTTPStatus              int    `json:"upstream_http_status"`
	UpstreamBody                    string `json:"upstream_body"`
	UpstreamStreamStarted           bool   `json:"upstream_stream_started"`
	UpstreamStreamCompleted         bool   `json:"upstream_stream_completed"`
	UpstreamStreamDoneSeen          bool   `json:"upstream_stream_done_seen"`
	UpstreamStreamEvents            int    `json:"upstream_stream_events"`
	ContextCompacted                bool   `json:"context_compacted"`
	ContextCompactionReason         string `json:"context_compaction_reason"`
	ContextCompactionSummary        string `json:"context_compaction_summary"`
	ContextCompactionCount          int    `json:"context_compaction_count"`
	ContextOriginalBytes            int64  `json:"context_original_bytes"`
	ContextCompactedBytes           int64  `json:"context_compacted_bytes"`
	ContextOriginalEstimatedTokens  int64  `json:"context_original_estimated_tokens"`
	ContextCompactedEstimatedTokens int64  `json:"context_compacted_estimated_tokens"`
	ContextWindowTokens             int64  `json:"context_window_tokens"`
	ContextCompactTokenLimit        int64  `json:"context_compact_token_limit"`
	ContextHardTokenLimit           int64  `json:"context_hard_token_limit"`
	ContextCompactByteLimit         int64  `json:"context_compact_byte_limit"`
	ContextHardByteLimit            int64  `json:"context_hard_byte_limit"`
	ContextKeepItems                int    `json:"context_keep_items"`
}

func (d GatewayUsageDiagnostic) Summary() string {
	parts := make([]string, 0, 6)
	if d.RequestBytes > 0 {
		parts = append(parts, fmt.Sprintf("request=%dB", d.RequestBytes))
	}
	if d.ResponseBytes > 0 {
		parts = append(parts, fmt.Sprintf("response=%dB", d.ResponseBytes))
	}
	if d.BackendOnly {
		parts = append(parts, "backend-only")
	}
	if d.FallbackBlocked {
		parts = append(parts, "fallback-blocked")
	} else if d.FallbackEnabled {
		parts = append(parts, "fallback-enabled")
	}
	if d.ReasoningEffort != "" {
		parts = append(parts, "effort="+d.ReasoningEffort)
	}
	if d.UpstreamHTTPStatus > 0 {
		parts = append(parts, fmt.Sprintf("upstream_http=%d", d.UpstreamHTTPStatus))
	}
	if d.UpstreamBody != "" {
		parts = append(parts, "upstream_body="+d.UpstreamBody)
	}
	if d.UpstreamStreamStarted {
		if d.UpstreamStreamCompleted {
			parts = append(parts, "upstream-stream-completed")
		} else {
			parts = append(parts, "upstream-stream-started")
		}
		if d.UpstreamStreamDoneSeen {
			parts = append(parts, "upstream-done-seen")
		}
		if d.UpstreamStreamEvents > 0 {
			parts = append(parts, fmt.Sprintf("upstream_events=%d", d.UpstreamStreamEvents))
		}
	}
	if d.ContextCompacted {
		parts = append(parts, "context-compacted")
		if d.ContextCompactionCount > 0 {
			parts = append(parts, fmt.Sprintf("compact_count=%d", d.ContextCompactionCount))
		}
		if d.ContextOriginalBytes > 0 || d.ContextCompactedBytes > 0 {
			parts = append(parts, fmt.Sprintf("context_bytes=%d->%d", d.ContextOriginalBytes, d.ContextCompactedBytes))
		}
		if d.ContextOriginalEstimatedTokens > 0 || d.ContextCompactedEstimatedTokens > 0 {
			parts = append(parts, fmt.Sprintf("context_tokens=%d->%d", d.ContextOriginalEstimatedTokens, d.ContextCompactedEstimatedTokens))
		}
		if d.ContextCompactionReason != "" {
			parts = append(parts, "compact_reason="+d.ContextCompactionReason)
		}
		if d.ContextCompactTokenLimit > 0 || d.ContextHardTokenLimit > 0 {
			parts = append(parts, fmt.Sprintf("context_token_limits=%d/%d", d.ContextCompactTokenLimit, d.ContextHardTokenLimit))
		}
	}
	return strings.Join(parts, ", ")
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
	ReasoningEffort        string
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
	Diagnostic             GatewayUsageDiagnostic
	CreatedAt              time.Time
	EstimatedCostUSD       *float64
}

type GatewayUsageFilter struct {
	Limit             int
	Offset            int
	KeyID             int
	KeyIDs            []int
	OwnerUserID       int
	SessionID         string
	Model             string
	ReasoningEffort   string
	Endpoint          string
	UpstreamMode      string
	UpstreamErrorType string
	ErrorType         string
	ExcludeErrorType  string
	StatusCode        int
	SuccessSet        bool
	Success           bool
	StartTime         time.Time
	EndTime           time.Time
}

type GatewayUsageSummary struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	ClientCanceled    int64
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
	ClientCanceled    int64
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
	ClientCanceled    int64
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
	SessionID              string
	APIKeyID               int
	APIKeyPrefix           string
	APIKeyName             string
	TotalRequests          int64
	SuccessRequests        int64
	FailedRequests         int64
	ClientCanceled         int64
	TotalTokens            int64
	InputTokens            int64
	OutputTokens           int64
	CachedTokens           int64
	ReasoningTokens        int64
	AverageDurationMS      int64
	BackendRequests        int64
	CLIRequests            int64
	FallbackRequests       int64
	FirstSeenAt            time.Time
	LastSeenAt             time.Time
	EstimatedCostUSD       *float64
	ContextCompactionCount int
	ContextSummary         string
	ContextOriginalBytes   int64
	ContextCompactedBytes  int64
	ContextOriginalTokens  int64
	ContextCompactedTokens int64
	ContextCompactedAt     *time.Time
}

type GatewayContextSummary struct {
	SessionID           string
	APIKeyID            int
	APIKeyPrefix        string
	Summary             string
	SummaryTokens       int64
	CompactionCount     int
	LastRequestID       string
	LastReason          string
	LastOriginalBytes   int64
	LastCompactedBytes  int64
	LastOriginalTokens  int64
	LastCompactedTokens int64
	LastError           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
	UpstreamStrategy               string
	DefaultReasoningEffort         string
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
	UpstreamStrategy               string
	DefaultReasoningEffort         string
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
	GetAPIKeyByID(ctx context.Context, id int) (*GatewayAPIKey, error)
	UpdateAPIKey(ctx context.Context, input UpdateGatewayAPIKeyInput) (*GatewayAPIKey, error)
	ResetAPIKeySecret(ctx context.Context, id int, secret GatewayAPIKeySecret) (*GatewayAPIKey, error)
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
	GetContextSummary(ctx context.Context, sessionID string) (*GatewayContextSummary, error)
	UpsertContextSummary(ctx context.Context, item GatewayContextSummary) error
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
	UpdateModelContextPolicy(ctx context.Context, id int, policy GatewayModelContextPolicy) (*GatewayModel, error)
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
	return NewGatewayAPIKeySecretWithRemark("")
}

func NewGatewayAPIKeySecretWithRemark(remark string) (GatewayAPIKeySecret, error) {
	remark, err := NormalizeGatewayAPIKeyRemark(remark)
	if err != nil {
		return GatewayAPIKeySecret{}, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return GatewayAPIKeySecret{}, err
	}

	randomPart := base64.RawURLEncoding.EncodeToString(raw)
	plain := GatewayAPIKeyPrefix + randomPart
	if remark != "" {
		plain = GatewayAPIKeyPrefix + remark + "_" + randomPart
	}
	return BuildGatewayAPIKeySecret(plain), nil
}

func NormalizeGatewayAPIKeyRemark(remark string) (string, error) {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return "", nil
	}
	if len(remark) > 80 {
		return "", ErrGatewayAPIKeyRemarkInvalid
	}
	for _, r := range remark {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return "", ErrGatewayAPIKeyRemarkInvalid
	}
	return remark, nil
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

func RebuildGatewayAPIKeySecretWithRemark(plain string, nextRemark string) (GatewayAPIKeySecret, error) {
	nextRemark, err := NormalizeGatewayAPIKeyRemark(nextRemark)
	if err != nil {
		return GatewayAPIKeySecret{}, err
	}
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return NewGatewayAPIKeySecretWithRemark(nextRemark)
	}
	if !strings.HasPrefix(plain, GatewayAPIKeyPrefix) {
		return GatewayAPIKeySecret{}, ErrBadParam
	}

	suffix := strings.TrimPrefix(plain, GatewayAPIKeyPrefix)
	if idx := strings.Index(suffix, "_"); idx >= 0 && idx+1 < len(suffix) {
		suffix = suffix[idx+1:]
	}
	if suffix == "" {
		return GatewayAPIKeySecret{}, ErrBadParam
	}
	if nextRemark == "" {
		return BuildGatewayAPIKeySecret(GatewayAPIKeyPrefix + suffix), nil
	}
	return BuildGatewayAPIKeySecret(GatewayAPIKeyPrefix + nextRemark + "_" + suffix), nil
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
	if mode := NormalizeGatewayUpstreamMode(os.Getenv("CODEX_UPSTREAM_MODE")); mode != "" {
		return mode
	}
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

func NormalizeGatewayUpstreamStrategy(strategy string) string {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case GatewayUpstreamStrategyBackendOnly:
		return GatewayUpstreamStrategyBackendOnly
	case GatewayUpstreamStrategyBackendWithFallback:
		return GatewayUpstreamStrategyBackendWithFallback
	case GatewayUpstreamStrategyCodexCLI:
		return GatewayUpstreamStrategyCodexCLI
	default:
		return ""
	}
}

func NormalizeGatewayAPIKeyUpstreamStrategy(strategy string) string {
	if strings.TrimSpace(strategy) == "" {
		return ""
	}
	return NormalizeGatewayUpstreamStrategy(strategy)
}

func NormalizeGatewayReasoningEffort(effort string) string {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case GatewayReasoningEffortLow:
		return GatewayReasoningEffortLow
	case GatewayReasoningEffortMed:
		return GatewayReasoningEffortMed
	case GatewayReasoningEffortHigh:
		return GatewayReasoningEffortHigh
	case GatewayReasoningEffortXHigh:
		return GatewayReasoningEffortXHigh
	default:
		return ""
	}
}

func NormalizeGatewayAPIKeyDefaultReasoningEffort(effort string) string {
	if strings.TrimSpace(effort) == "" {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(effort), GatewayReasoningEffortNone) {
		return GatewayReasoningEffortNone
	}
	return NormalizeGatewayReasoningEffort(effort)
}

func GatewayUpstreamStrategy(mode string, fallbackEnabled bool) string {
	mode = NormalizeGatewayUpstreamMode(mode)
	if mode == GatewayUpstreamModeCodexCLI {
		return GatewayUpstreamStrategyCodexCLI
	}
	if fallbackEnabled {
		return GatewayUpstreamStrategyBackendWithFallback
	}
	return GatewayUpstreamStrategyBackendOnly
}

func DefaultGatewayUpstreamFallbackEnabled() bool {
	return parseGatewayBool(os.Getenv("CODEX_UPSTREAM_FALLBACK_ENABLED"))
}

func parseGatewayBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func formatGatewayBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (uc *GatewayUsecase) CreateAPIKey(ctx context.Context, input CreateGatewayAPIKeyInput) (*CreatedGatewayAPIKey, error) {
	name, err := NormalizeGatewayAPIKeyRemark(input.Name)
	if err != nil {
		return nil, err
	}
	input.Name = name
	input.QuotaTotalTokens = normalizeNonNegativeInt64(input.QuotaTotalTokens)
	input.QuotaDailyTokens = normalizeNonNegativeInt64(input.QuotaDailyTokens)
	input.QuotaWeeklyTokens = normalizeNonNegativeInt64(input.QuotaWeeklyTokens)
	input.QuotaDailyInputTokens = normalizeNonNegativeInt64(input.QuotaDailyInputTokens)
	input.QuotaWeeklyInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyInputTokens)
	input.QuotaDailyOutputTokens = normalizeNonNegativeInt64(input.QuotaDailyOutputTokens)
	input.QuotaWeeklyOutputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyOutputTokens)
	input.QuotaDailyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaDailyBillableInputTokens)
	input.QuotaWeeklyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyBillableInputTokens)
	rawUpstreamStrategy := strings.TrimSpace(input.UpstreamStrategy)
	input.UpstreamStrategy = NormalizeGatewayAPIKeyUpstreamStrategy(input.UpstreamStrategy)
	if rawUpstreamStrategy != "" && input.UpstreamStrategy == "" {
		return nil, ErrGatewayUpstreamModeInvalid
	}
	rawDefaultReasoningEffort := strings.TrimSpace(input.DefaultReasoningEffort)
	input.DefaultReasoningEffort = NormalizeGatewayAPIKeyDefaultReasoningEffort(input.DefaultReasoningEffort)
	if rawDefaultReasoningEffort != "" && input.DefaultReasoningEffort == "" {
		return nil, ErrGatewayReasoningInvalid
	}
	allowedModels, err := normalizeAllowedCodexModels(input.AllowedModels)
	if err != nil {
		return nil, err
	}
	input.AllowedModels = allowedModels

	secret, err := NewGatewayAPIKeySecretWithRemark(input.Name)
	if err != nil {
		return nil, err
	}
	if input.Name == "" {
		input.Name = "key" + secret.KeyHash[:8]
		secret, err = RebuildGatewayAPIKeySecretWithRemark(secret.PlainKey, input.Name)
		if err != nil {
			return nil, err
		}
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
	name, err := NormalizeGatewayAPIKeyRemark(input.Name)
	if err != nil {
		return nil, err
	}
	input.Name = name
	input.QuotaTotalTokens = normalizeNonNegativeInt64(input.QuotaTotalTokens)
	input.QuotaDailyTokens = normalizeNonNegativeInt64(input.QuotaDailyTokens)
	input.QuotaWeeklyTokens = normalizeNonNegativeInt64(input.QuotaWeeklyTokens)
	input.QuotaDailyInputTokens = normalizeNonNegativeInt64(input.QuotaDailyInputTokens)
	input.QuotaWeeklyInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyInputTokens)
	input.QuotaDailyOutputTokens = normalizeNonNegativeInt64(input.QuotaDailyOutputTokens)
	input.QuotaWeeklyOutputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyOutputTokens)
	input.QuotaDailyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaDailyBillableInputTokens)
	input.QuotaWeeklyBillableInputTokens = normalizeNonNegativeInt64(input.QuotaWeeklyBillableInputTokens)
	rawUpstreamStrategy := strings.TrimSpace(input.UpstreamStrategy)
	input.UpstreamStrategy = NormalizeGatewayAPIKeyUpstreamStrategy(input.UpstreamStrategy)
	if rawUpstreamStrategy != "" && input.UpstreamStrategy == "" {
		return nil, ErrGatewayUpstreamModeInvalid
	}
	rawDefaultReasoningEffort := strings.TrimSpace(input.DefaultReasoningEffort)
	input.DefaultReasoningEffort = NormalizeGatewayAPIKeyDefaultReasoningEffort(input.DefaultReasoningEffort)
	if rawDefaultReasoningEffort != "" && input.DefaultReasoningEffort == "" {
		return nil, ErrGatewayReasoningInvalid
	}
	allowedModels, err := normalizeAllowedCodexModels(input.AllowedModels)
	if err != nil {
		return nil, err
	}
	input.AllowedModels = allowedModels
	if input.Name == "" {
		input.Name = "key" + strconv.Itoa(input.ID)
	}
	return uc.repo.UpdateAPIKey(ctx, input)
}

func (uc *GatewayUsecase) ResetAPIKeySecret(ctx context.Context, id int) (*CreatedGatewayAPIKey, error) {
	if id <= 0 {
		return nil, ErrBadParam
	}
	current, err := uc.repo.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name, err := NormalizeGatewayAPIKeyRemark(current.Name)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = "key" + strconv.Itoa(id)
	}
	secret, err := NewGatewayAPIKeySecretWithRemark(name)
	if err != nil {
		return nil, err
	}
	item, err := uc.repo.ResetAPIKeySecret(ctx, id, secret)
	if err != nil {
		return nil, err
	}
	return &CreatedGatewayAPIKey{GatewayAPIKey: *item, PlainKey: secret.PlainKey}, nil
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
	filter.KeyIDs = normalizePositiveIDs(filter.KeyIDs)
	if len(filter.KeyIDs) == 0 && filter.KeyID > 0 {
		filter.KeyIDs = []int{filter.KeyID}
	}
	filter.SessionID = strings.TrimSpace(filter.SessionID)
	filter.Model = strings.TrimSpace(filter.Model)
	filter.Endpoint = strings.TrimSpace(filter.Endpoint)
	filter.UpstreamMode = NormalizeGatewayUpstreamMode(filter.UpstreamMode)
	filter.UpstreamErrorType = strings.TrimSpace(filter.UpstreamErrorType)
	filter.ErrorType = strings.TrimSpace(filter.ErrorType)
	filter.ExcludeErrorType = strings.TrimSpace(filter.ExcludeErrorType)
	return filter
}

func normalizePositiveIDs(ids []int) []int {
	out := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
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

func (uc *GatewayUsecase) GetContextSummary(ctx context.Context, sessionID string) (*GatewayContextSummary, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, nil
	}
	return uc.repo.GetContextSummary(ctx, sessionID)
}

func (uc *GatewayUsecase) UpsertContextSummary(ctx context.Context, item GatewayContextSummary) error {
	item.SessionID = strings.TrimSpace(item.SessionID)
	item.APIKeyPrefix = strings.TrimSpace(item.APIKeyPrefix)
	item.Summary = strings.TrimSpace(item.Summary)
	item.LastRequestID = strings.TrimSpace(item.LastRequestID)
	item.LastReason = strings.TrimSpace(item.LastReason)
	item.LastError = strings.TrimSpace(item.LastError)
	if item.SessionID == "" || item.Summary == "" {
		return nil
	}
	if item.SummaryTokens <= 0 {
		item.SummaryTokens = int64(len([]rune(item.Summary))+3) / 4
	}
	return uc.repo.UpsertContextSummary(ctx, item)
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

func (uc *GatewayUsecase) GetCodexUpstreamFallbackEnabled(ctx context.Context) (bool, error) {
	value, err := uc.repo.GetGatewaySetting(ctx, GatewaySettingCodexFallback)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(value) == "" {
		return DefaultGatewayUpstreamFallbackEnabled(), nil
	}
	return parseGatewayBool(value), nil
}

func (uc *GatewayUsecase) GetCodexUpstreamStrategy(ctx context.Context) (string, string, bool, error) {
	mode, err := uc.GetCodexUpstreamMode(ctx)
	if err != nil {
		return "", "", false, err
	}
	fallbackEnabled, err := uc.GetCodexUpstreamFallbackEnabled(ctx)
	if err != nil {
		return "", "", false, err
	}
	return GatewayUpstreamStrategy(mode, fallbackEnabled), mode, fallbackEnabled, nil
}

func (uc *GatewayUsecase) GetGatewayDefaultReasoningEffort(ctx context.Context) (string, error) {
	value, err := uc.repo.GetGatewaySetting(ctx, GatewaySettingDefaultReasoning)
	if err != nil {
		return "", err
	}
	return NormalizeGatewayReasoningEffort(value), nil
}

func (uc *GatewayUsecase) SetGatewayDefaultReasoningEffort(ctx context.Context, effort string) (string, error) {
	raw := strings.TrimSpace(effort)
	normalized := NormalizeGatewayReasoningEffort(raw)
	if raw != "" && normalized == "" {
		return "", ErrGatewayReasoningInvalid
	}
	if err := uc.repo.SetGatewaySetting(ctx, GatewaySettingDefaultReasoning, normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func (uc *GatewayUsecase) GetEffectiveReasoningEffort(ctx context.Context, key *GatewayAPIKey, requested string) (string, error) {
	requested = NormalizeGatewayReasoningEffort(requested)
	if key != nil {
		keyDefault := NormalizeGatewayAPIKeyDefaultReasoningEffort(key.DefaultReasoningEffort)
		switch keyDefault {
		case GatewayReasoningEffortNone:
			return requested, nil
		case GatewayReasoningEffortLow, GatewayReasoningEffortMed, GatewayReasoningEffortHigh, GatewayReasoningEffortXHigh:
			return keyDefault, nil
		}
	}
	globalDefault, err := uc.GetGatewayDefaultReasoningEffort(ctx)
	if err != nil {
		return "", err
	}
	if globalDefault != "" {
		return globalDefault, nil
	}
	return requested, nil
}

func (uc *GatewayUsecase) GetEffectiveCodexUpstreamStrategy(ctx context.Context, key *GatewayAPIKey) (string, string, bool, error) {
	if key != nil {
		switch NormalizeGatewayAPIKeyUpstreamStrategy(key.UpstreamStrategy) {
		case GatewayUpstreamStrategyBackendOnly:
			return GatewayUpstreamStrategyBackendOnly, GatewayUpstreamModeCodexBackend, false, nil
		case GatewayUpstreamStrategyBackendWithFallback:
			return GatewayUpstreamStrategyBackendWithFallback, GatewayUpstreamModeCodexBackend, true, nil
		case GatewayUpstreamStrategyCodexCLI:
			return GatewayUpstreamStrategyCodexCLI, GatewayUpstreamModeCodexCLI, false, nil
		}
	}
	return uc.GetCodexUpstreamStrategy(ctx)
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

func (uc *GatewayUsecase) SetCodexUpstreamFallbackEnabled(ctx context.Context, enabled bool) (bool, error) {
	if err := uc.repo.SetGatewaySetting(ctx, GatewaySettingCodexFallback, formatGatewayBool(enabled)); err != nil {
		return false, err
	}
	return enabled, nil
}

func (uc *GatewayUsecase) SetCodexUpstreamStrategy(ctx context.Context, strategy string) (string, string, bool, error) {
	strategy = NormalizeGatewayUpstreamStrategy(strategy)
	var mode string
	var fallbackEnabled bool
	switch strategy {
	case GatewayUpstreamStrategyBackendOnly:
		mode = GatewayUpstreamModeCodexBackend
	case GatewayUpstreamStrategyBackendWithFallback:
		mode = GatewayUpstreamModeCodexBackend
		fallbackEnabled = true
	case GatewayUpstreamStrategyCodexCLI:
		mode = GatewayUpstreamModeCodexCLI
	default:
		return "", "", false, ErrGatewayUpstreamModeInvalid
	}
	if err := uc.repo.SetGatewaySetting(ctx, GatewaySettingCodexUpstreamMode, mode); err != nil {
		return "", "", false, err
	}
	if err := uc.repo.SetGatewaySetting(ctx, GatewaySettingCodexFallback, formatGatewayBool(fallbackEnabled)); err != nil {
		return "", "", false, err
	}
	return strategy, mode, fallbackEnabled, nil
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

func (uc *GatewayUsecase) GetModelByID(ctx context.Context, modelID string) (*GatewayModel, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, ErrBadParam
	}
	return uc.repo.GetModelByID(ctx, modelID)
}

func (uc *GatewayUsecase) UpdateModelContextPolicy(ctx context.Context, id int, policy GatewayModelContextPolicy) (*GatewayModel, error) {
	if id <= 0 {
		return nil, ErrBadParam
	}
	if err := validateGatewayModelContextPolicy(policy); err != nil {
		return nil, err
	}
	return uc.repo.UpdateModelContextPolicy(ctx, id, policy)
}

func GatewayContextFallbackPolicyFromEnv() GatewayModelContextPolicy {
	return GatewayModelContextPolicy{
		ContextCompactTokens: int64FromEnvOptional("GATEWAY_CONTEXT_COMPACT_TOKENS"),
		ContextHardTokens:    int64FromEnvOptional("GATEWAY_CONTEXT_HARD_TOKENS"),
		ContextCompactBytes:  int64FromEnvOptional("GATEWAY_CONTEXT_COMPACT_BYTES"),
		ContextHardBytes:     int64FromEnvOptional("GATEWAY_CONTEXT_HARD_BYTES"),
		ContextKeepItems:     intFromEnvOptional("GATEWAY_CONTEXT_KEEP_ITEMS"),
	}
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

func intFromEnvOptional(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func int64FromEnvOptional(key string) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
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

func validateGatewayModelContextPolicy(policy GatewayModelContextPolicy) error {
	const maxTokens = int64(2_000_000)
	const maxBytes = int64(32 << 20)
	if policy.ContextWindowTokens < 0 ||
		policy.ContextCompactTokens < 0 ||
		policy.ContextHardTokens < 0 ||
		policy.ContextCompactBytes < 0 ||
		policy.ContextHardBytes < 0 ||
		policy.ContextKeepItems < 0 {
		return ErrGatewayModelContextInvalid
	}
	for _, value := range []int64{policy.ContextWindowTokens, policy.ContextCompactTokens, policy.ContextHardTokens} {
		if value > maxTokens {
			return ErrGatewayModelContextInvalid
		}
	}
	for _, value := range []int64{policy.ContextCompactBytes, policy.ContextHardBytes} {
		if value > maxBytes {
			return ErrGatewayModelContextInvalid
		}
	}
	if policy.ContextCompactTokens > 0 && policy.ContextHardTokens > 0 && policy.ContextHardTokens <= policy.ContextCompactTokens {
		return ErrGatewayModelContextInvalid
	}
	if policy.ContextCompactBytes > 0 && policy.ContextHardBytes > 0 && policy.ContextHardBytes <= policy.ContextCompactBytes {
		return ErrGatewayModelContextInvalid
	}
	if policy.ContextWindowTokens > 0 {
		if policy.ContextCompactTokens > policy.ContextWindowTokens || policy.ContextHardTokens > policy.ContextWindowTokens {
			return ErrGatewayModelContextInvalid
		}
	}
	if policy.ContextKeepItems > 0 && (policy.ContextKeepItems < 2 || policy.ContextKeepItems > 50) {
		return ErrGatewayModelContextInvalid
	}
	return nil
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
