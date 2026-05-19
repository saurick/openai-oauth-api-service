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
	defaultGatewayContextCompactBytes    = 850_000
	defaultGatewayContextHardBytes       = 1_050_000
	defaultGatewayContextCompactTokens   = 180_000
	defaultGatewayContextHardTokens      = 255_000
	defaultGatewayContextKeepItems       = 8
	defaultGatewayContextSummaryMaxRunes = 12_000
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
	text := promptTextForUsageEstimate(path, body)
	if strings.TrimSpace(text) == "" {
		text = string(body)
	}
	return estimateTokenCount(text)
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
	Text  string
}

func compactChatPayload(payload map[string]any, keepItems int, old *[]gatewayContextTextSegment) bool {
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	if len(messages) == 1 {
		message := mapValue(messages[0])
		if message == nil || !compactLargeContent(message, old, "chat."+stringValue(message["role"])) {
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
		if text := gatewayContextTextFromValue(mapValue(item)); text != "" {
			*old = append(*old, gatewayContextTextSegment{Label: "chat." + stringValue(mapValue(item)["role"]), Text: text})
		}
	}
	next := make([]any, 0, leadEnd+1+len(messages)-keepStart)
	next = append(next, messages[:leadEnd]...)
	next = append(next, map[string]any{"role": "user", "content": gatewayContextSummaryMessagePlaceholder})
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
	if !compactLargeContent(message, old, "chat."+stringValue(message["role"])) {
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
		*old = append(*old, gatewayContextTextSegment{Label: "responses.input", Text: v})
		payload["input"] = gatewayContextSummaryMessagePlaceholder + "\n\n" + tailText(v, 2000)
		return true
	case []any:
		if len(v) == 0 {
			return false
		}
		if len(v) == 1 {
			item := mapValue(v[0])
			if item == nil || !compactLargeContent(item, old, "responses."+stringValue(item["type"])) {
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
			if text := gatewayContextTextFromValue(mapValue(item)); text != "" {
				*old = append(*old, gatewayContextTextSegment{Label: "responses." + stringValue(mapValue(item)["type"]), Text: text})
			}
		}
		next := make([]any, 0, 1+len(v)-keepStart)
		next = append(next, map[string]any{
			"type":    "message",
			"role":    "user",
			"content": []any{map[string]any{"type": "input_text", "text": gatewayContextSummaryMessagePlaceholder}},
		})
		next = append(next, v[keepStart:]...)
		payload["input"] = next
		return len(*old) > 0
	default:
		return false
	}
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
	if !compactLargeContent(item, old, "responses."+stringValue(item["type"])) {
		return false
	}
	items[largestIndex] = item
	return true
}

const gatewayContextSummaryMessagePlaceholder = "[gateway_context_summary]"

func insertGatewayContextSummary(path string, payload map[string]any, summary string) {
	message := "以下是网关自动压缩的较早工程会话摘要。请把它作为历史上下文，优先遵循后续未压缩的最新消息。\n\n" + summary
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

func compactLargeContent(item map[string]any, old *[]gatewayContextTextSegment, label string) bool {
	if item == nil {
		return false
	}
	text := gatewayContextTextFromValue(item)
	if len([]rune(text)) < 4000 {
		return false
	}
	*old = append(*old, gatewayContextTextSegment{Label: label, Text: text})
	replaceContentText(item, gatewayContextSummaryMessagePlaceholder+"\n\n"+tailText(text, 2000))
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
	lines := make([]string, 0, 64)
	lines = append(lines, "## 自动压缩摘要")
	lines = append(lines, "- 目的：保留较早工程上下文，减少发送给上游模型的历史体积。")
	if prev := strings.TrimSpace(previous); prev != "" {
		lines = append(lines, "", "## 已有摘要", limitRunes(prev, 3000))
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
