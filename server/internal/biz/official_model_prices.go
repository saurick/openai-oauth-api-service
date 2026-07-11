package biz

const DefaultCodexModelID = "gpt-5.6-sol"

var CodexModelIDs = []string{
	DefaultCodexModelID,
	"gpt-5.6-terra",
	"gpt-5.6-luna",
	"gpt-5.5",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex",
	"gpt-5.3-codex-spark",
	"gpt-5.2",
}

var OfficialModelContextWindows = map[string]int64{
	DefaultCodexModelID:   1_050_000,
	"gpt-5.6-terra":       1_050_000,
	"gpt-5.6-luna":        1_050_000,
	"gpt-5.5":             400_000,
	"gpt-5.4":             400_000,
	"gpt-5.4-mini":        400_000,
	"gpt-5.3-codex":       400_000,
	"gpt-5.3-codex-spark": 400_000,
	"gpt-5.2":             400_000,
}

// OfficialModelPrices keeps Codex picker model prices that have published API token rates.
// Sources: OpenAI API model docs and Codex rate card, checked 2026-07-11.
var OfficialModelPrices = []*GatewayModelPrice{
	{ModelID: DefaultCodexModelID, InputUSDPerMillion: 5, CachedInputUSDPerMillion: 0.5, OutputUSDPerMillion: 30},
	{ModelID: "gpt-5.6-terra", InputUSDPerMillion: 2.5, CachedInputUSDPerMillion: 0.25, OutputUSDPerMillion: 15},
	{ModelID: "gpt-5.6-luna", InputUSDPerMillion: 1, CachedInputUSDPerMillion: 0.1, OutputUSDPerMillion: 6},
	{ModelID: "gpt-5.5", InputUSDPerMillion: 5, CachedInputUSDPerMillion: 0.5, OutputUSDPerMillion: 30},
	{ModelID: "gpt-5.4", InputUSDPerMillion: 2.5, CachedInputUSDPerMillion: 0.25, OutputUSDPerMillion: 15},
	{ModelID: "gpt-5.4-mini", InputUSDPerMillion: 0.75, CachedInputUSDPerMillion: 0.075, OutputUSDPerMillion: 4.5},
	{ModelID: "gpt-5.3-codex", InputUSDPerMillion: 1.75, CachedInputUSDPerMillion: 0.175, OutputUSDPerMillion: 14},
	{ModelID: "gpt-5.2", InputUSDPerMillion: 1.75, CachedInputUSDPerMillion: 0.175, OutputUSDPerMillion: 14},
}

func IsOfficialCodexModelID(modelID string) bool {
	return CodexModelIDSet()[modelID]
}

func CodexModelIDSet() map[string]bool {
	out := make(map[string]bool, len(CodexModelIDs))
	for _, modelID := range CodexModelIDs {
		if modelID == "" {
			continue
		}
		out[modelID] = true
	}
	return out
}

func OfficialModelContextWindowTokens(modelID string) int64 {
	return OfficialModelContextWindows[modelID]
}

func RecommendedGatewayContextPolicyForModel(modelID string) GatewayModelContextPolicy {
	window := OfficialModelContextWindowTokens(modelID)
	if window <= 0 {
		return GatewayModelContextPolicy{}
	}
	compactTokens := window * 65 / 100
	hardTokens := window * 95 / 100
	if window >= 400_000 {
		compactTokens = 260_000
		hardTokens = 380_000
	}
	return GatewayModelContextPolicy{
		ContextWindowTokens:  window,
		ContextCompactTokens: compactTokens,
		ContextHardTokens:    hardTokens,
		ContextCompactBytes:  minPositiveInt64(compactTokens*4, 1_000_000),
		ContextHardBytes:     hardTokens * 5,
		ContextKeepItems:     8,
	}
}

func minPositiveInt64(a, b int64) int64 {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}

func EffectiveGatewayModelContextPolicy(model *GatewayModel, fallback GatewayModelContextPolicy) GatewayModelContextPolicy {
	modelID := ""
	if model != nil {
		modelID = model.ModelID
	}
	recommended := RecommendedGatewayContextPolicyForModel(modelID)
	out := recommended
	if fallback.ContextWindowTokens > 0 {
		out.ContextWindowTokens = fallback.ContextWindowTokens
	}
	if fallback.ContextCompactTokens > 0 {
		out.ContextCompactTokens = fallback.ContextCompactTokens
	}
	if fallback.ContextHardTokens > 0 {
		out.ContextHardTokens = fallback.ContextHardTokens
	}
	if fallback.ContextCompactBytes > 0 {
		out.ContextCompactBytes = fallback.ContextCompactBytes
	}
	if fallback.ContextHardBytes > 0 {
		out.ContextHardBytes = fallback.ContextHardBytes
	}
	if fallback.ContextKeepItems > 0 {
		out.ContextKeepItems = fallback.ContextKeepItems
	}
	if recommended.ContextWindowTokens > 0 {
		out.ContextWindowTokens = recommended.ContextWindowTokens
	}
	if model == nil {
		return out
	}
	if model.ContextWindowTokens > 0 {
		out.ContextWindowTokens = model.ContextWindowTokens
	}
	if model.ContextCompactTokens > 0 {
		out.ContextCompactTokens = model.ContextCompactTokens
	}
	if model.ContextHardTokens > 0 {
		out.ContextHardTokens = model.ContextHardTokens
	}
	if model.ContextCompactBytes > 0 {
		out.ContextCompactBytes = model.ContextCompactBytes
	}
	if model.ContextHardBytes > 0 {
		out.ContextHardBytes = model.ContextHardBytes
	}
	if model.ContextKeepItems > 0 {
		out.ContextKeepItems = model.ContextKeepItems
	}
	return out
}

func OfficialModelPriceMap() map[string]*GatewayModelPrice {
	out := make(map[string]*GatewayModelPrice, len(OfficialModelPrices))
	for _, item := range OfficialModelPrices {
		if item == nil || item.ModelID == "" {
			continue
		}
		copied := *item
		out[copied.ModelID] = &copied
	}
	return out
}
