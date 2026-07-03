package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"server/internal/biz"
)

const (
	gatewayContextErrorType              = "context_length_exceeded"
	defaultGatewayContextCompactBytes    = 1_000_000
	defaultGatewayContextHardBytes       = 1_900_000
	defaultGatewayContextCompactTokens   = 180_000
	defaultGatewayContextHardTokens      = 255_000
	defaultGatewayContextKeepItems       = 8
	defaultGatewayContextSummaryMaxRunes = 12_000
	defaultGatewayContextHeadMaxRunes    = 2_000
	defaultGatewayContextRecentMaxRunes  = 8_000
	defaultGatewayContextAnchorMaxRunes  = 4_000
)

var errGatewayContextLengthExceeded = errors.New("上下文已超过模型窗口，请压缩历史或开启新会话")

type gatewayContextPreparation struct {
	Body       []byte
	Diagnostic biz.GatewayUsageDiagnostic
}

type gatewayContextCompaction struct {
	Body            []byte
	Summary         string
	Reason          string
	OriginalBytes   int64
	CompactedBytes  int64
	OriginalTokens  int64
	CompactedTokens int64
	Changed         bool
}

func (h *openAIGatewayHandler) prepareGatewayContext(ctx context.Context, key *biz.GatewayAPIKey, requestID string, sessionID string, path string, body []byte, requestModel string, reasoningEffort string) (gatewayContextPreparation, error) {
	diagnostic := gatewayUsageDiagnosticForRequest(path, body, reasoningEffort)
	policy := h.effectiveGatewayContextPolicy(ctx, requestModel)
	diagnostic.ContextOriginalBytes = int64(len(body))
	diagnostic.ContextOriginalEstimatedTokens = estimateGatewayRequestTokens(path, body)
	diagnostic.ContextWindowTokens = policy.ContextWindowTokens
	diagnostic.ContextCompactTokenLimit = policy.ContextCompactTokens
	diagnostic.ContextHardTokenLimit = policy.ContextHardTokens
	diagnostic.ContextCompactByteLimit = policy.ContextCompactBytes
	diagnostic.ContextHardByteLimit = policy.ContextHardBytes
	diagnostic.ContextKeepItems = policy.ContextKeepItems

	if !gatewayContextNeedsCompaction(len(body), diagnostic.ContextOriginalEstimatedTokens, policy) {
		return gatewayContextPreparation{Body: body, Diagnostic: diagnostic}, nil
	}

	previousSummary := ""
	previousCount := 0
	if h != nil && h.gatewayUC != nil && strings.TrimSpace(sessionID) != "" {
		if stored, err := h.gatewayUC.GetContextSummary(ctx, sessionID); err == nil && stored != nil {
			previousSummary = stored.Summary
			previousCount = stored.CompactionCount
		} else if err != nil && h.log != nil {
			h.log.WithContext(ctx).Warnf("get gateway context summary failed request_id=%s session_id=%s err=%v", requestID, sessionID, err)
		}
	}

	compacted, err := compactGatewayContextRequest(path, body, previousSummary, policy.ContextKeepItems)
	if err != nil {
		diagnostic.ContextCompactionReason = "parse_failed"
		diagnostic.UpstreamBody = summarizeErrorForUsage(err)
		return gatewayContextPreparation{Body: body, Diagnostic: diagnostic}, errGatewayContextLengthExceeded
	}
	if !compacted.Changed {
		diagnostic.ContextCompactionReason = "not_compactable"
		return gatewayContextPreparation{Body: body, Diagnostic: diagnostic}, errGatewayContextLengthExceeded
	}

	diagnostic.ContextCompacted = true
	diagnostic.ContextCompactionReason = compacted.Reason
	diagnostic.ContextCompactionSummary = compacted.Summary
	diagnostic.ContextCompactionCount = previousCount + 1
	diagnostic.ContextOriginalBytes = compacted.OriginalBytes
	diagnostic.ContextCompactedBytes = compacted.CompactedBytes
	diagnostic.ContextOriginalEstimatedTokens = compacted.OriginalTokens
	diagnostic.ContextCompactedEstimatedTokens = compacted.CompactedTokens

	if h != nil && h.gatewayUC != nil && strings.TrimSpace(sessionID) != "" {
		item := biz.GatewayContextSummary{
			SessionID:           sessionID,
			Summary:             compacted.Summary,
			SummaryTokens:       estimateTokenCount(compacted.Summary),
			CompactionCount:     previousCount + 1,
			LastRequestID:       requestID,
			LastReason:          compacted.Reason,
			LastOriginalBytes:   compacted.OriginalBytes,
			LastCompactedBytes:  compacted.CompactedBytes,
			LastOriginalTokens:  compacted.OriginalTokens,
			LastCompactedTokens: compacted.CompactedTokens,
			UpdatedAt:           time.Now(),
		}
		if key != nil {
			item.APIKeyID = key.ID
			item.APIKeyPrefix = key.KeyPrefix
		}
		if err := h.gatewayUC.UpsertContextSummary(ctx, item); err != nil && h.log != nil {
			h.log.WithContext(ctx).Warnf("upsert gateway context summary failed request_id=%s session_id=%s err=%v", requestID, sessionID, err)
		}
	}

	if gatewayContextExceedsHardLimit(len(compacted.Body), compacted.CompactedTokens, policy) {
		return gatewayContextPreparation{Body: compacted.Body, Diagnostic: diagnostic}, errGatewayContextLengthExceeded
	}
	return gatewayContextPreparation{Body: compacted.Body, Diagnostic: diagnostic}, nil
}

func (h *openAIGatewayHandler) effectiveGatewayContextPolicy(ctx context.Context, modelID string) biz.GatewayModelContextPolicy {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = biz.DefaultCodexModelID
	}
	var model *biz.GatewayModel
	if h != nil && h.gatewayUC != nil {
		item, err := h.gatewayUC.GetModelByID(ctx, modelID)
		if err == nil {
			model = item
		} else if h.log != nil {
			h.log.WithContext(ctx).Warnf("get gateway model context policy failed model=%s err=%v", modelID, err)
		}
	}
	if model == nil {
		model = &biz.GatewayModel{ModelID: modelID}
	}
	return normalizeGatewayContextPolicy(biz.EffectiveGatewayModelContextPolicy(model, biz.GatewayContextFallbackPolicyFromEnv()))
}

func normalizeGatewayContextPolicy(policy biz.GatewayModelContextPolicy) biz.GatewayModelContextPolicy {
	if policy.ContextCompactTokens <= 0 {
		policy.ContextCompactTokens = defaultGatewayContextCompactTokens
	}
	if policy.ContextHardTokens <= 0 {
		policy.ContextHardTokens = defaultGatewayContextHardTokens
	}
	if policy.ContextCompactBytes <= 0 {
		policy.ContextCompactBytes = defaultGatewayContextCompactBytes
	}
	if policy.ContextHardBytes <= 0 {
		policy.ContextHardBytes = defaultGatewayContextHardBytes
	}
	if policy.ContextKeepItems <= 0 {
		policy.ContextKeepItems = defaultGatewayContextKeepItems
	}
	if policy.ContextKeepItems < 2 {
		policy.ContextKeepItems = 2
	}
	return policy
}

func gatewayContextNeedsCompaction(bytes int, tokens int64, policy biz.GatewayModelContextPolicy) bool {
	return int64(bytes) >= policy.ContextCompactBytes || tokens >= policy.ContextCompactTokens
}

func gatewayContextExceedsHardLimit(bytes int, tokens int64, policy biz.GatewayModelContextPolicy) bool {
	return int64(bytes) >= policy.ContextHardBytes || tokens >= policy.ContextHardTokens
}

func estimateGatewayRequestTokens(path string, body []byte) int64 {
	text := gatewayContextTextForUsageEstimate(path, body)
	if strings.TrimSpace(text) == "" {
		text = string(body)
	}
	return estimateTokenCount(text)
}

func gatewayContextTextForUsageEstimate(path string, body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	parts := make([]string, 0, 64)
	if instructions := stringValue(payload["instructions"]); instructions != "" {
		parts = append(parts, instructions)
	}
	if path == "/v1/responses" {
		appendGatewayContextEstimateText(&parts, payload["input"])
	} else {
		appendGatewayContextEstimateText(&parts, payload["messages"])
	}
	if tools := payload["tools"]; tools != nil {
		if raw, err := json.Marshal(tools); err == nil {
			parts = append(parts, string(raw))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func appendGatewayContextEstimateText(parts *[]string, value any) {
	switch v := value.(type) {
	case string:
		if text := strings.TrimSpace(v); text != "" {
			*parts = append(*parts, text)
		}
	case []any:
		for _, item := range v {
			appendGatewayContextEstimateText(parts, item)
		}
	case map[string]any:
		for _, key := range []string{"content", "text", "arguments", "output"} {
			appendGatewayContextEstimateText(parts, v[key])
		}
		for _, key := range []string{"function", "tool_calls", "summary"} {
			appendGatewayContextEstimateText(parts, v[key])
		}
	}
}

func compactGatewayContextRequest(path string, body []byte, previousSummary string, keepItems int) (gatewayContextCompaction, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return gatewayContextCompaction{}, err
	}
	originalTokens := estimateGatewayRequestTokens(path, body)
	originalBytes := int64(len(body))
	if keepItems < 2 {
		keepItems = 2
	}

	var old []gatewayContextTextSegment
	changed := false
	if path == "/v1/responses" {
		changed = compactResponsesPayload(payload, keepItems, &old)
	} else {
		changed = compactChatPayload(payload, keepItems, &old)
	}
	if !changed || len(old) == 0 {
		return gatewayContextCompaction{Changed: false, OriginalBytes: originalBytes, OriginalTokens: originalTokens}, nil
	}
	compactLargePayloadStrings(payload, &old, []string{"payload"})

	summary := buildGatewayContextSummary(previousSummary, old)
	insertGatewayContextSummary(path, payload, summary)

	nextBody, err := json.Marshal(payload)
	if err != nil {
		return gatewayContextCompaction{}, err
	}
	return gatewayContextCompaction{
		Body:            nextBody,
		Summary:         summary,
		Reason:          "request_preflight",
		OriginalBytes:   originalBytes,
		CompactedBytes:  int64(len(nextBody)),
		OriginalTokens:  originalTokens,
		CompactedTokens: estimateGatewayRequestTokens(path, nextBody),
		Changed:         true,
	}, nil
}

type gatewayContextTextSegment struct {
	Label string
	Role  string
	Text  string
}

type gatewayContextRestoreState struct {
	SchemaVersion                     string                          `json:"schema_version"`
	CurrentUserGoal                   string                          `json:"current_user_goal"`
	CurrentActiveGoal                 string                          `json:"current_active_goal"`
	LatestUserInstruction             string                          `json:"latest_user_instruction"`
	LatestUncompressedUserInstruction string                          `json:"latest_uncompressed_user_instruction,omitempty"`
	PinnedRawUserDirectives           []string                        `json:"pinned_raw_user_directives"`
	CurrentEffectiveConstraints       []string                        `json:"current_effective_constraints,omitempty"`
	MustNotDo                         []string                        `json:"must_not_do"`
	AllowedActions                    []string                        `json:"allowed_actions"`
	RequiresUserConfirmationBefore    []string                        `json:"requires_user_confirmation_before"`
	CurrentTaskPhase                  string                          `json:"current_task_phase"`
	CompletedWork                     []string                        `json:"completed_work"`
	RemainingWork                     []string                        `json:"remaining_work"`
	NextAction                        gatewayContextRestoreNextAction `json:"next_action"`
	Uncertainties                     []string                        `json:"uncertainties"`
	HistoricalContextOnly             []string                        `json:"historical_context_only"`
	ObsoleteOrSupersededGoals         []string                        `json:"obsolete_or_superseded_goals"`
	ObsoleteOrSupersededConstraints   []string                        `json:"obsolete_or_superseded_constraints,omitempty"`
	RestoreAudit                      []string                        `json:"restore_audit"`
	Source                            string                          `json:"source"`
}

type gatewayContextRestoreNextAction struct {
	Type                 string `json:"type"`
	Description          string `json:"description"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

func compactChatPayload(payload map[string]any, keepItems int, old *[]gatewayContextTextSegment) bool {
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	if len(messages) == 1 {
		message := mapValue(messages[0])
		if message == nil || !compactLargeContent(message, old, "chat."+stringValue(message["role"]), stringValue(message["role"])) {
			return false
		}
		messages[0] = message
		payload["messages"] = messages
		return true
	}

	leadEnd := 0
	for leadEnd < len(messages) {
		role := strings.TrimSpace(stringValue(mapValue(messages[leadEnd])["role"]))
		if role != "system" && role != "developer" {
			break
		}
		leadEnd++
	}
	keepStart := len(messages) - keepItems
	if keepStart < leadEnd {
		keepStart = leadEnd
	}
	for keepStart > leadEnd && chatMessageNeedsPrevious(mapValue(messages[keepStart])) {
		keepStart--
	}
	if keepStart <= leadEnd {
		return compactLargestChatMessage(messages, old)
	}
	for _, item := range messages[leadEnd:keepStart] {
		message := mapValue(item)
		role := stringValue(message["role"])
		if text := gatewayContextTextFromValue(message); text != "" {
			*old = append(*old, gatewayContextTextSegment{Label: "chat." + role, Role: role, Text: text})
		}
	}
	next := make([]any, 0, leadEnd+1+len(messages)-keepStart)
	next = append(next, messages[:leadEnd]...)
	next = append(next, map[string]any{"role": "user", "content": gatewayContextSummaryPlaceholderWithHandoff(*old)})
	next = append(next, messages[keepStart:]...)
	payload["messages"] = next
	return len(*old) > 0
}

func compactLargestChatMessage(messages []any, old *[]gatewayContextTextSegment) bool {
	largestIndex := -1
	largestLen := 0
	for i, item := range messages {
		message := mapValue(item)
		text := gatewayContextTextFromValue(message)
		if len([]rune(text)) > largestLen {
			largestIndex = i
			largestLen = len([]rune(text))
		}
	}
	if largestIndex < 0 || largestLen < 4000 {
		return false
	}
	message := mapValue(messages[largestIndex])
	if !compactLargeContent(message, old, "chat."+stringValue(message["role"]), stringValue(message["role"])) {
		return false
	}
	messages[largestIndex] = message
	return true
}

func compactResponsesPayload(payload map[string]any, keepItems int, old *[]gatewayContextTextSegment) bool {
	input := payload["input"]
	switch v := input.(type) {
	case string:
		if len([]rune(v)) < 4000 {
			return false
		}
		*old = append(*old, gatewayContextTextSegment{Label: "responses.input", Role: "user", Text: v})
		payload["input"] = gatewayContextSummaryMessagePlaceholder + "\n\n" + gatewayContextExecutionAnchorText("responses.input", v)
		return true
	case []any:
		if len(v) == 0 {
			return false
		}
		if len(v) == 1 {
			item := mapValue(v[0])
			if item == nil || !compactLargeContent(item, old, "responses."+stringValue(item["type"]), gatewayResponseItemRole(item)) {
				return false
			}
			v[0] = item
			payload["input"] = v
			return true
		}
		keepStart := len(v) - keepItems
		if keepStart < 0 {
			keepStart = 0
		}
		for keepStart > 0 && responsesItemNeedsPrevious(mapValue(v[keepStart])) {
			keepStart--
		}
		if keepStart == 0 {
			return compactLargestResponsesItem(v, old)
		}
		for _, item := range v[:keepStart] {
			m := mapValue(item)
			if text := gatewayContextTextFromValue(m); text != "" {
				*old = append(*old, gatewayContextTextSegment{Label: "responses." + stringValue(m["type"]), Role: gatewayResponseItemRole(m), Text: text})
			}
		}
		next := make([]any, 0, 1+len(v)-keepStart)
		next = append(next, map[string]any{
			"type":    "message",
			"role":    "user",
			"content": []any{map[string]any{"type": "input_text", "text": gatewayContextSummaryPlaceholderWithHandoff(*old)}},
		})
		next = append(next, v[keepStart:]...)
		compactLargeResponsesItems(next[1:], old)
		payload["input"] = next
		return len(*old) > 0
	default:
		return false
	}
}

func compactLargeResponsesItems(items []any, old *[]gatewayContextTextSegment) bool {
	changed := false
	for i, raw := range items {
		item := mapValue(raw)
		if item == nil {
			continue
		}
		text := gatewayContextTextFromValue(item)
		if strings.Contains(text, gatewayContextSummaryMessagePlaceholder) || len([]rune(text)) < 4000 {
			continue
		}
		if compactLargeContent(item, old, "responses.recent."+stringValue(item["type"]), gatewayResponseItemRole(item)) {
			items[i] = item
			changed = true
		}
	}
	return changed
}

func compactLargePayloadStrings(value any, old *[]gatewayContextTextSegment, path []string) bool {
	switch v := value.(type) {
	case map[string]any:
		changed := false
		for key, child := range v {
			nextPath := append(path, key)
			if gatewayContextSkipGenericCompaction(nextPath) {
				continue
			}
			if text, ok := child.(string); ok && gatewayContextCompactableStringKey(key) {
				if compactLargeStringValue(text, old, strings.Join(nextPath, "."), gatewayPayloadStringRole(nextPath), func(next string) {
					v[key] = next
				}) {
					changed = true
				}
				continue
			}
			if compactLargePayloadStrings(child, old, nextPath) {
				changed = true
			}
		}
		return changed
	case []any:
		changed := false
		for i, child := range v {
			nextPath := append(path, fmt.Sprintf("[%d]", i))
			if compactLargePayloadStrings(child, old, nextPath) {
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

func compactLargeStringValue(text string, old *[]gatewayContextTextSegment, label string, role string, replace func(string)) bool {
	if strings.Contains(text, gatewayContextSummaryMessagePlaceholder) || strings.Contains(text, "gateway_context_restore_state.v1") {
		return false
	}
	if len([]rune(text)) < 4000 {
		return false
	}
	*old = append(*old, gatewayContextTextSegment{Label: label, Role: role, Text: text})
	replace(gatewayContextSummaryMessagePlaceholder + "\n\n" + gatewayContextExecutionAnchorText(label, text))
	return true
}

func gatewayContextCompactableStringKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "instructions", "input", "content", "text", "arguments", "output":
		return true
	default:
		return false
	}
}

func gatewayContextSkipGenericCompaction(path []string) bool {
	for _, part := range path {
		switch part {
		case "tools", "tool_choice", "metadata":
			return true
		}
	}
	return false
}

func compactLargestResponsesItem(items []any, old *[]gatewayContextTextSegment) bool {
	largestIndex := -1
	largestLen := 0
	for i, item := range items {
		text := gatewayContextTextFromValue(mapValue(item))
		if len([]rune(text)) > largestLen {
			largestIndex = i
			largestLen = len([]rune(text))
		}
	}
	if largestIndex < 0 || largestLen < 4000 {
		return false
	}
	item := mapValue(items[largestIndex])
	if !compactLargeContent(item, old, "responses."+stringValue(item["type"]), gatewayResponseItemRole(item)) {
		return false
	}
	items[largestIndex] = item
	return true
}

const gatewayContextSummaryMessagePlaceholder = "[gateway_context_summary]"

func gatewayContextSummaryPlaceholderWithHandoff(segments []gatewayContextTextSegment) string {
	handoff := gatewayContextHandoffText(segments)
	if handoff == "" {
		return gatewayContextSummaryMessagePlaceholder
	}
	return gatewayContextSummaryMessagePlaceholder + "\n\n" + handoff
}

func insertGatewayContextSummary(path string, payload map[string]any, summary string) {
	message := "以下是网关自动压缩后的工程会话状态包。必须先读取其中的 JSON schema 字段并做 restore_audit，再决定是否继续；不要从 historical_context_only、obsolete_or_superseded_constraints 或历史路径/日志里自行生成新目标。latest_uncompressed_user_instruction 和后续未压缩用户消息优先于旧状态包；如果它明确允许继续执行或工具调用，不要继续套用旧的 stopped / must_not_do / requires_user_confirmation_before。只有 current_effective_constraints、must_not_do 和 current_task_phase 共同表示当前仍停止、暂停、只读或需要确认时，下一步才只能确认或停止。\n\n" + summary
	if path == "/v1/responses" {
		input, _ := payload["input"].([]any)
		for _, item := range input {
			m := mapValue(item)
			if text := gatewayContextTextFromValue(m); strings.Contains(text, gatewayContextSummaryMessagePlaceholder) {
				replaceContentText(m, strings.Replace(text, gatewayContextSummaryMessagePlaceholder, message, 1))
				return
			}
		}
		if s, ok := payload["input"].(string); ok && strings.Contains(s, gatewayContextSummaryMessagePlaceholder) {
			payload["input"] = strings.Replace(s, gatewayContextSummaryMessagePlaceholder, message, 1)
		}
		return
	}
	messages, _ := payload["messages"].([]any)
	for _, item := range messages {
		m := mapValue(item)
		if text := gatewayContextTextFromValue(m); strings.Contains(text, gatewayContextSummaryMessagePlaceholder) {
			replaceContentText(m, strings.Replace(text, gatewayContextSummaryMessagePlaceholder, message, 1))
			return
		}
	}
}

func compactLargeContent(item map[string]any, old *[]gatewayContextTextSegment, label string, role string) bool {
	if item == nil {
		return false
	}
	text := gatewayContextTextFromValue(item)
	if len([]rune(text)) < 4000 {
		return false
	}
	*old = append(*old, gatewayContextTextSegment{Label: label, Role: role, Text: text})
	replaceContentText(item, gatewayContextSummaryMessagePlaceholder+"\n\n"+gatewayContextExecutionAnchorText(label, text))
	return true
}

func chatMessageNeedsPrevious(message map[string]any) bool {
	if message == nil {
		return false
	}
	role := strings.TrimSpace(stringValue(message["role"]))
	return role == "tool"
}

func responsesItemNeedsPrevious(item map[string]any) bool {
	if item == nil {
		return false
	}
	return strings.TrimSpace(stringValue(item["type"])) == "function_call_output"
}

func gatewayResponseItemRole(item map[string]any) string {
	if item == nil {
		return ""
	}
	if role := strings.TrimSpace(stringValue(item["role"])); role != "" {
		return role
	}
	switch strings.TrimSpace(stringValue(item["type"])) {
	case "function_call_output":
		return "tool"
	case "message":
		content := item["content"]
		if parts, ok := content.([]any); ok {
			for _, part := range parts {
				switch stringValue(mapValue(part)["type"]) {
				case "output_text":
					return "assistant"
				case "input_text":
					return "user"
				}
			}
		}
	}
	return ""
}

func gatewayPayloadStringRole(path []string) string {
	joined := strings.Join(path, ".")
	switch {
	case strings.Contains(joined, ".instructions"):
		return "user"
	case strings.Contains(joined, ".input"), strings.Contains(joined, ".content"), strings.Contains(joined, ".text"):
		return "user"
	case strings.Contains(joined, ".output"):
		return "tool"
	default:
		return ""
	}
}

func gatewayContextTextFromValue(item map[string]any) string {
	if item == nil {
		return ""
	}
	content := contentValue(item["content"])
	parts := make([]string, 0, 3)
	if content.Text != "" {
		parts = append(parts, content.Text)
	}
	if v := stringValue(item["arguments"]); v != "" {
		parts = append(parts, v)
	}
	if v := stringValue(item["output"]); v != "" {
		parts = append(parts, v)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func replaceContentText(item map[string]any, text string) {
	if item == nil {
		return
	}
	switch content := item["content"].(type) {
	case string:
		item["content"] = text
	case []any:
		replaced := false
		for _, part := range content {
			m := mapValue(part)
			if m == nil {
				continue
			}
			switch stringValue(m["type"]) {
			case "text", "input_text", "output_text":
				if !replaced {
					m["text"] = text
					replaced = true
				} else {
					m["text"] = ""
				}
			}
		}
		if !replaced {
			content = append([]any{map[string]any{"type": "input_text", "text": text}}, content...)
		}
		item["content"] = content
	default:
		if _, ok := item["arguments"]; ok {
			item["arguments"] = text
		} else if _, ok := item["output"]; ok {
			item["output"] = text
		} else {
			item["content"] = text
		}
	}
}

func buildGatewayContextSummary(previous string, segments []gatewayContextTextSegment) string {
	state := buildGatewayContextRestoreState(previous, segments)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		return limitRunes(string(raw), defaultGatewayContextSummaryMaxRunes)
	}

	lines := make([]string, 0, 64)
	lines = append(lines, "## 自动压缩状态包")
	lines = append(lines, "- 目的：保留较早工程上下文，减少发送给上游模型的历史体积。")
	if prev := strings.TrimSpace(previous); prev != "" {
		lines = append(lines, "", "## 已有摘要", limitRunes(prev, 3000))
	}

	if anchors := recentGatewayProgressAnchors(segments, 8); len(anchors) > 0 {
		lines = append(lines, "", "## 最近进度与交接锚点")
		lines = append(lines, anchors...)
	}

	paths := uniqueMatches(segments, regexp.MustCompile(`([A-Za-z]:\\[^\s"'<>|]+|/(?:[\w.\-]+/)+[\w.\-]+|[\w.\-]+/[\w.\-\/]+)`), 40)
	if len(paths) > 0 {
		lines = append(lines, "", "## 文件与路径线索")
		for _, path := range paths {
			lines = append(lines, "- "+path)
		}
	}

	keyLines := keywordLines(segments, []string{
		"error", "failed", "failure", "panic", "timeout", "502", "499", "context", "上下文", "超限",
		"fallback", "stream", "heartbeat", "tool", "function_call", "go test", "pnpm", "make ",
	}, 80)
	if len(keyLines) > 0 {
		lines = append(lines, "", "## 关键排查线索")
		lines = append(lines, keyLines...)
	}

	lines = append(lines, "", "## 压缩片段")
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("### %s", segment.Label))
		lines = append(lines, headTailText(text, 700, 900))
	}
	return limitRunes(strings.TrimSpace(strings.Join(lines, "\n")), defaultGatewayContextSummaryMaxRunes)
}

func buildGatewayContextRestoreState(previous string, segments []gatewayContextTextSegment) gatewayContextRestoreState {
	previousState, hasPreviousState := parseGatewayContextRestoreState(previous)
	directives := pinnedGatewayRawDirectives(segments, 12)
	latestFromCurrentSegments := latestGatewayUserInstruction(segments)
	latestInstruction := latestFromCurrentSegments
	if latestInstruction == "" && len(directives) > 0 {
		latestInstruction = directives[len(directives)-1]
	}
	if latestInstruction == "" && hasPreviousState {
		latestInstruction = previousState.LatestUserInstruction
	}
	directives = gatewayResolveDirectiveConflicts(directives, latestInstruction)
	currentGoal := currentGatewayGoal(segments)
	if currentGoal == "" && hasPreviousState {
		currentGoal = previousState.CurrentUserGoal
	}
	phase := gatewayTaskPhase(latestInstruction, directives)
	mustNotDo := gatewayDirectiveLines(directives, gatewayStopOrRestrictionMarkers())
	if gatewayInstructionAllowsContinue(latestInstruction) {
		mustNotDo = nil
	}
	if len(mustNotDo) == 0 && hasPreviousState && strings.TrimSpace(latestFromCurrentSegments) == "" && !gatewayInstructionAllowsContinue(latestInstruction) {
		mustNotDo = previousState.MustNotDo
	}
	allowed := gatewayDirectiveLines(directives, gatewayAllowContinueMarkers())
	if gatewayInstructionAllowsContinue(latestInstruction) && len(allowed) == 0 {
		allowed = append(allowed, "最新用户消息明确允许继续执行和工具调用")
	}
	currentEffectiveConstraints := gatewayCurrentEffectiveConstraints(mustNotDo, allowed, latestInstruction)
	completed := gatewayDirectiveLines(gatewayAllProgressLines(segments), []string{"完成", "已完成", "通过", "修复完", "验证", "passed", "done", "fixed", "verified"})
	remaining := gatewayDirectiveLines(gatewayAllProgressLines(segments), []string{"下一步", "待办", "剩余", "之后", "继续", "remaining", "todo", "next"})
	uncertainties := make([]string, 0, 2)
	if strings.TrimSpace(currentGoal) == "" {
		uncertainties = append(uncertainties, "current_user_goal was not explicit in compacted text")
	}
	if strings.TrimSpace(latestInstruction) == "" {
		uncertainties = append(uncertainties, "latest_user_instruction was not explicit in compacted text")
	}
	requiresConfirmation := gatewayNeedsConfirmation(phase, mustNotDo, latestInstruction, uncertainties)
	nextAction := gatewayNextAction(phase, remaining, requiresConfirmation)
	if phase == "stopped" && len(mustNotDo) == 0 {
		mustNotDo = append(mustNotDo, "停止状态：不得继续执行工具调用")
	}
	return gatewayContextRestoreState{
		SchemaVersion:                     "gateway_context_restore_state.v1",
		CurrentUserGoal:                   currentGoal,
		CurrentActiveGoal:                 currentGoal,
		LatestUserInstruction:             latestInstruction,
		LatestUncompressedUserInstruction: latestFromCurrentSegments,
		PinnedRawUserDirectives:           directives,
		CurrentEffectiveConstraints:       currentEffectiveConstraints,
		MustNotDo:                         mustNotDo,
		AllowedActions:                    allowed,
		RequiresUserConfirmationBefore:    gatewayConfirmationActions(requiresConfirmation, mustNotDo),
		CurrentTaskPhase:                  phase,
		CompletedWork:                     completed,
		RemainingWork:                     remaining,
		NextAction:                        nextAction,
		Uncertainties:                     uncertainties,
		HistoricalContextOnly:             gatewayHistoricalContext(segments, previous, hasPreviousState),
		ObsoleteOrSupersededGoals:         obsoleteGatewayGoals(segments, currentGoal),
		ObsoleteOrSupersededConstraints:   gatewayObsoleteConstraints(previousState, hasPreviousState, segments, directives, latestInstruction),
		RestoreAudit: []string{
			"Confirm current_user_goal and latest_user_instruction before continuing; latest_uncompressed_user_instruction overrides older structured state.",
			"Treat current_effective_constraints, must_not_do, and current_task_phase as current state; treat historical_context_only and obsolete_or_superseded_constraints as non-current.",
			"If latest_uncompressed_user_instruction explicitly allows continuing, do not keep older stopped or must_not_do constraints active.",
			"If current_task_phase is stopped or waiting_for_user without a newer allow/continue instruction, do not call tools; ask the user or stop.",
			"If any uncertainty conflicts with the next action, ask the user before shell/file/browser/ssh/restart operations.",
		},
		Source: "gateway_preflight_compaction",
	}
}

func parseGatewayContextRestoreState(previous string) (gatewayContextRestoreState, bool) {
	previous = strings.TrimSpace(previous)
	if previous == "" {
		return gatewayContextRestoreState{}, false
	}
	var state gatewayContextRestoreState
	if err := json.Unmarshal([]byte(previous), &state); err == nil && state.SchemaVersion != "" {
		return state, true
	}
	start := strings.Index(previous, "{")
	end := strings.LastIndex(previous, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(previous[start:end+1]), &state); err == nil && state.SchemaVersion != "" {
			return state, true
		}
	}
	return gatewayContextRestoreState{}, false
}

func pinnedGatewayRawDirectives(segments []gatewayContextTextSegment, limit int) []string {
	markers := []string{
		"当前请求", "最新用户", "最近一条用户", "本轮目标", "当前目标", "授权", "允许", "必须", "一定要", "只能",
		"停止", "暂停", "不要", "禁止", "不得", "只读", "不要执行", "不要写", "不要重启",
		"current request", "latest user", "current goal", "authorized", "allowed", "must", "only",
		"stop", "pause", "do not", "don't", "must not", "read-only", "no tool",
	}
	lines := gatewayDirectiveLines(gatewayUserProgressLines(segments), markers)
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	for i := range lines {
		lines[i] = limitRunes(lines[i], 700)
	}
	return lines
}

func latestGatewayUserInstruction(segments []gatewayContextTextSegment) string {
	markers := append([]string{"当前请求", "最新用户", "最近一条用户", "本轮目标", "当前目标"}, gatewayStopOrRestrictionMarkers()...)
	lines := gatewayDirectiveLines(gatewayUserProgressLines(segments), markers)
	if len(lines) == 0 {
		return ""
	}
	return limitRunes(lines[len(lines)-1], 900)
}

func currentGatewayGoal(segments []gatewayContextTextSegment) string {
	lines := gatewayDirectiveLines(gatewayUserProgressLines(segments), []string{"本轮目标", "当前目标", "当前请求", "current goal", "current request"})
	if len(lines) == 0 {
		return ""
	}
	return limitRunes(lines[len(lines)-1], 900)
}

func gatewayUserProgressLines(segments []gatewayContextTextSegment) []string {
	out := make([]string, 0, 64)
	for _, segment := range segments {
		if !gatewaySegmentCanProvideUserInstruction(segment) {
			continue
		}
		for _, raw := range strings.Split(segment.Text, "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || len([]rune(line)) > 900 || gatewayLineIsCompactionMeta(line) {
				continue
			}
			out = append(out, line)
		}
	}
	return out
}

func gatewaySegmentCanProvideUserInstruction(segment gatewayContextTextSegment) bool {
	role := strings.ToLower(strings.TrimSpace(segment.Role))
	switch role {
	case "assistant", "tool", "system", "developer":
		return false
	case "user":
		return true
	}
	label := strings.ToLower(strings.TrimSpace(segment.Label))
	if strings.Contains(label, "assistant") || strings.Contains(label, "tool") || strings.Contains(label, "system") || strings.Contains(label, "developer") {
		return false
	}
	return strings.Contains(label, "input") || strings.Contains(label, "instructions") || strings.Contains(label, "content") || strings.Contains(label, "text")
}

func gatewayAllProgressLines(segments []gatewayContextTextSegment) []string {
	out := make([]string, 0, 64)
	for _, segment := range segments {
		for _, raw := range strings.Split(segment.Text, "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || len([]rune(line)) > 900 || gatewayLineIsCompactionMeta(line) {
				continue
			}
			out = append(out, line)
		}
	}
	return out
}

func gatewayDirectiveLines(lines []string, markers []string) []string {
	out := make([]string, 0, len(lines))
	seen := make(map[string]struct{})
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if lower == "" || gatewayLineIsCompactionMeta(line) {
			continue
		}
		matched := false
		for _, marker := range markers {
			if strings.Contains(lower, strings.ToLower(marker)) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	if len(out) > 16 {
		return out[len(out)-16:]
	}
	return out
}

func gatewayResolveDirectiveConflicts(directives []string, latest string) []string {
	if len(directives) == 0 {
		return directives
	}
	out := make([]string, 0, len(directives))
	switch {
	case gatewayInstructionAllowsContinue(latest):
		for _, directive := range directives {
			if gatewayLineIsStopOrRestriction(directive) {
				continue
			}
			out = append(out, directive)
		}
	case gatewayInstructionStops(latest):
		for _, directive := range directives {
			if gatewayLineAllowsContinue(directive) {
				continue
			}
			out = append(out, directive)
		}
	default:
		out = directives
	}
	if len(out) == 0 && strings.TrimSpace(latest) != "" {
		out = append(out, latest)
	}
	return out
}

func gatewayCurrentEffectiveConstraints(mustNotDo []string, allowed []string, latest string) []string {
	out := make([]string, 0, len(mustNotDo)+len(allowed)+1)
	out = append(out, mustNotDo...)
	if gatewayInstructionAllowsContinue(latest) {
		return []string{"latest_user_instruction 明确允许继续执行和工具调用"}
	}
	return append(out, allowed...)
}

func gatewayObsoleteConstraints(previous gatewayContextRestoreState, hasPrevious bool, segments []gatewayContextTextSegment, currentDirectives []string, latest string) []string {
	out := make([]string, 0, 8)
	current := make(map[string]struct{}, len(currentDirectives))
	for _, directive := range currentDirectives {
		current[directive] = struct{}{}
	}
	if hasPrevious && gatewayInstructionAllowsContinue(latest) {
		for _, item := range previous.MustNotDo {
			if strings.TrimSpace(item) != "" {
				out = append(out, limitRunes("superseded previous must_not_do: "+item, 500))
			}
		}
		if previous.CurrentTaskPhase == "stopped" || previous.CurrentTaskPhase == "waiting_for_user" {
			out = append(out, "superseded previous phase: "+previous.CurrentTaskPhase)
		}
	}
	for _, line := range gatewayDirectiveLines(gatewayAllProgressLines(segments), append(gatewayStopOrRestrictionMarkers(), gatewayAllowContinueMarkers()...)) {
		if _, ok := current[line]; ok {
			continue
		}
		if gatewayInstructionAllowsContinue(latest) && gatewayLineIsStopOrRestriction(line) {
			out = append(out, limitRunes("superseded by latest allow/continue: "+line, 500))
		}
	}
	return out
}

func gatewayLineIsCompactionMeta(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return true
	}
	for _, marker := range []string{
		"gateway_context_restore_state.v1",
		"gateway_preflight_compaction",
		"latest_user_instruction",
		"current_task_phase",
		"must_not_do",
		"requires_user_confirmation_before",
		"pinned_raw_user_directives",
		"historical_context_only",
		"obsolete_or_superseded",
		"restore_audit",
		"以下是网关自动压缩",
		"压缩恢复执行锚点",
		"若 latest_user_instruction",
		"treat current_effective_constraints",
		"confirm current_user_goal",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func gatewayStopOrRestrictionMarkers() []string {
	return []string{
		"停止", "暂停", "不要", "禁止", "不得", "不能", "只读", "不要执行", "不要写", "不要重启", "未经", "需要确认",
		"stop", "pause", "do not", "don't", "must not", "cannot", "read-only", "no tool", "without confirmation", "needs confirmation",
	}
}

func gatewayTaskPhase(latest string, directives []string) string {
	text := strings.ToLower(strings.Join(append([]string{latest}, directives...), "\n"))
	switch {
	case gatewayInstructionAllowsContinue(latest):
		return "executing"
	case strings.Contains(text, "停止") || strings.Contains(text, "stop"):
		return "stopped"
	case strings.Contains(text, "暂停") || strings.Contains(text, "只读") || strings.Contains(text, "等待") ||
		strings.Contains(text, "pause") || strings.Contains(text, "read-only") || strings.Contains(text, "waiting"):
		return "waiting_for_user"
	case strings.Contains(text, "验证") || strings.Contains(text, "回归") || strings.Contains(text, "测试") ||
		strings.Contains(text, "verify") || strings.Contains(text, "regression") || strings.Contains(text, "test"):
		return "verifying"
	default:
		return "executing"
	}
}

func gatewayNeedsConfirmation(phase string, mustNotDo []string, latest string, uncertainties []string) bool {
	if gatewayInstructionAllowsContinue(latest) && len(mustNotDo) == 0 {
		return false
	}
	if phase == "stopped" || phase == "waiting_for_user" {
		return true
	}
	if len(uncertainties) > 0 {
		return true
	}
	text := strings.ToLower(strings.Join(append(mustNotDo, latest), "\n"))
	return strings.Contains(text, "确认") || strings.Contains(text, "confirmation") || strings.Contains(text, "未经")
}

func gatewayInstructionAllowsContinue(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, marker := range gatewayAllowContinueMarkers() {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func gatewayInstructionStops(text string) bool {
	return gatewayLineIsStopOrRestriction(text) && !gatewayInstructionAllowsContinue(text)
}

func gatewayLineIsStopOrRestriction(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" || gatewayLineAllowsContinue(text) {
		return false
	}
	for _, marker := range gatewayStopOrRestrictionMarkers() {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func gatewayLineAllowsContinue(text string) bool {
	return gatewayInstructionAllowsContinue(text)
}

func gatewayAllowContinueMarkers() []string {
	return []string{
		"继续执行并允许工具调用",
		"明确允许工具调用",
		"允许工具调用",
		"可以调用工具",
		"继续执行",
		"继续做",
		"继续处理",
		"可以继续",
		"直接继续",
		"continue and allow tool",
		"allow tool calls",
		"authorized to use tools",
		"continue execution",
		"proceed",
	}
}

func gatewayConfirmationActions(required bool, mustNotDo []string) []string {
	base := []string{"tool_call", "file_write", "shell", "ssh", "restart_service"}
	if required || len(mustNotDo) > 0 {
		return base
	}
	return []string{}
}

func gatewayNextAction(phase string, remaining []string, requiresConfirmation bool) gatewayContextRestoreNextAction {
	if phase == "stopped" {
		return gatewayContextRestoreNextAction{Type: "no_op", Description: "用户已要求停止；不得继续工具调用。", RequiresConfirmation: true}
	}
	if requiresConfirmation {
		return gatewayContextRestoreNextAction{Type: "ask_user", Description: "恢复状态存在禁止事项或不确定性；继续前先确认。", RequiresConfirmation: true}
	}
	description := "按 latest_user_instruction 继续执行当前任务。"
	if len(remaining) > 0 {
		description = remaining[len(remaining)-1]
	}
	return gatewayContextRestoreNextAction{Type: "execute", Description: limitRunes(description, 700), RequiresConfirmation: false}
}

func gatewayHistoricalContext(segments []gatewayContextTextSegment, previous string, hasPreviousState bool) []string {
	out := make([]string, 0, 16)
	if prev := strings.TrimSpace(previous); prev != "" {
		label := "previous_summary"
		if hasPreviousState {
			label = "previous_structured_state"
		}
		out = append(out, label+": "+limitRunes(prev, 1000))
	}
	paths := uniqueMatches(segments, regexp.MustCompile(`([A-Za-z]:\\[^\s"'<>|]+|/(?:[\w.\-]+/)+[\w.\-]+|[\w.\-]+/[\w.\-\/]+)`), 10)
	for _, path := range paths {
		out = append(out, "path: "+path)
	}
	keyLines := keywordLines(segments, []string{"error", "failed", "failure", "panic", "timeout", "502", "499", "context", "上下文", "超限", "fallback", "stream"}, 10)
	for _, line := range keyLines {
		out = append(out, limitRunes(line, 400))
	}
	return out
}

func obsoleteGatewayGoals(segments []gatewayContextTextSegment, currentGoal string) []string {
	lines := gatewayDirectiveLines(gatewayAllProgressLines(segments), []string{"旧目标", "历史目标", "obsolete", "superseded", "old goal"})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(currentGoal) != "" && strings.Contains(line, currentGoal) {
			continue
		}
		out = append(out, limitRunes(line, 500))
	}
	return out
}

func recentGatewayContextText(text string) string {
	head := limitRunes(text, defaultGatewayContextHeadMaxRunes)
	tail := tailText(text, defaultGatewayContextRecentMaxRunes)
	parts := make([]string, 0, 3)
	if strings.TrimSpace(head) != "" {
		parts = append(parts, "## 原文开头\n"+head)
	}
	if gatewayContextHasProgressMarker(tail) {
		parts = append(parts, "## 原文末尾\n"+tail)
		return strings.TrimSpace(strings.Join(parts, "\n\n"))
	}
	anchor := recentGatewayProgressAnchorText(text)
	if anchor != "" {
		parts = append(parts, "## 最近进度锚点\n"+anchor)
	}
	parts = append(parts, "## 原文末尾\n"+tail)
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func gatewayContextExecutionAnchorText(label string, text string) string {
	body := recentGatewayContextText(text)
	if body == "" {
		return ""
	}
	return strings.TrimSpace("## 压缩恢复执行锚点\n以下内容来自被压缩原文；如果出现“当前请求 / 最新会话进度 / 下一步”，它仍是当前要继续执行的任务状态，不要当作旧摘要忽略。\n\n### " + label + "\n" + body)
}

func recentGatewayProgressAnchors(segments []gatewayContextTextSegment, limit int) []string {
	out := make([]string, 0, limit)
	seen := make(map[string]struct{})
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		anchor := recentGatewayProgressAnchorText(segment.Text)
		if anchor == "" {
			continue
		}
		anchor = limitRunes(anchor, 900)
		key := segment.Label + ":" + anchor
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, "- ["+segment.Label+"] "+strings.ReplaceAll(anchor, "\n", "\n  "))
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func gatewayContextHandoffText(segments []gatewayContextTextSegment) string {
	blocks := make([]string, 0, 4)
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		text := strings.TrimSpace(segment.Text)
		if text == "" || !gatewayContextHasProgressMarker(text) {
			continue
		}
		block := recentGatewayContextText(text)
		if block == "" {
			continue
		}
		blocks = append(blocks, "### "+segment.Label+"\n"+block)
		if len(blocks) >= 3 {
			break
		}
	}
	if len(blocks) == 0 {
		return ""
	}
	return strings.TrimSpace("## 压缩恢复执行锚点\n以下内容来自被压缩原文；如果出现“当前请求 / 最新会话进度 / 下一步”，它仍是当前要继续执行的任务状态，不要当作旧摘要忽略；若后续还有未压缩消息，则以后续未压缩消息为准。\n\n" + strings.Join(blocks, "\n\n"))
}

func recentGatewayProgressAnchorText(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return ""
	}
	start := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if gatewayContextHasProgressMarker(lines[i]) {
			start = i
			break
		}
	}
	if start < 0 {
		return ""
	}
	from := start - 6
	if from < 0 {
		from = 0
	}
	to := start + 18
	if to > len(lines) {
		to = len(lines)
	}
	block := strings.TrimSpace(strings.Join(lines[from:to], "\n"))
	return limitRunes(block, defaultGatewayContextAnchorMaxRunes)
}

func gatewayContextHasProgressMarker(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, marker := range []string{
		"最新会话进度",
		"当前请求",
		"当前进度",
		"当前状态",
		"本轮目标",
		"下一步",
		"待办",
		"已完成",
		"验证",
		"部署",
		"阻塞",
		"风险",
		"回归",
		"交接",
		"继续",
		"latest progress",
		"current progress",
		"current status",
		"next step",
		"todo",
		"done",
		"verified",
		"deployed",
		"blocked",
		"risk",
		"handoff",
		"resume",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func keywordLines(segments []gatewayContextTextSegment, keywords []string, limit int) []string {
	out := make([]string, 0, limit)
	seen := make(map[string]struct{})
	for _, segment := range segments {
		for _, raw := range strings.Split(segment.Text, "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || len([]rune(line)) > 260 {
				continue
			}
			lower := strings.ToLower(line)
			matched := false
			for _, keyword := range keywords {
				if strings.Contains(lower, strings.ToLower(keyword)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			key := segment.Label + ":" + line
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, "- ["+segment.Label+"] "+line)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func uniqueMatches(segments []gatewayContextTextSegment, pattern *regexp.Regexp, limit int) []string {
	out := make([]string, 0, limit)
	seen := make(map[string]struct{})
	for _, segment := range segments {
		for _, match := range pattern.FindAllString(segment.Text, -1) {
			match = strings.Trim(match, ".,;:()[]{}")
			if match == "" {
				continue
			}
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			out = append(out, match)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func headTailText(text string, head int, tail int) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= head+tail+64 {
		return text
	}
	return limitRunes(text, head) + "\n...\n" + tailText(text, tail)
}

func tailText(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[len(runes)-maxRunes:])
}

func limitRunes(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}
