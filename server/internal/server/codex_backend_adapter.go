package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"server/internal/biz"
)

const (
	codexUpstreamModeCLI     = biz.GatewayUpstreamModeCodexCLI
	codexUpstreamModeBackend = biz.GatewayUpstreamModeCodexBackend

	defaultCodexBackendBaseURL = "https://chatgpt.com/backend-api/codex"
	defaultCodexBackendTimeout = 600 * time.Second
	codexOAuthClientID         = "app_EMoamEEZ73f0CkXaXp7hrann"
	defaultCodexRefreshURL     = "https://auth.openai.com/oauth/token"
	codexBackendRefreshSkew    = 5 * time.Minute
	defaultCodexBackendPrompt  = "You are a concise assistant. Follow the user's instructions exactly."
)

var defaultCodexBackendClient = &codexBackendClient{httpClient: &stdhttp.Client{}}

type codexBackendClient struct {
	mu         sync.Mutex
	httpClient *stdhttp.Client
}

type codexAuthFile struct {
	Tokens      codexAuthTokens `json:"tokens"`
	LastRefresh string          `json:"last_refresh"`
}

type codexAuthTokens struct {
	IDToken      json.RawMessage `json:"id_token"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	AccountID    string          `json:"account_id"`
}

func codexUpstreamMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CODEX_UPSTREAM_MODE"))) {
	case "":
		return codexUpstreamModeBackend
	case codexUpstreamModeCLI:
		return codexUpstreamModeCLI
	case codexUpstreamModeBackend:
		return codexUpstreamModeBackend
	default:
		return codexUpstreamModeBackend
	}
}

func (h *openAIGatewayHandler) runCodexBackend(ctx context.Context, path string, body []byte, requestModel string, reasoningEffort string) (string, openAIUsageMetrics, error) {
	return defaultCodexBackendClient.run(ctx, path, body, requestModel, reasoningEffort)
}

func (c *codexBackendClient) run(ctx context.Context, path string, body []byte, requestModel string, reasoningEffort string) (string, openAIUsageMetrics, error) {
	requestBody, model, err := codexBackendRequestFromGateway(path, body, requestModel, reasoningEffort)
	if err != nil {
		return "", openAIUsageMetrics{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := codexBackendTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	responseBody, err := c.postResponses(reqCtx, requestBody, false)
	if err != nil && isCodexBackendUnauthorized(err) {
		responseBody, err = c.postResponses(reqCtx, requestBody, true)
	}
	if reqCtx.Err() == context.DeadlineExceeded {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex backend upstream timed out after %s", timeout)
	}
	if err != nil {
		return "", openAIUsageMetrics{}, err
	}

	content, metrics, err := parseCodexBackendSSE(responseBody)
	if err != nil {
		return "", openAIUsageMetrics{}, err
	}
	if strings.TrimSpace(content) == "" {
		return "", openAIUsageMetrics{}, fmt.Errorf("codex backend upstream returned empty answer")
	}
	if metrics.TotalTokens <= 0 {
		metrics = estimateCodexCLIUsage(model, promptTextForUsageEstimate(path, body), content)
	}
	if metrics.Model == "" {
		metrics.Model = model
	}
	return content, metrics, nil
}

func (c *codexBackendClient) postResponses(ctx context.Context, body map[string]any, forceRefresh bool) ([]byte, error) {
	accessToken, accountID, err := c.codexAccessToken(ctx, forceRefresh)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, codexBackendResponsesURL(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", codexBackendUserAgent())
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode == stdhttp.StatusUnauthorized {
		return nil, codexBackendHTTPError{status: resp.StatusCode, body: responseBody}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, codexBackendHTTPError{status: resp.StatusCode, body: responseBody}
	}
	return responseBody, nil
}

func (c *codexBackendClient) codexAccessToken(ctx context.Context, forceRefresh bool) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	authPath, err := codexAuthFilePath()
	if err != nil {
		return "", "", err
	}
	auth, raw, err := loadCodexAuthFile(authPath)
	if err != nil {
		return "", "", err
	}
	accountID := strings.TrimSpace(auth.Tokens.AccountID)
	if accountID == "" {
		accountID = accountIDFromTokenData(auth.Tokens)
	}
	if strings.TrimSpace(auth.Tokens.AccessToken) == "" {
		return "", "", fmt.Errorf("codex auth file has no access token: %s", authPath)
	}
	if !forceRefresh && !jwtExpiresSoon(auth.Tokens.AccessToken, codexBackendRefreshSkew) {
		return auth.Tokens.AccessToken, accountID, nil
	}
	if strings.TrimSpace(auth.Tokens.RefreshToken) == "" {
		return "", "", fmt.Errorf("codex auth file has no refresh token: %s", authPath)
	}

	refreshed, err := requestCodexTokenRefresh(ctx, c.httpClient, auth.Tokens.RefreshToken)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(refreshed.AccessToken) != "" {
		auth.Tokens.AccessToken = refreshed.AccessToken
	}
	if strings.TrimSpace(refreshed.RefreshToken) != "" {
		auth.Tokens.RefreshToken = refreshed.RefreshToken
	}
	if len(refreshed.IDToken) > 0 {
		auth.Tokens.IDToken = refreshed.IDToken
	}
	if accountID == "" {
		accountID = accountIDFromTokenData(auth.Tokens)
	}
	if accountID != "" {
		auth.Tokens.AccountID = accountID
	}
	if err := saveCodexAuthFile(authPath, raw, auth.Tokens); err != nil {
		return "", "", err
	}
	return auth.Tokens.AccessToken, accountID, nil
}

type codexRefreshResponse struct {
	IDToken      json.RawMessage `json:"id_token"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
}

func requestCodexTokenRefresh(ctx context.Context, client *stdhttp.Client, refreshToken string) (codexRefreshResponse, error) {
	payload, _ := json.Marshal(map[string]string{
		"client_id":     codexOAuthClientID,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, codexRefreshURL(), bytes.NewReader(payload))
	if err != nil {
		return codexRefreshResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", codexBackendUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return codexRefreshResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return codexRefreshResponse{}, fmt.Errorf("codex token refresh failed: status=%d body=%s", resp.StatusCode, summarizeCodexBackendBody(body))
	}
	var refreshed codexRefreshResponse
	if err := json.Unmarshal(body, &refreshed); err != nil {
		return codexRefreshResponse{}, err
	}
	if strings.TrimSpace(refreshed.AccessToken) == "" {
		return codexRefreshResponse{}, fmt.Errorf("codex token refresh returned no access token")
	}
	return refreshed, nil
}

func codexBackendRequestFromGateway(path string, body []byte, requestModel string, reasoningEffort string) (map[string]any, string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", err
	}
	model := strings.TrimSpace(requestModel)
	if model == "" {
		model = strings.TrimSpace(stringValue(payload["model"]))
	}
	if model == "" {
		model = strings.TrimSpace(os.Getenv("CODEX_CLI_MODEL"))
	}
	if model == "" {
		model = biz.DefaultCodexModelID
	}

	instructions, input, err := codexBackendInputFromPayload(path, payload)
	if err != nil {
		return nil, "", err
	}
	if len(input) == 0 {
		return nil, "", fmt.Errorf("codex backend upstream prompt is empty")
	}

	req := map[string]any{
		"model":               model,
		"input":               input,
		"tools":               []any{},
		"tool_choice":         "auto",
		"parallel_tool_calls": false,
		"store":               false,
		"stream":              true,
		"include":             []any{},
	}
	if strings.TrimSpace(instructions) == "" {
		instructions = defaultCodexBackendPrompt
	}
	req["instructions"] = instructions
	if reasoningEffort != "" {
		req["reasoning"] = map[string]any{"effort": reasoningEffort}
	}
	return req, model, nil
}

func codexBackendInputFromPayload(path string, payload map[string]any) (string, []any, error) {
	if path == "/v1/responses" {
		if instructions := strings.TrimSpace(stringValue(payload["instructions"])); instructions != "" {
			input, err := codexBackendResponseInputItems(payload["input"])
			return instructions, input, err
		}
		input, err := codexBackendResponseInputItems(payload["input"])
		return "", input, err
	}
	return codexBackendChatInputItems(payload["messages"])
}

func codexBackendChatInputItems(value any) (string, []any, error) {
	messages, _ := value.([]any)
	instructionParts := make([]string, 0)
	input := make([]any, 0, len(messages))
	for _, item := range messages {
		message := mapValue(item)
		if message == nil {
			continue
		}
		role := strings.TrimSpace(stringValue(message["role"]))
		content := contentValue(message["content"])
		if content.empty() {
			continue
		}
		switch role {
		case "system", "developer":
			if strings.TrimSpace(content.Text) != "" {
				instructionParts = append(instructionParts, content.Text)
			}
		case "user", "assistant":
			item, err := codexBackendMessageItem(role, content)
			if err != nil {
				return "", nil, err
			}
			if item != nil {
				input = append(input, item)
			}
		}
	}
	return strings.TrimSpace(strings.Join(instructionParts, "\n\n")), input, nil
}

func codexBackendResponseInputItems(value any) ([]any, error) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		item, err := codexBackendMessageItem("user", gatewayMessageContent{Text: v})
		if err != nil {
			return nil, err
		}
		return []any{item}, nil
	case []any:
		input := make([]any, 0, len(v))
		for _, item := range v {
			message := mapValue(item)
			if message == nil {
				continue
			}
			role := strings.TrimSpace(stringValue(message["role"]))
			if role == "" {
				role = "user"
			}
			if role != "user" && role != "assistant" {
				continue
			}
			content := contentValue(message["content"])
			if content.empty() {
				continue
			}
			item, err := codexBackendMessageItem(role, content)
			if err != nil {
				return nil, err
			}
			if item != nil {
				input = append(input, item)
			}
		}
		return input, nil
	default:
		content := contentValue(value)
		if content.empty() {
			return nil, nil
		}
		item, err := codexBackendMessageItem("user", content)
		if err != nil {
			return nil, err
		}
		return []any{item}, nil
	}
}

func codexBackendMessageItem(role string, content gatewayMessageContent) (map[string]any, error) {
	normalizedFiles, err := normalizeGatewayPDFSources(content.Files)
	if err != nil {
		return nil, err
	}
	parts := make([]any, 0, 1+len(content.Images)+len(normalizedFiles))
	text := strings.TrimSpace(content.Text)
	if text != "" {
		partType := "input_text"
		if role == "assistant" {
			partType = "output_text"
		}
		parts = append(parts, map[string]any{"type": partType, "text": text})
	}
	for _, image := range content.Images {
		if strings.TrimSpace(image.Raw) == "" {
			continue
		}
		if err := validateGatewayImageSource(image); err != nil {
			return nil, err
		}
		parts = append(parts, map[string]any{
			"type":      "input_image",
			"image_url": image.Raw,
			"detail":    "high",
		})
	}
	for i, file := range normalizedFiles {
		filename := strings.TrimSpace(file.Filename)
		if filename == "" {
			filename = fmt.Sprintf("attachment-%d.pdf", i+1)
		}
		parts = append(parts, map[string]any{
			"type":      "input_file",
			"file_data": file.Raw,
			"filename":  filename,
		})
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return map[string]any{
		"type":    "message",
		"role":    role,
		"content": parts,
	}, nil
}

func parseCodexBackendSSE(body []byte) (string, openAIUsageMetrics, error) {
	content := strings.Builder{}
	metrics := openAIUsageMetrics{}
	var finalText string

	for _, data := range sseDataMessages(body) {
		if data == "[DONE]" || strings.TrimSpace(data) == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		switch stringValue(event["type"]) {
		case "response.output_text.delta":
			content.WriteString(stringValue(event["delta"]))
		case "response.output_item.done":
			if text := codexBackendOutputTextFromValue(event["item"]); text != "" {
				finalText = text
			}
		case "response.completed":
			response := mapValue(event["response"])
			if usageMetrics := codexUsageMetricsFromMap(mapValue(response["usage"])); usageMetrics.TotalTokens > 0 {
				metrics = usageMetrics
			}
			if text := codexBackendOutputTextFromValue(response); text != "" {
				finalText = text
			}
		case "response.failed":
			return "", openAIUsageMetrics{}, fmt.Errorf("codex backend response failed: %s", summarizeCodexBackendBody([]byte(data)))
		case "response.incomplete":
			return "", openAIUsageMetrics{}, fmt.Errorf("codex backend response incomplete: %s", summarizeCodexBackendBody([]byte(data)))
		}
	}

	text := strings.TrimSpace(content.String())
	if strings.TrimSpace(finalText) != "" {
		text = strings.TrimSpace(finalText)
	}
	return text, metrics, nil
}

func sseDataMessages(body []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxGatewayRequestBytes)
	messages := make([]string, 0)
	current := strings.Builder{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current.Len() > 0 {
				messages = append(messages, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if current.Len() > 0 {
				current.WriteByte('\n')
			}
			current.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if current.Len() > 0 {
		messages = append(messages, strings.TrimSpace(current.String()))
	}
	return messages
}

func codexBackendOutputTextFromValue(value any) string {
	switch v := value.(type) {
	case map[string]any:
		if outputText := stringValue(v["output_text"]); outputText != "" {
			return outputText
		}
		if v["type"] == "message" {
			return outputTextFromCodexContent(v["content"])
		}
		if output, ok := v["output"].([]any); ok {
			parts := make([]string, 0, len(output))
			for _, item := range output {
				if text := codexBackendOutputTextFromValue(item); text != "" {
					parts = append(parts, text)
				}
			}
			return strings.TrimSpace(strings.Join(parts, "\n"))
		}
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := codexBackendOutputTextFromValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	}
	return ""
}

func loadCodexAuthFile(path string) (codexAuthFile, map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return codexAuthFile{}, nil, err
	}
	var auth codexAuthFile
	if err := json.Unmarshal(body, &auth); err != nil {
		return codexAuthFile{}, nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return codexAuthFile{}, nil, err
	}
	return auth, raw, nil
}

func saveCodexAuthFile(path string, raw map[string]any, tokens codexAuthTokens) error {
	if raw == nil {
		raw = map[string]any{}
	}
	tokenMap := mapValue(raw["tokens"])
	if tokenMap == nil {
		tokenMap = map[string]any{}
		raw["tokens"] = tokenMap
	}
	tokenMap["access_token"] = tokens.AccessToken
	tokenMap["refresh_token"] = tokens.RefreshToken
	if tokens.AccountID != "" {
		tokenMap["account_id"] = tokens.AccountID
	}
	if len(tokens.IDToken) > 0 {
		var idToken any
		if err := json.Unmarshal(tokens.IDToken, &idToken); err == nil {
			tokenMap["id_token"] = idToken
		}
	}
	raw["last_refresh"] = time.Now().UTC().Format(time.RFC3339Nano)

	encoded, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o600)
}

func codexAuthFilePath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("CODEX_AUTH_FILE")); path != "" {
		return path, nil
	}
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "auth.json"), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".codex", "auth.json"), nil
}

func accountIDFromTokenData(tokens codexAuthTokens) string {
	if acc := accountIDFromJWT(tokens.AccessToken); acc != "" {
		return acc
	}
	if len(tokens.IDToken) == 0 {
		return ""
	}
	var idTokenString string
	if err := json.Unmarshal(tokens.IDToken, &idTokenString); err == nil {
		return accountIDFromJWT(idTokenString)
	}
	var claims map[string]any
	if err := json.Unmarshal(tokens.IDToken, &claims); err == nil {
		return stringValue(claims["chatgpt_account_id"])
	}
	return ""
}

func accountIDFromJWT(token string) string {
	claims := jwtClaims(token)
	return stringValue(claims["chatgpt_account_id"])
}

func jwtExpiresSoon(token string, skew time.Duration) bool {
	exp := int64Value(jwtClaims(token)["exp"])
	if exp <= 0 {
		return false
	}
	return time.Until(time.Unix(exp, 0)) <= skew
}

func jwtClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

func codexBackendResponsesURL() string {
	return strings.TrimRight(codexBackendBaseURL(), "/") + "/responses"
}

func codexBackendBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("CODEX_BACKEND_BASE_URL")); raw != "" {
		return raw
	}
	return defaultCodexBackendBaseURL
}

func codexRefreshURL() string {
	if raw := strings.TrimSpace(os.Getenv("CODEX_REFRESH_TOKEN_URL_OVERRIDE")); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(os.Getenv("CODEX_BACKEND_REFRESH_URL")); raw != "" {
		return raw
	}
	return defaultCodexRefreshURL
}

func codexBackendTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CODEX_BACKEND_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultCodexBackendTimeout
}

func codexBackendUserAgent() string {
	if raw := strings.TrimSpace(os.Getenv("CODEX_BACKEND_USER_AGENT")); raw != "" {
		return raw
	}
	return "codex-cli"
}

func promptTextForUsageEstimate(path string, body []byte) string {
	text, err := promptFromGatewayRequest(path, body)
	if err != nil {
		return ""
	}
	return text
}

type codexBackendHTTPError struct {
	status int
	body   []byte
}

func (e codexBackendHTTPError) Error() string {
	return fmt.Sprintf("codex backend upstream failed: status=%d body=%s", e.status, summarizeCodexBackendBody(e.body))
}

func isCodexBackendUnauthorized(err error) bool {
	httpErr, ok := err.(codexBackendHTTPError)
	return ok && httpErr.status == stdhttp.StatusUnauthorized
}

func summarizeCodexBackendBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	const maxLen = 800
	if len(text) > maxLen {
		text = text[:maxLen]
	}
	return text
}
