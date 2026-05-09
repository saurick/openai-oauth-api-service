package biz

const DefaultCodexModelID = "gpt-5.5"

var CodexModelIDs = []string{
	DefaultCodexModelID,
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex",
	"gpt-5.3-codex-spark",
	"gpt-5.2",
}

// OfficialModelPrices keeps Codex picker model prices that have published API token rates.
// Sources: OpenAI API model docs and Codex rate card, checked 2026-05-09.
var OfficialModelPrices = []*GatewayModelPrice{
	{ModelID: DefaultCodexModelID, InputUSDPerMillion: 5, CachedInputUSDPerMillion: 0.5, OutputUSDPerMillion: 30},
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
