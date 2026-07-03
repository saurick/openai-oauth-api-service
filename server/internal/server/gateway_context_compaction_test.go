package server

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"server/internal/biz"
)

func TestCompactGatewayContextChatPreservesRecentMessages(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_KEEP_ITEMS", "2")
	oldText := strings.Repeat("old failure log /tmp/app/main.go context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","stream":true,"messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"user","content":` + mustJSONQuote(oldText) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest question"},` +
		`{"role":"assistant","content":"latest answer"}` +
		`]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	if compacted.CompactedBytes >= compacted.OriginalBytes {
		t.Fatalf("compacted bytes = %d, original = %d", compacted.CompactedBytes, compacted.OriginalBytes)
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	messages := payload["messages"].([]any)
	if got := stringValue(messages[len(messages)-2].(map[string]any)["content"]); got != "latest question" {
		t.Fatalf("latest user message = %q", got)
	}
	if !strings.Contains(compacted.Summary, "/tmp/app/main.go") {
		t.Fatalf("summary missing path: %s", compacted.Summary)
	}
}

func TestCompactGatewayContextResponsesString(t *testing.T) {
	oldText := strings.Repeat("tool output line with 502 and stream heartbeat\n", 300)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(oldText) + `}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "previous summary", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "previous summary") {
		t.Fatalf("compacted input missing previous summary: %s", input)
	}
	if !strings.Contains(input, "tool output line") {
		t.Fatalf("compacted input missing tail context")
	}
}

func TestCompactGatewayContextResponsesStringPreservesLatestProgressAnchor(t *testing.T) {
	oldDiscussion := strings.Repeat("几个小时前的旧讨论：先处理 OAuth callback，再看 usage chart\n", 260)
	latestProgress := strings.Join([]string{
		"最新会话进度：已经修复上下文压缩的主路径，并且 server/internal/server 单测通过。",
		"验证：go test ./internal/server -count=1 通过。",
		"下一步：ssh sauri@192.168.0.45 到 Windows Codex 多会话压缩测试，之后部署到 192.168.0.133。",
	}, "\n")
	lateSchemaNoise := strings.Repeat("tool schema field description without current task signal\n", 260)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(oldDiscussion+latestProgress+"\n"+lateSchemaNoise) + `}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "最新会话进度：已经修复上下文压缩的主路径") {
		t.Fatalf("compacted input missing latest progress anchor: %s", input)
	}
	if !strings.Contains(input, "部署到 192.168.0.133") {
		t.Fatalf("compacted input missing deploy handoff: %s", input)
	}
	state := mustGatewayRestoreState(t, compacted.Summary)
	if !strings.Contains(state.LatestUserInstruction, "最新会话进度：已经修复上下文压缩的主路径") &&
		!strings.Contains(strings.Join(state.RemainingWork, "\n"), "部署到 192.168.0.133") {
		t.Fatalf("summary missing structured progress handoff: %+v", state)
	}
}

func TestCompactGatewayContextResponsesStringPreservesCurrentInstructionAtHead(t *testing.T) {
	currentInstruction := strings.Join([]string{
		"当前请求：只输出 MARKER=WIN-COMPRESS-HEAD 和 NEXT=部署到 192.168.0.133。",
		"不要检查文件，不要执行命令，不要把下面旧讨论当成新任务。",
	}, "\n")
	oldDiscussion := strings.Repeat("几个小时前的旧讨论：排查 OAuth callback 和 usage 图表。\n", 260)
	latestProgress := "最新会话进度：MARKER WIN-COMPRESS-HEAD 已完成压缩回归。\n下一步：部署到 192.168.0.133。\n"
	lateSchemaNoise := strings.Repeat("tool schema tail noise without current task signal\n", 260)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(currentInstruction+"\n"+oldDiscussion+latestProgress+lateSchemaNoise) + `}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "压缩恢复执行锚点") {
		t.Fatalf("compacted input missing execution anchor section: %s", input)
	}
	if !strings.Contains(input, "当前请求：只输出 MARKER=WIN-COMPRESS-HEAD") {
		t.Fatalf("compacted input missing current instruction head: %s", input)
	}
	if !strings.Contains(input, "不要检查文件，不要执行命令") {
		t.Fatalf("compacted input missing no-tool instruction: %s", input)
	}
	if !strings.Contains(input, "最新会话进度：MARKER WIN-COMPRESS-HEAD") {
		t.Fatalf("compacted input missing latest progress anchor: %s", input)
	}
}

func TestCompactGatewayContextResponsesArrayPreservesCompressedHandoff(t *testing.T) {
	currentInstruction := strings.Join([]string{
		"当前请求：只输出 MARKER=WIN-COMPRESS-ARRAY 和 NEXT=部署到 192.168.0.133。",
		"不要检查文件，不要执行命令，不要把下面旧讨论当成新任务。",
		"最新会话进度：已经完成本地单测，正在做 Windows Codex 多会话压缩回归。",
		"下一步：测试通过后部署到 192.168.0.133。",
	}, "\n")
	oldNoise := strings.Repeat("旧工具输出：context_length_exceeded /workspace/service/gateway.go\n", 120)
	tailNoise := strings.Repeat("尾部工具 schema 噪声，没有当前任务指令。\n", 80)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":[` +
		`{"type":"message","role":"system","content":[{"type":"input_text","text":"系统规则：blocked 时说明风险，next step 只作为通用说明，不是用户当前请求。"}]},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":` + mustJSONQuote(currentInstruction) + `}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"我会继续处理。"}]},` +
		`{"type":"function_call_output","call_id":"call_old","output":` + mustJSONQuote(oldNoise) + `},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":` + mustJSONQuote(tailNoise) + `}]},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"继续"}]}` +
		`]}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := payload["input"].([]any)
	inserted := gatewayContextTextFromValue(mapValue(input[0]))
	if !strings.Contains(inserted, "压缩恢复执行锚点") {
		t.Fatalf("inserted message missing handoff section: %s", inserted)
	}
	if !strings.Contains(inserted, "当前请求：只输出 MARKER=WIN-COMPRESS-ARRAY") {
		t.Fatalf("inserted message missing current instruction: %s", inserted)
	}
	if !strings.Contains(inserted, "Windows Codex 多会话压缩回归") {
		t.Fatalf("inserted message missing latest progress: %s", inserted)
	}
	if !strings.Contains(inserted, "部署到 192.168.0.133") {
		t.Fatalf("inserted message missing deploy handoff: %s", inserted)
	}
}

func TestCompactGatewayContextChatArrayPreservesCompressedHandoff(t *testing.T) {
	currentInstruction := strings.Join([]string{
		"当前请求：只输出 MARKER=CHAT-COMPRESS-ARRAY。",
		"最新会话进度：chat 压缩回归正在验证。",
		"下一步：部署到 192.168.0.133。",
	}, "\n")
	oldNoise := strings.Repeat("旧终端输出：go test ./... passed /workspace/server/main.go\n", 120)
	body := []byte(`{"model":"gpt-5.5","stream":true,"messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"user","content":` + mustJSONQuote(currentInstruction) + `},` +
		`{"role":"assistant","content":"处理中"},` +
		`{"role":"user","content":` + mustJSONQuote(oldNoise) + `},` +
		`{"role":"assistant","content":"latest answer"},` +
		`{"role":"user","content":"继续"}` +
		`]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	messages := payload["messages"].([]any)
	inserted := gatewayContextTextFromValue(mapValue(messages[1]))
	if !strings.Contains(inserted, "压缩恢复执行锚点") {
		t.Fatalf("inserted message missing handoff section: %s", inserted)
	}
	if !strings.Contains(inserted, "当前请求：只输出 MARKER=CHAT-COMPRESS-ARRAY") {
		t.Fatalf("inserted message missing current instruction: %s", inserted)
	}
	if !strings.Contains(inserted, "部署到 192.168.0.133") {
		t.Fatalf("inserted message missing deploy handoff: %s", inserted)
	}
}

func TestCompactGatewayContextCarriesPreviousSummaryAcrossCompactions(t *testing.T) {
	firstHistory := strings.Repeat("phase one failed at /workspace/service/auth.go with oauth callback timeout\n", 240)
	firstBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(firstHistory+"next step is patch auth callback handling") + `}`)

	first, err := compactGatewayContextRequest("/v1/responses", firstBody, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("expected first compaction")
	}
	if !strings.Contains(first.Summary, "/workspace/service/auth.go") {
		t.Fatalf("first summary missing auth path: %s", first.Summary)
	}

	secondHistory := strings.Repeat("phase two terminal output says regression passed but deploy smoke still pending\n", 240)
	secondBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(secondHistory+"继续，从上次打断的部署验证开始") + `}`)

	second, err := compactGatewayContextRequest("/v1/responses", secondBody, first.Summary, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Changed {
		t.Fatal("expected second compaction")
	}

	var payload map[string]any
	if err := json.Unmarshal(second.Body, &payload); err != nil {
		t.Fatal(err)
	}
	input := stringValue(payload["input"])
	if !strings.Contains(input, "/workspace/service/auth.go") {
		t.Fatalf("second compacted input missing previous summary: %s", input)
	}
	if !strings.Contains(input, "部署验证开始") {
		t.Fatalf("second compacted input missing latest resume instruction: %s", input)
	}
	if !strings.Contains(second.Summary, "/workspace/service/auth.go") {
		t.Fatalf("second summary did not carry first summary forward: %s", second.Summary)
	}
}

func TestCompactGatewayContextSingleLargeChatMessageIncludesSummary(t *testing.T) {
	oldText := strings.Repeat("single huge prompt with /workspace/app/main.go and context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":` + mustJSONQuote(oldText+"reply FINAL_OK") + `}]}`)

	compacted, err := compactGatewayContextRequest("/v1/chat/completions", body, "", 8)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	messages := payload["messages"].([]any)
	content := stringValue(messages[0].(map[string]any)["content"])
	if !strings.Contains(content, "gateway_context_restore_state.v1") {
		t.Fatalf("compacted content missing structured restore state: %s", content)
	}
	if !strings.Contains(content, "FINAL_OK") {
		t.Fatalf("compacted content missing tail instruction: %s", content)
	}
}

func TestCompactGatewayContextSummaryUsesStructuredStateForStopDirective(t *testing.T) {
	oldGoal := strings.Repeat("历史目标：修复 bug Y，不要把它当成当前任务。\n", 260)
	stopDirective := strings.Join([]string{
		"当前请求：停止做一切事情。",
		"不要检查文件，不要执行命令，不要 ssh，不要重启服务。",
	}, "\n")
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(oldGoal+stopDirective) + `}`)

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	state := mustGatewayRestoreState(t, compacted.Summary)
	if state.CurrentTaskPhase != "stopped" {
		t.Fatalf("phase = %q, want stopped; state=%+v", state.CurrentTaskPhase, state)
	}
	if state.NextAction.Type != "no_op" || !state.NextAction.RequiresConfirmation {
		t.Fatalf("next action = %+v, want no_op with confirmation", state.NextAction)
	}
	if !strings.Contains(strings.Join(state.MustNotDo, "\n"), "不要执行命令") {
		t.Fatalf("must_not_do missing no-command directive: %+v", state.MustNotDo)
	}
	if strings.Contains(state.CurrentUserGoal, "bug Y") {
		t.Fatalf("historical goal leaked into current goal: %+v", state)
	}
}

func TestCompactGatewayContextCarriesStructuredGoalAcrossRepeatedCompactions(t *testing.T) {
	goalA := "当前目标：只修 bug A。"
	firstBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(goalA+"\n"+strings.Repeat("历史日志：bug B 出现在旧日志里。\n", 260)) + `}`)
	first, err := compactGatewayContextRequest("/v1/responses", firstBody, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	firstState := mustGatewayRestoreState(t, first.Summary)
	if !strings.Contains(firstState.CurrentUserGoal, "bug A") {
		t.Fatalf("first goal = %q, want bug A", firstState.CurrentUserGoal)
	}

	secondBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote("继续。\n"+strings.Repeat("历史噪声：bug B / bug C 相关日志。\n", 260)) + `}`)
	second, err := compactGatewayContextRequest("/v1/responses", secondBody, first.Summary, 2)
	if err != nil {
		t.Fatal(err)
	}
	secondState := mustGatewayRestoreState(t, second.Summary)
	if !strings.Contains(secondState.CurrentUserGoal, "bug A") {
		t.Fatalf("second goal drifted: %+v", secondState)
	}
	if strings.Contains(secondState.CurrentUserGoal, "bug B") || strings.Contains(secondState.CurrentUserGoal, "bug C") {
		t.Fatalf("historical goal leaked into current goal: %+v", secondState)
	}
}

func TestCompactGatewayContextResponsesArrayCompactsHugeLatestResumeInstruction(t *testing.T) {
	firstGoal := "当前请求：只输出 MARKER=WIN_COMPRESS_SCHEMA_V1 和 ACTION=NO_TOOL。"
	firstBody := []byte(`{"model":"gpt-5.5","stream":true,"input":` + mustJSONQuote(firstGoal+"\n"+strings.Repeat("第一轮历史日志：context_length_exceeded 旧目标。\n", 260)) + `}`)
	first, err := compactGatewayContextRequest("/v1/responses", firstBody, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("expected first compaction")
	}

	latest := strings.Join([]string{
		"第二轮当前请求：只输出 MARKER=WIN_RESUME_COMPRESS_SCHEMA_V2 和 ACTION=NO_TOOL。",
		"禁止调用工具；不要延续第一轮 marker；历史 OLD_GOAL 都是噪声。",
	}, "\n")
	hugeLatest := latest + "\n" + strings.Repeat("HISTORICAL_CONTEXT_ONLY old-marker WIN_COMPRESS_SCHEMA_V1 obsolete goal drift restart_service shell file_write ssh tool_call denied.\n", 7600) +
		"\n最新用户指令原话：只输出 MARKER=WIN_RESUME_COMPRESS_SCHEMA_V2 和 ACTION=NO_TOOL。"
	secondBody := []byte(`{"model":"gpt-5.5","stream":true,"input":[` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"较早上下文：第一轮 marker 已完成。"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"处理中。"}]},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"中间历史：不要当成当前目标。"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"继续等待。"}]},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":` + mustJSONQuote(hugeLatest) + `}]}` +
		`]}`)
	if len(secondBody) <= 1_000_000 {
		t.Fatalf("test body length = %d, want > 1000000", len(secondBody))
	}

	second, err := compactGatewayContextRequest("/v1/responses", secondBody, first.Summary, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Changed {
		t.Fatal("expected second compaction")
	}
	if second.CompactedBytes >= 1_000_000 {
		t.Fatalf("compacted bytes = %d, want < 1000000", second.CompactedBytes)
	}
	state := mustGatewayRestoreState(t, second.Summary)
	if !strings.Contains(state.LatestUserInstruction, "WIN_RESUME_COMPRESS_SCHEMA_V2") {
		t.Fatalf("latest instruction drifted: %+v", state)
	}
	if strings.Contains(state.LatestUserInstruction, "WIN_COMPRESS_SCHEMA_V1") {
		t.Fatalf("latest instruction retained first marker: %+v", state)
	}
}

func TestCompactGatewayContextGenericSecondPassCompactsHugeInstructionString(t *testing.T) {
	hugeInstruction := "第二轮当前请求：只输出 MARKER=WIN_GENERIC_SECOND_PASS 和 ACTION=NO_TOOL。\n" +
		strings.Repeat("HISTORICAL_CONTEXT_ONLY old noise should not remain raw.\n", 7600) +
		"\n最新用户指令原话：只输出 MARKER=WIN_GENERIC_SECOND_PASS 和 ACTION=NO_TOOL。"
	oldInput := strings.Repeat("第一轮旧上下文：context_length_exceeded old marker.\n", 300)
	body := []byte(`{"model":"gpt-5.5","stream":true,"instructions":` + mustJSONQuote(hugeInstruction) + `,"input":` + mustJSONQuote(oldInput) + `}`)
	if len(body) <= 400000 {
		t.Fatalf("test body length = %d, want large payload", len(body))
	}

	compacted, err := compactGatewayContextRequest("/v1/responses", body, "", 8)
	if err != nil {
		t.Fatal(err)
	}
	if !compacted.Changed {
		t.Fatal("expected compaction")
	}
	if compacted.CompactedBytes >= int64(len(body)) {
		t.Fatalf("compacted bytes = %d, original = %d", compacted.CompactedBytes, len(body))
	}
	var payload map[string]any
	if err := json.Unmarshal(compacted.Body, &payload); err != nil {
		t.Fatal(err)
	}
	instructions := stringValue(payload["instructions"])
	if !strings.Contains(instructions, gatewayContextSummaryMessagePlaceholder) || len([]rune(instructions)) >= 20000 {
		t.Fatalf("huge instructions were not generically compacted, length=%d", len([]rune(instructions)))
	}
	state := mustGatewayRestoreState(t, compacted.Summary)
	if !strings.Contains(state.LatestUserInstruction, "WIN_GENERIC_SECOND_PASS") {
		t.Fatalf("latest instruction missing generic second-pass marker: %+v", state)
	}
}

func TestGatewayContextLatestUserAllowOverridesPreviousStoppedState(t *testing.T) {
	previous := gatewayContextRestoreState{
		SchemaVersion:                  "gateway_context_restore_state.v1",
		CurrentUserGoal:                "旧目标：停止所有操作。",
		CurrentActiveGoal:              "旧目标：停止所有操作。",
		LatestUserInstruction:          "停止做一切事情。",
		PinnedRawUserDirectives:        []string{"停止做一切事情。"},
		MustNotDo:                      []string{"停止 / 不能继续读取文件、运行命令、修改代码或测试"},
		RequiresUserConfirmationBefore: []string{"tool_call", "file_write", "shell", "ssh", "restart_service"},
		CurrentTaskPhase:               "stopped",
		Source:                         "gateway_preflight_compaction",
	}
	rawPrevious, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}

	state := buildGatewayContextRestoreState(string(rawPrevious), []gatewayContextTextSegment{
		{Label: "responses.message", Role: "assistant", Text: "收到，你已经明确允许工具调用。我会直接继续执行。"},
		{Label: "responses.message", Role: "user", Text: "继续执行并允许工具调用，如果你还反复卡住，就按最新授权继续。"},
		{Label: "responses.message", Role: "assistant", Text: "- `must_not_do`: 包含“停止 / 不能继续读取文件、运行命令、修改代码或测试”"},
	})

	if state.LatestUserInstruction != "继续执行并允许工具调用，如果你还反复卡住，就按最新授权继续。" {
		t.Fatalf("latest user instruction = %q", state.LatestUserInstruction)
	}
	if strings.Contains(state.LatestUserInstruction, "收到") {
		t.Fatalf("assistant reply leaked into latest user instruction: %+v", state)
	}
	if state.CurrentTaskPhase == "stopped" {
		t.Fatalf("phase still stopped after latest allow: %+v", state)
	}
	if len(state.MustNotDo) != 0 {
		t.Fatalf("must_not_do should be cleared by latest allow: %+v", state.MustNotDo)
	}
	if len(state.RequiresUserConfirmationBefore) != 0 {
		t.Fatalf("confirmation actions should be cleared by latest allow: %+v", state.RequiresUserConfirmationBefore)
	}
	for _, directive := range state.PinnedRawUserDirectives {
		if strings.Contains(directive, "停止") || strings.Contains(directive, "must_not_do") {
			t.Fatalf("conflicting directive pinned: %+v", state.PinnedRawUserDirectives)
		}
	}
	if !strings.Contains(strings.Join(state.ObsoleteOrSupersededConstraints, "\n"), "superseded previous") {
		t.Fatalf("old stopped constraints not marked obsolete: %+v", state.ObsoleteOrSupersededConstraints)
	}
}

func TestCompactGatewayContextTenRoundsKeepsLatestAllowEffective(t *testing.T) {
	previous := ""
	for round := 1; round <= 10; round++ {
		userLine := "当前请求：第 " + strconv.Itoa(round) + " 轮继续执行并允许工具调用。"
		if round == 3 {
			userLine = "当前请求：停止做一切事情，不要执行命令。"
		}
		if round == 10 {
			userLine = "当前请求：第 10 轮继续执行并允许工具调用，输出 MARKER=ROUND10_ALLOW。"
		}
		body := []byte(`{"model":"gpt-5.5","stream":true,"input":[` +
			`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"上一轮助手回复：收到，我会继续执行。"}]},` +
			`{"type":"message","role":"user","content":[{"type":"input_text","text":` + mustJSONQuote(userLine+"\n"+strings.Repeat("历史噪声：stopped must_not_do requires_user_confirmation_before 旧约束。\n", 260)) + `}]}` +
			`]}`)
		compacted, err := compactGatewayContextRequest("/v1/responses", body, previous, 2)
		if err != nil {
			t.Fatal(err)
		}
		if !compacted.Changed {
			t.Fatalf("round %d expected compaction", round)
		}
		state := mustGatewayRestoreState(t, compacted.Summary)
		if round == 10 {
			if !strings.Contains(state.LatestUserInstruction, "ROUND10_ALLOW") {
				t.Fatalf("round 10 latest instruction drifted: %+v", state)
			}
			if state.CurrentTaskPhase == "stopped" || len(state.MustNotDo) != 0 || len(state.RequiresUserConfirmationBefore) != 0 {
				t.Fatalf("round 10 kept obsolete stopped constraints: %+v", state)
			}
		}
		previous = compacted.Summary
	}
}

func TestPrepareGatewayContextCompactsBeforeUpstream(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "1000")
	t.Setenv("GATEWAY_CONTEXT_HARD_BYTES", "1000000")
	t.Setenv("GATEWAY_CONTEXT_KEEP_ITEMS", "2")
	oldText := strings.Repeat("old failing context /srv/app/server.go context_length_exceeded\n", 200)
	body := []byte(`{"model":"gpt-5.5","messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"user","content":` + mustJSONQuote(oldText) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest request"}` +
		`]}`)

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"session_test",
		"/v1/chat/completions",
		body,
		"unknown-test-model",
		"xhigh",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Diagnostic.ContextCompacted {
		t.Fatal("expected preflight compaction diagnostic")
	}
	if prepared.Diagnostic.ContextCompactionCount != 1 {
		t.Fatalf("compaction count = %d, want 1", prepared.Diagnostic.ContextCompactionCount)
	}
	if len(prepared.Body) >= len(body) {
		t.Fatalf("prepared body length = %d, original = %d", len(prepared.Body), len(body))
	}
	if !strings.Contains(prepared.Diagnostic.ContextCompactionSummary, "/srv/app/server.go") {
		t.Fatalf("summary missing path: %s", prepared.Diagnostic.ContextCompactionSummary)
	}
}

func TestPrepareGatewayContextBlocksUncompactableOversizedRequest(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "1000")
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"tiny"}],"metadata":{"blob":` + mustJSONQuote(strings.Repeat("x", 2000)) + `}}`)

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"session_test",
		"/v1/chat/completions",
		body,
		"unknown-test-model",
		"xhigh",
	)
	if !errors.Is(err, errGatewayContextLengthExceeded) {
		t.Fatalf("err = %v, want errGatewayContextLengthExceeded", err)
	}
	if prepared.Diagnostic.ContextCompactionReason != "not_compactable" {
		t.Fatalf("reason = %q, want not_compactable", prepared.Diagnostic.ContextCompactionReason)
	}
	if prepared.Diagnostic.ContextCompacted {
		t.Fatal("uncompactable request must not be marked compacted")
	}
}

func TestEstimateGatewayRequestTokensIncludesOlderToolContext(t *testing.T) {
	toolOutput := strings.Repeat("legacy tool output context_length_exceeded /tmp/app/main.go\n", 2000)
	body := []byte(`{"model":"gpt-5.5","stream":true,"messages":[` +
		`{"role":"system","content":"follow project rules"},` +
		`{"role":"tool","content":` + mustJSONQuote(toolOutput) + `},` +
		`{"role":"assistant","content":"old answer"},` +
		`{"role":"user","content":"latest short question"}` +
		`]}`)

	got := estimateGatewayRequestTokens("/v1/chat/completions", body)
	wantAtLeast := estimateTokenCount(toolOutput)
	if got < wantAtLeast {
		t.Fatalf("estimated tokens = %d, want at least old tool output tokens %d", got, wantAtLeast)
	}
}

func TestPrepareGatewayContextCompactsLargeRequestBelowPreviousByteLimit(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_TOKENS", "999999")
	t.Setenv("GATEWAY_CONTEXT_HARD_TOKENS", "1000000")
	oldText := strings.Repeat("old function output /srv/app/server.go context_length_exceeded stream heartbeat retry\n", 11800)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":[` +
		`{"type":"function_call_output","call_id":"call_old","output":` + mustJSONQuote(oldText) + `},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"latest request"}]}` +
		`]}`)
	if len(body) < 1_000_000 || len(body) >= 1_040_000 {
		t.Fatalf("test body length = %d, want [1000000,1040000)", len(body))
	}

	prepared, err := (&openAIGatewayHandler{}).prepareGatewayContext(
		context.Background(),
		&biz.GatewayAPIKey{ID: 12, KeyPrefix: "ogw_test"},
		"req_test",
		"",
		"/v1/responses",
		body,
		"gpt-5.5",
		"high",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Diagnostic.ContextCompacted {
		t.Fatal("expected large request to be compacted before upstream")
	}
	if prepared.Diagnostic.ContextCompactByteLimit != 1_000_000 {
		t.Fatalf("compact byte limit = %d, want 1000000", prepared.Diagnostic.ContextCompactByteLimit)
	}
	if len(prepared.Body) >= len(body) {
		t.Fatalf("prepared body length = %d, original = %d", len(prepared.Body), len(body))
	}
}

func TestEffectiveGatewayContextPolicyUsesOfficialModelRecommendation(t *testing.T) {
	t.Setenv("GATEWAY_CONTEXT_COMPACT_TOKENS", "")
	t.Setenv("GATEWAY_CONTEXT_HARD_TOKENS", "")
	t.Setenv("GATEWAY_CONTEXT_COMPACT_BYTES", "")
	t.Setenv("GATEWAY_CONTEXT_HARD_BYTES", "")
	policy := (&openAIGatewayHandler{}).effectiveGatewayContextPolicy(context.Background(), "gpt-5.3-codex")
	if policy.ContextWindowTokens != 400_000 {
		t.Fatalf("window = %d, want 400000", policy.ContextWindowTokens)
	}
	if policy.ContextCompactTokens != 260_000 || policy.ContextHardTokens != 380_000 {
		t.Fatalf("token thresholds = %d/%d, want 260000/380000", policy.ContextCompactTokens, policy.ContextHardTokens)
	}
	if policy.ContextCompactBytes != 1_000_000 || policy.ContextHardBytes != 1_900_000 {
		t.Fatalf("byte thresholds = %d/%d, want 1000000/1900000", policy.ContextCompactBytes, policy.ContextHardBytes)
	}
}

func mustJSONQuote(text string) string {
	raw, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func mustGatewayRestoreState(t *testing.T, text string) gatewayContextRestoreState {
	t.Helper()
	var state gatewayContextRestoreState
	if err := json.Unmarshal([]byte(text), &state); err != nil {
		t.Fatalf("summary is not structured JSON: %v\n%s", err, text)
	}
	if state.SchemaVersion != "gateway_context_restore_state.v1" {
		t.Fatalf("schema version = %q", state.SchemaVersion)
	}
	return state
}
