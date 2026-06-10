package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/adapter/claude"
	"github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
	"github.com/skys-mission/lmm-adapter-go/adapter/openairesp"
	"github.com/skys-mission/lmm-adapter-go/stream"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

// =============================================================================
// Test Configuration
// =============================================================================

type testConfig struct {
	// OpenAI Responses API (gpt-5-nano)
	openaiRespURL   string
	openaiRespKey   string
	openaiRespModel string

	// OpenAI Chat Completions API (kimi-k2.6)
	openaiChatURL   string
	openaiChatKey   string
	openaiChatModel string

	// Claude Messages API (GLM-5V-Turbo)
	claudeURL   string
	claudeKey   string
	claudeModel string
}

// getConfig reads integration test configuration from environment variables.
// Default URLs and models are examples — override them for your providers:
//
//	OPENAI_RESP_URL    – OpenAI Responses-compatible endpoint
//	OPENAI_CHAT_URL    – OpenAI Chat Completions-compatible endpoint
//	CLAUDE_URL         – Claude Messages-compatible endpoint
//	OPENAI_RESP_KEY / OPENAI_CHAT_KEY / CLAUDE_KEY – API keys (required)
//	OPENAI_RESP_MODEL / OPENAI_CHAT_MODEL / CLAUDE_MODEL – model names
func getConfig() testConfig {
	return testConfig{
		openaiRespURL:   getEnv("OPENAI_RESP_URL", "https://opencode.ai/zen/v1/responses"),
		openaiRespKey:   getEnv("OPENAI_RESP_KEY", ""),
		openaiRespModel: getEnv("OPENAI_RESP_MODEL", "gpt-5-nano"),

		openaiChatURL:   getEnv("OPENAI_CHAT_URL", "https://opencode.ai/zen/go/v1/chat/completions"),
		openaiChatKey:   getEnv("OPENAI_CHAT_KEY", ""),
		openaiChatModel: getEnv("OPENAI_CHAT_MODEL", "kimi-k2.6"),

		claudeURL:   getEnv("CLAUDE_URL", "https://open.bigmodel.cn/api/anthropic"),
		claudeKey:   getEnv("CLAUDE_KEY", ""),
		claudeModel: getEnv("CLAUDE_MODEL", "GLM-5V-Turbo"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func skipIfNoKey(t *testing.T, key, name string) {
	t.Helper()
	if key == "" {
		t.Skipf("Skipping %s test: no API key provided", name)
	}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// =============================================================================
// Helper: log unified response
// =============================================================================

func logUniResponse(t *testing.T, prefix string, resp *uni.Response, report *adapter.Report) {
	t.Helper()
	t.Logf("%s: ID=%s Model=%s StopReason=%s Messages=%d LostFields=%d",
		prefix, resp.ID, resp.Model, resp.StopReason, len(resp.Messages), len(report.LostFields))
	for i, msg := range resp.Messages {
		t.Logf("%s: Message[%d] Role=%s ContentParts=%d", prefix, i, msg.Role, len(msg.Content))
		for j, part := range msg.Content {
			switch p := part.(type) {
			case uni.TextPart:
				t.Logf("%s:   Part[%d] Text=%q", prefix, j, truncateStr(p.Text, 200))
			case uni.ImagePart:
				t.Logf("%s:   Part[%d] Image url=%s", prefix, j, p.URL)
			case uni.ThinkingPart:
				t.Logf("%s:   Part[%d] Thinking=%q", prefix, j, truncateStr(p.Thinking, 100))
			case uni.ToolUsePart:
				t.Logf("%s:   Part[%d] ToolUse tool=%s", prefix, j, p.ToolName)
			default:
				t.Logf("%s:   Part[%d] Type=%s", prefix, j, part.ContentType())
			}
		}
	}
	for _, lf := range report.LostFields {
		t.Logf("%s:   LostField: %s.%s - %s", prefix, lf.Source, lf.Field, lf.Reason)
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// =============================================================================
// Section 1: Direct native-protocol tests (baseline)
// =============================================================================

func TestNative_OpenAIResponses(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	a := openairesp.New()

	req := map[string]any{
		"model": cfg.openaiRespModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "用一句话介绍北京"},
				},
			},
		},
		"max_output_tokens": 1000,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "NativeOpenAIResp", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message in response")
	}

	// Verify the response has text content
	hasText := false
	for _, msg := range uniResp.Messages {
		for _, part := range msg.Content {
			if tp, ok := part.(uni.TextPart); ok && tp.Text != "" {
				hasText = true
			}
		}
	}
	if !hasText {
		t.Fatal("Expected non-empty text content in response")
	}
}

func TestNative_OpenAIChat(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	a := openaichat.New()

	req := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "用一句话介绍上海"},
		},
		"max_tokens": 200,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "NativeOpenAIChat", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message in response")
	}
}

func TestNative_ClaudeMessages(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	a := claude.New()

	req := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{"role": "user", "content": "用一句话介绍深圳"},
		},
		"max_tokens": 200,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(reqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "NativeClaude", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message in response")
	}
}

// =============================================================================
// Section 2: Cross-protocol request-response conversion (6 combinations)
// =============================================================================

// 2a. Claude → OpenAI Chat

func TestCrossProtocol_ClaudeToOpenAIChat_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)

	// Step 1: Build a Claude-format request
	claudeReq := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "What is the capital of France? Answer in one short sentence."},
		},
		"max_tokens": 500,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	// Step 2: Convert Claude → OpenAI Chat
	openaiReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (Claude→OpenAIChat) failed: %v", err)
	}
	t.Logf("Claude→OpenAIChat request: %s", string(openaiReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Step 3: Call the real OpenAI Chat API
	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Step 4: Convert OpenAI Chat response → Claude
	claudeRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (OpenAIChat→Claude) failed: %v", err)
	}
	t.Logf("OpenAIChat→Claude response: %s", string(claudeRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Step 5: Verify the final Claude response structure
	var claudeResp map[string]any
	if err := json.Unmarshal(claudeRespJSON, &claudeResp); err != nil {
		t.Fatalf("Failed to unmarshal final Claude response: %v", err)
	}

	if claudeResp["type"] != "message" {
		t.Errorf("Expected type=message, got %v", claudeResp["type"])
	}
	if claudeResp["role"] != "assistant" {
		t.Errorf("Expected role=assistant, got %v", claudeResp["role"])
	}

	content, ok := claudeResp["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected non-empty content array in Claude response")
	}

	firstBlock := content[0].(map[string]any)
	if firstBlock["type"] == "text" {
		t.Logf("✓ Cross-protocol response text: %v", firstBlock["text"])
	}
}

// 2b. Claude → OpenAI Responses

func TestCrossProtocol_ClaudeToOpenAIResponses_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openairesp.New()),
	)

	claudeReq := map[string]any{
		"model": cfg.openaiRespModel,
		"messages": []map[string]any{
			{"role": "user", "content": "What is 2+2? Answer in one word."},
		},
		"max_tokens": 500,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	// Convert Claude → OpenAI Responses
	openaiReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIResponses,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (Claude→OpenAIResp) failed: %v", err)
	}
	t.Logf("Claude→OpenAIResp request: %s", string(openaiReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Call OpenAI Responses API
	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert OpenAI Responses → Claude
	claudeRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolOpenAIResponses,
		adapter.ProtocolClaudeMessages,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (OpenAIResp→Claude) failed: %v", err)
	}
	t.Logf("OpenAIResp→Claude response: %s", string(claudeRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	var claudeResp map[string]any
	if err := json.Unmarshal(claudeRespJSON, &claudeResp); err != nil {
		t.Fatalf("Failed to unmarshal final Claude response: %v", err)
	}

	if claudeResp["type"] != "message" {
		t.Errorf("Expected type=message, got %v", claudeResp["type"])
	}

	content, ok := claudeResp["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected non-empty content array in Claude response")
	}

	firstBlock := content[0].(map[string]any)
	if firstBlock["type"] == "text" {
		t.Logf("✓ Cross-protocol response text: %v", firstBlock["text"])
	}
}

// 2c. OpenAI Chat → Claude

func TestCrossProtocol_OpenAIChatToClaude_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(claude.New()),
	)

	openaiReq := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Name the largest planet in our solar system."},
		},
		"max_tokens": 500,
	}
	openaiReqJSON, _ := json.Marshal(openaiReq)

	// Convert OpenAI Chat → Claude
	claudeReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		openaiReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (OpenAIChat→Claude) failed: %v", err)
	}
	t.Logf("OpenAIChat→Claude request: %s", string(claudeReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Call Claude API
	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(claudeReqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert Claude → OpenAI Chat
	openaiRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (Claude→OpenAIChat) failed: %v", err)
	}
	t.Logf("Claude→OpenAIChat response: %s", string(openaiRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	var openaiResp map[string]any
	if err := json.Unmarshal(openaiRespJSON, &openaiResp); err != nil {
		t.Fatalf("Failed to unmarshal final OpenAI Chat response: %v", err)
	}

	choices, ok := openaiResp["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatal("Expected non-empty choices in OpenAI Chat response")
	}

	choice := choices[0].(map[string]any)
	msg := choice["message"].(map[string]any)
	t.Logf("✓ Cross-protocol response: role=%v content=%v", msg["role"], truncateStr(msg["content"].(string), 200))
}

// 2d. OpenAI Chat → OpenAI Responses

func TestCrossProtocol_OpenAIChatToOpenAIResponses_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openairesp.New()),
	)

	openaiReq := map[string]any{
		"model": cfg.openaiRespModel,
		"messages": []map[string]any{
			{"role": "user", "content": "What color is the sky? Answer briefly."},
		},
		"max_tokens": 500,
	}
	openaiReqJSON, _ := json.Marshal(openaiReq)

	// Convert OpenAI Chat → OpenAI Responses
	respReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolOpenAIResponses,
		openaiReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (OpenAIChat→OpenAIResp) failed: %v", err)
	}
	t.Logf("OpenAIChat→OpenAIResp request: %s", string(respReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Call OpenAI Responses API
	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(respReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert OpenAI Responses → OpenAI Chat
	openaiRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolOpenAIResponses,
		adapter.ProtocolOpenAIChat,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (OpenAIResp→OpenAIChat) failed: %v", err)
	}
	t.Logf("OpenAIResp→OpenAIChat response: %s", string(openaiRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	var result map[string]any
	if err := json.Unmarshal(openaiRespJSON, &result); err != nil {
		t.Fatalf("Failed to unmarshal final response: %v", err)
	}

	choices, ok := result["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatal("Expected non-empty choices")
	}
	t.Logf("✓ Cross-protocol response received with %d choices", len(choices))
}

// 2e. OpenAI Responses → Claude

func TestCrossProtocol_OpenAIResponsesToClaude_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	c := adapter.NewConverter(
		adapter.WithAdapter(openairesp.New()),
		adapter.WithAdapter(claude.New()),
	)

	respReq := map[string]any{
		"model": cfg.claudeModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Say 'hello' in French."},
				},
			},
		},
		"max_output_tokens": 500,
	}
	respReqJSON, _ := json.Marshal(respReq)

	// Convert OpenAI Responses → Claude
	claudeReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolOpenAIResponses,
		adapter.ProtocolClaudeMessages,
		respReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (OpenAIResp→Claude) failed: %v", err)
	}
	t.Logf("OpenAIResp→Claude request: %s", string(claudeReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Call Claude API
	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(claudeReqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert Claude → OpenAI Responses
	finalRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIResponses,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (Claude→OpenAIResp) failed: %v", err)
	}
	t.Logf("Claude→OpenAIResp response: %s", string(finalRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	var result map[string]any
	if err := json.Unmarshal(finalRespJSON, &result); err != nil {
		t.Fatalf("Failed to unmarshal final response: %v", err)
	}

	output, ok := result["output"].([]any)
	if !ok || len(output) == 0 {
		t.Fatal("Expected non-empty output in OpenAI Responses response")
	}
	t.Logf("✓ Cross-protocol response received with %d output items", len(output))
}

// 2f. OpenAI Responses → OpenAI Chat

func TestCrossProtocol_OpenAIResponsesToOpenAIChat_RoundTrip(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	c := adapter.NewConverter(
		adapter.WithAdapter(openairesp.New()),
		adapter.WithAdapter(openaichat.New()),
	)

	respReq := map[string]any{
		"model": cfg.openaiChatModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "What is the speed of light?"},
				},
			},
		},
		"max_output_tokens": 500,
	}
	respReqJSON, _ := json.Marshal(respReq)

	// Convert OpenAI Responses → OpenAI Chat
	chatReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolOpenAIResponses,
		adapter.ProtocolOpenAIChat,
		respReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest (OpenAIResp→OpenAIChat) failed: %v", err)
	}
	t.Logf("OpenAIResp→OpenAIChat request: %s", string(chatReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Call OpenAI Chat API
	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(chatReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert OpenAI Chat → OpenAI Responses
	finalRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolOpenAIResponses,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse (OpenAIChat→OpenAIResp) failed: %v", err)
	}
	t.Logf("OpenAIChat→OpenAIResp response: %s", string(finalRespJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	var result map[string]any
	if err := json.Unmarshal(finalRespJSON, &result); err != nil {
		t.Fatalf("Failed to unmarshal final response: %v", err)
	}

	if _, ok := result["output"]; !ok {
		t.Fatal("Expected 'output' field in OpenAI Responses response")
	}
	t.Logf("✓ Cross-protocol conversion successful")
}

// =============================================================================
// Section 3: Multimodal tests (text + image)
// =============================================================================

func TestMultimodal_OpenAIResponses_ImageURL(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	a := openairesp.New()

	req := map[string]any{
		"model": cfg.openaiRespModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Describe this image briefly:"},
					{"type": "input_image", "image_url": "data:image/png;base64,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"},
				},
			},
		},
		"max_output_tokens": 500,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "MultimodalOpenAIResp", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message")
	}

	hasText := false
	for _, msg := range uniResp.Messages {
		for _, part := range msg.Content {
			if tp, ok := part.(uni.TextPart); ok && tp.Text != "" {
				hasText = true
			}
		}
	}
	if !hasText {
		t.Fatal("Expected text response for image description")
	}
}

func TestMultimodal_OpenAIChat_ImageURL(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	a := openaichat.New()

	req := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Describe this image:"},
					{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64,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"}},
				},
			},
		},
		"max_tokens": 200,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "MultimodalOpenAIChat", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestMultimodal_ClaudeMessages_ImageURL(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	a := claude.New()

	req := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Describe this image briefly:"},
					{"type": "image", "source": map[string]any{
						"type": "url",
						"url":  "data:image/png;base64,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",
					}},
				},
			},
		},
		"max_tokens": 200,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(reqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, report, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	logUniResponse(t, "MultimodalClaude", uniResp, report)

	if len(uniResp.Messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

// =============================================================================
// Section 4: Cross-protocol multimodal (image in one format, convert to another)
// =============================================================================

func TestCrossProtocol_Multimodal_ClaudeImageToOpenAIChat(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)

	// Claude-format request with image URL
	claudeReq := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "What do you see in this image?"},
					{"type": "image", "source": map[string]any{
						"type": "url",
						"url":  "data:image/png;base64,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",
					}},
				},
			},
		},
		"max_tokens": 200,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	// Convert Claude → OpenAI Chat
	openaiReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}
	t.Logf("Multimodal Claude→OpenAIChat request: %s", string(openaiReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Verify image URL is preserved in the conversion
	if !bytes.Contains(openaiReqJSON, []byte("image_url")) {
		t.Fatal("Expected image_url in converted request")
	}

	// Call OpenAI Chat API
	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert response back
	claudeRespJSON, report, err := c.ConvertResponse(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse failed: %v", err)
	}
	t.Logf("Multimodal response: %s", string(claudeRespJSON))

	var claudeResp map[string]any
	json.Unmarshal(claudeRespJSON, &claudeResp)
	t.Logf("✓ Cross-protocol multimodal conversion successful")
}

func TestCrossProtocol_Multimodal_OpenAIChatImageToClaude(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(claude.New()),
	)

	// OpenAI Chat request with image URL
	openaiReq := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "What do you see?"},
					{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64,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"}},
				},
			},
		},
		"max_tokens": 200,
	}
	openaiReqJSON, _ := json.Marshal(openaiReq)

	// Convert OpenAI Chat → Claude
	claudeReqJSON, _, err := c.ConvertRequest(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		openaiReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}
	t.Logf("Multimodal OpenAIChat→Claude request: %s", string(claudeReqJSON))

	// Verify image is converted to Claude format
	if !bytes.Contains(claudeReqJSON, []byte("image")) {
		t.Fatal("Expected image in converted Claude request")
	}

	// Call Claude API
	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(claudeReqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	t.Logf("✓ Cross-protocol multimodal (OpenAIChat→Claude) conversion successful")
	t.Logf("Response: %s", truncateStr(string(respBody), 300))
}

// =============================================================================
// Section 5: Streaming tests
// =============================================================================

func TestStreaming_OpenAIResponses_Native(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	req := map[string]any{
		"model": cfg.openaiRespModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Count from 1 to 5."},
				},
			},
		},
		"max_output_tokens": 500,
		"stream":            true,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := stream.NewSSEReader(resp.Body)
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}
		eventCount++

		uniEvent, _, err := openairesp.New().DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)
	}

	t.Logf("Total SSE events: %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}

	hasText := false
	for _, msg := range finalResp.Messages {
		for _, part := range msg.Content {
			if tp, ok := part.(uni.TextPart); ok && len(tp.Text) > 0 {
				hasText = true
				t.Logf("Accumulated text: %s", tp.Text)
			}
		}
	}
	if !hasText {
		t.Fatal("Expected text content in accumulated streaming response")
	}
}

func TestStreaming_OpenAIChat_Native(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	req := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Count from 1 to 3."},
		},
		"max_tokens": 200,
		"stream":     true,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := stream.NewSSEReader(resp.Body)
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}

		// Skip [DONE] sentinel
		if strings.TrimSpace(event.Data) == "[DONE]" {
			continue
		}

		eventCount++

		uniEvent, _, err := openaichat.New().DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)
	}

	t.Logf("Total SSE events: %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}
}

func TestStreaming_ClaudeMessages_Native(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	req := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Count from 1 to 3."},
		},
		"max_tokens": 200,
		"stream":     true,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(reqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := stream.NewSSEReader(resp.Body)
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}
		eventCount++

		uniEvent, _, err := claude.New().DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)
	}

	t.Logf("Total SSE events: %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}
}

// =============================================================================
// Section 6: Cross-protocol streaming (pipeline)
// =============================================================================

func TestStreaming_CrossProtocol_ClaudeRequestToOpenAIChatAPI_ToClaudeStream(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)

	// Build Claude request, convert to OpenAI Chat
	claudeReq := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "List 3 programming languages."},
		},
		"max_tokens": 500,
		"stream":     true,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	openaiReqJSON, _, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}
	t.Logf("Streaming Claude→OpenAIChat request: %s", string(openaiReqJSON))

	// Call OpenAI Chat API with streaming
	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Convert streaming response back to Claude format
	reader := stream.NewSSEReader(resp.Body)
	srcAdapter := openaichat.New()
	dstAdapter := claude.New()
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}
		if strings.TrimSpace(event.Data) == "[DONE]" {
			continue
		}

		// Decode → unified → Encode
		uniEvent, _, err := srcAdapter.DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)

		claudeEvent, _, err := dstAdapter.EncodeStreamEvent(uniEvent)
		if err != nil {
			t.Logf("EncodeStreamEvent warning: %v", err)
			continue
		}
		eventCount++
		_ = claudeEvent
	}

	t.Logf("Cross-protocol streaming events: %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}
	if eventCount == 0 {
		t.Fatal("Expected at least one streaming event")
	}

	// Verify text content
	for _, msg := range finalResp.Messages {
		for _, part := range msg.Content {
			if tp, ok := part.(uni.TextPart); ok {
				t.Logf("Streaming cross-protocol text: %s", tp.Text)
			}
		}
	}
}

func TestStreaming_CrossProtocol_ClaudeRequestToOpenAIResponsesAPI_ToClaudeStream(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openairesp.New()),
	)

	claudeReq := map[string]any{
		"model": cfg.openaiRespModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Name 3 colors."},
		},
		"max_tokens": 500,
		"stream":     true,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	openaiReqJSON, _, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIResponses,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}

	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := stream.NewSSEReader(resp.Body)
	srcAdapter := openairesp.New()
	dstAdapter := claude.New()
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}

		uniEvent, _, err := srcAdapter.DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)

		_, _, err = dstAdapter.EncodeStreamEvent(uniEvent)
		if err != nil {
			t.Logf("EncodeStreamEvent warning: %v", err)
			continue
		}
		eventCount++
	}

	t.Logf("Cross-protocol streaming events: %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}

	for _, msg := range finalResp.Messages {
		for _, part := range msg.Content {
			if tp, ok := part.(uni.TextPart); ok {
				t.Logf("Streaming cross-protocol text: %s", tp.Text)
			}
		}
	}
}

func TestStreaming_CrossProtocol_OpenAIChatRequestToClaudeAPI_ToOpenAIChatStream(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(claude.New()),
	)

	openaiReq := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Name 3 fruits."},
		},
		"max_tokens": 500,
		"stream":     true,
	}
	openaiReqJSON, _ := json.Marshal(openaiReq)

	claudeReqJSON, _, err := c.ConvertRequest(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		openaiReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}

	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(claudeReqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := stream.NewSSEReader(resp.Body)
	srcAdapter := claude.New()
	dstAdapter := openaichat.New()
	acc := stream.NewAccumulator()
	eventCount := 0

	for {
		event, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read SSE event failed: %v", err)
		}
		if event.Data == "" {
			continue
		}

		uniEvent, _, err := srcAdapter.DecodeStreamEvent(json.RawMessage(event.Data))
		if err != nil {
			t.Logf("DecodeStreamEvent warning: %v", err)
			continue
		}
		acc.Accumulate(uniEvent)

		_, _, err = dstAdapter.EncodeStreamEvent(uniEvent)
		if err != nil {
			t.Logf("EncodeStreamEvent warning: %v", err)
			continue
		}
		eventCount++
	}

	t.Logf("Cross-protocol streaming events (Claude→OpenAIChat): %d", eventCount)
	finalResp := acc.Response()
	t.Logf("Accumulated: ID=%s Model=%s Messages=%d StopReason=%s",
		finalResp.ID, finalResp.Model, len(finalResp.Messages), finalResp.StopReason)

	if len(finalResp.Messages) == 0 {
		t.Fatal("Expected at least one accumulated message")
	}
	if eventCount == 0 {
		t.Fatal("Expected at least one streaming event")
	}
}

// =============================================================================
// Section 7: System message handling
// =============================================================================

func TestSystemMessage_ClaudeToOpenAIChat(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)

	// Claude uses top-level "system" field
	claudeReq := map[string]any{
		"model": cfg.openaiChatModel,
		"system": []map[string]any{
			{"type": "text", "text": "You are a helpful assistant that speaks like a pirate."},
		},
		"messages": []map[string]any{
			{"role": "user", "content": "Hello! Who are you?"},
		},
		"max_tokens": 500,
	}
	claudeReqJSON, _ := json.Marshal(claudeReq)

	// Convert Claude → OpenAI Chat
	openaiReqJSON, report, err := c.ConvertRequest(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		claudeReqJSON,
	)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}
	t.Logf("System msg Claude→OpenAIChat request: %s", string(openaiReqJSON))
	t.Logf("LostFields: %d", len(report.LostFields))

	// Verify system message is in messages array with system role
	if !bytes.Contains(openaiReqJSON, []byte(`"role":"system"`)) {
		t.Error("Expected system role in converted OpenAI Chat messages")
	}

	// Call OpenAI Chat API
	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(openaiReqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert back
	claudeRespJSON, _, err := c.ConvertResponse(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		respBody,
	)
	if err != nil {
		t.Fatalf("ConvertResponse failed: %v", err)
	}

	var claudeResp map[string]any
	json.Unmarshal(claudeRespJSON, &claudeResp)

	content, ok := claudeResp["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("Expected content in response")
	}
	t.Logf("✓ System message round-trip successful")
}

// =============================================================================
// Section 8: Error handling
// =============================================================================

func TestErrorHandling_InvalidAPIKey(t *testing.T) {
	// Use an intentionally invalid key
	req := map[string]any{
		"model": "gpt-5-nano",
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Hello"},
				},
			},
		},
	}
	reqJSON, _ := json.Marshal(req)

	cfg := getConfig()
	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer invalid-key-12345")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Should receive an error response
	if resp.StatusCode == 200 {
		t.Log("Warning: Invalid API key was accepted by the server")
	} else {
		t.Logf("Got expected error status %d: %s", resp.StatusCode, truncateStr(string(body), 300))

		// Test error decoding
		_, _, err = openairesp.New().DecodeError(body, resp.StatusCode)
		if err != nil {
			t.Logf("DecodeError (expected for non-openai format): %v", err)
		}
	}
}

// =============================================================================
// Section 9: Token estimation verification
// =============================================================================

func TestTokenEstimation_AgainstRealUsage(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	a := openaichat.New()

	req := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Hello, how are you?"},
		},
		"max_tokens": 500,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	uniResp, _, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}

	t.Logf("Actual usage: input=%d output=%d total=%d",
		uniResp.Usage.InputTokens, uniResp.Usage.OutputTokens, uniResp.Usage.TotalTokens)

	if uniResp.Usage.TotalTokens <= 0 {
		t.Fatal("Expected positive token usage from API")
	}

	// Basic sanity: input + output should be > 0
	if uniResp.Usage.InputTokens+uniResp.Usage.OutputTokens <= 0 {
		t.Fatal("Expected positive token counts")
	}
}

// =============================================================================
// Section 10: Round-trip decode→encode→decode for all three protocols
// =============================================================================

func TestRoundTripDecodeEncodeDecode_OpenAIResponses(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiRespKey, "OpenAI Responses")

	a := openairesp.New()

	req := map[string]any{
		"model": cfg.openaiRespModel,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Say hello."},
				},
			},
		},
		"max_output_tokens": 500,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiRespURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiRespKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Round-trip: Decode → Encode → Decode
	uni1, _, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("First decode failed: %v", err)
	}
	encoded, _, err := a.EncodeResponse(uni1)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	uni2, _, err := a.DecodeResponse(encoded)
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}

	// Verify key fields preserved
	if uni1.StopReason != uni2.StopReason {
		t.Errorf("StopReason mismatch: %q vs %q", uni1.StopReason, uni2.StopReason)
	}
	if len(uni1.Messages) != len(uni2.Messages) {
		t.Errorf("Message count mismatch: %d vs %d", len(uni1.Messages), len(uni2.Messages))
	}
	t.Logf("✓ OpenAI Responses round-trip (decode→encode→decode) successful")
}

func TestRoundTripDecodeEncodeDecode_OpenAIChat(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.openaiChatKey, "OpenAI Chat")

	a := openaichat.New()

	req := map[string]any{
		"model": cfg.openaiChatModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Say hello."},
		},
		"max_tokens": 500,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.openaiChatURL, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.openaiChatKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Round-trip: Decode → Encode → Decode
	uni1, _, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("First decode failed: %v", err)
	}
	encoded, _, err := a.EncodeResponse(uni1)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	uni2, _, err := a.DecodeResponse(encoded)
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}

	if uni1.StopReason != uni2.StopReason {
		t.Errorf("StopReason mismatch: %q vs %q", uni1.StopReason, uni2.StopReason)
	}
	if len(uni1.Messages) != len(uni2.Messages) {
		t.Errorf("Message count mismatch: %d vs %d", len(uni1.Messages), len(uni2.Messages))
	}
	t.Logf("✓ OpenAI Chat round-trip (decode→encode→decode) successful")
}

func TestRoundTripDecodeEncodeDecode_Claude(t *testing.T) {
	cfg := getConfig()
	skipIfNoKey(t, cfg.claudeKey, "Claude Messages")

	a := claude.New()

	req := map[string]any{
		"model": cfg.claudeModel,
		"messages": []map[string]any{
			{"role": "user", "content": "Say hello."},
		},
		"max_tokens": 500,
	}
	reqJSON, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", cfg.claudeURL+"/v1/messages", bytes.NewReader(reqJSON))
	httpReq.Header.Set("x-api-key", cfg.claudeKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient().Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Round-trip: Decode → Encode → Decode
	uni1, _, err := a.DecodeResponse(body)
	if err != nil {
		t.Fatalf("First decode failed: %v", err)
	}
	encoded, _, err := a.EncodeResponse(uni1)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	uni2, _, err := a.DecodeResponse(encoded)
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}

	if uni1.StopReason != uni2.StopReason {
		t.Errorf("StopReason mismatch: %q vs %q", uni1.StopReason, uni2.StopReason)
	}
	if len(uni1.Messages) != len(uni2.Messages) {
		t.Errorf("Message count mismatch: %d vs %d", len(uni1.Messages), len(uni2.Messages))
	}
	t.Logf("✓ Claude Messages round-trip (decode→encode→decode) successful")
}
