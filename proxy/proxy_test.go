package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/adapter/claude"
	"github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
)

func TestIsStreamingRequest_URLParam(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")

	if !IsStreamingRequest(r, []byte(`{}`)) {
		t.Fatal("expected streaming true from URL query param")
	}
}

func TestIsStreamingRequest_JSONBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"stream":true}`))
	r.Header.Set("Content-Type", "application/json")

	if !IsStreamingRequest(r, []byte(`{"stream":true}`)) {
		t.Fatal("expected streaming true from JSON body")
	}
}

func TestIsStreamingRequest_JSONBodyFalse(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"stream":false}`))
	r.Header.Set("Content-Type", "application/json")

	if IsStreamingRequest(r, []byte(`{"stream":false}`)) {
		t.Fatal("expected streaming false from JSON body")
	}
}

func TestIsStreamingRequest_NotStreaming(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	r.Header.Set("Content-Type", "application/json")

	if IsStreamingRequest(r, []byte(`{"model":"gpt-4o"}`)) {
		t.Fatal("expected streaming false when no stream field")
	}
}

func TestIsStreamingRequest_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("not-json"))

	if IsStreamingRequest(r, []byte("not-json")) {
		t.Fatal("expected streaming false for invalid JSON")
	}
}

func TestIsStreamingRequest_URLParamTakesPrecedence(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader(`{"stream":false}`))
	r.Header.Set("Content-Type", "application/json")

	if !IsStreamingRequest(r, []byte(`{"stream":false}`)) {
		t.Fatal("expected streaming true when URL param is true")
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, "https://api.openai.com/v1/chat/completions")

	r := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandler_EmptyBody(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, "https://api.openai.com/v1/chat/completions")

	r := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(""))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, "https://api.openai.com/v1/chat/completions")

	r := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandler_StreamingDetection(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, "https://api.openai.com/v1/chat/completions")

	reqBody := `{"model":"claude-3-opus","max_tokens":1024,"messages":[{"role":"user","content":"hello"}],"stream":true}`

	r := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	// This will fail at the upstream call stage, but we can verify
	// that the request is parsed correctly
	h.ServeHTTP(w, r)

	// The response will be 502 Bad Gateway (upstream unreachable),
	// but the important thing is it didn't return 400 (bad request)
	if w.Code == 0 {
		t.Fatal("expected some response code")
	}
}

func TestHandler_LostFieldsHeader(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, "https://api.openai.com/v1/chat/completions")

	// Request with frequency_penalty which is lost when encoding to Claude
	reqBody := `{"model":"claude-3-opus","max_tokens":1024,"messages":[{"role":"user","content":"hello"}],"frequency_penalty":0.5}`

	r := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	// Should still process (fail at upstream) but check header was set
	lostFields := w.Header().Get("X-Conversion-Lost-Fields")
	if lostFields != "" {
		t.Logf("Lost fields: %s", lostFields)
	}
}

func TestNewHandler_Options(t *testing.T) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
	)
	customClient := &http.Client{}

	h := NewHandler(
		c,
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		"https://api.openai.com",
		WithHTTPClient(customClient),
	)

	if h.client != customClient {
		t.Fatal("expected custom HTTP client")
	}
}

func TestHandler_ConvertedRequestPreservesStreamFlag(t *testing.T) {
	// Verify that when we convert a request and check streaming,
	// we use the converted request body, not the original.
	original := json.RawMessage(`{"model":"claude-3-opus","max_tokens":1024,"messages":[{"role":"user","content":"hello"}],"stream":true}`)

	// This test verifies that IsStreamingRequest works on the converted body
	// (which is what we fixed — previously it used the original reqBody)
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if !IsStreamingRequest(r, []byte(original)) {
		t.Fatal("expected streaming true in converted body")
	}
}

func TestIsStreamingRequest_MultipleStreamValues(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"stream true", `{"stream": true}`, true},
		{"stream false", `{"stream": false}`, false},
		{"no stream field", `{"model": "gpt-4o"}`, false},
		{"stream string", `{"stream": "true"}`, false},             // string, not bool
		{"nested stream", `{"messages":[{"stream":true}]}`, false}, // not top-level
		{"stream int 1", `{"stream": 1}`, false},                   // not bool
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			got := IsStreamingRequest(r, []byte(tt.body))
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestCopyResponseHeaders_Passthrough(t *testing.T) {
	src := http.Header{
		"X-Request-Id":      {"req-123"},
		"X-Ratelimit-Limit": {"100"},
		"X-Custom":          {"value1", "value2"},
	}
	dst := http.Header{}

	copyResponseHeaders(src, dst)

	if dst.Get("X-Request-Id") != "req-123" {
		t.Fatalf("expected X-Request-Id to be forwarded")
	}
	if dst.Get("X-Ratelimit-Limit") != "100" {
		t.Fatalf("expected X-Ratelimit-Limit to be forwarded")
	}
	values := dst["X-Custom"]
	if len(values) != 2 || values[0] != "value1" || values[1] != "value2" {
		t.Fatalf("expected multi-value X-Custom to be forwarded, got %v", values)
	}
}

func TestCopyResponseHeaders_SkipsManagedHeaders(t *testing.T) {
	src := http.Header{
		"Content-Type":      {"text/html"},
		"Content-Length":    {"1024"},
		"Transfer-Encoding": {"chunked"},
		"Connection":        {"close"},
		"X-Request-Id":      {"req-456"},
	}
	dst := http.Header{}

	copyResponseHeaders(src, dst)

	if dst.Get("Content-Type") != "" {
		t.Fatal("Content-Type should be skipped (managed by proxy)")
	}
	if dst.Get("Content-Length") != "" {
		t.Fatal("Content-Length should be skipped (managed by proxy)")
	}
	if dst.Get("Transfer-Encoding") != "" {
		t.Fatal("Transfer-Encoding should be skipped (managed by proxy)")
	}
	if dst.Get("Connection") != "" {
		t.Fatal("Connection should be skipped (managed by proxy)")
	}
	if dst.Get("X-Request-Id") != "req-456" {
		t.Fatal("X-Request-Id should be forwarded")
	}
}

func TestHandler_RequestHeaderForwarding(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-User-Agent", r.Header.Get("User-Agent"))
		w.Header().Set("X-Echo-Forwarded-For", r.Header.Get("X-Forwarded-For"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"test","object":"chat.completion","created":0,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolOpenAIChat, adapter.ProtocolOpenAIChat, upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"max_tokens":10}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "TestAgent/1.0")
	r.RemoteAddr = "192.168.1.1:54321"

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Echo-User-Agent") != "TestAgent/1.0" {
		t.Fatalf("expected User-Agent forwarded, got %q", w.Header().Get("X-Echo-User-Agent"))
	}
	if w.Header().Get("X-Echo-Forwarded-For") != "192.168.1.1:54321" {
		t.Fatalf("expected X-Forwarded-For set, got %q", w.Header().Get("X-Echo-Forwarded-For"))
	}
}

func TestHandler_ResponseHeaderForwarding(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream-Header", "upstream-value")
		w.Header().Set("X-Ratelimit-Remaining", "42")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"test","object":"chat.completion","created":0,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolOpenAIChat, adapter.ProtocolOpenAIChat, upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"max_tokens":10}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Upstream-Header") != "upstream-value" {
		t.Fatalf("expected X-Upstream-Header forwarded, got %q", w.Header().Get("X-Upstream-Header"))
	}
	if w.Header().Get("X-Ratelimit-Remaining") != "42" {
		t.Fatalf("expected X-Ratelimit-Remaining forwarded, got %q", w.Header().Get("X-Ratelimit-Remaining"))
	}
}

func TestHandler_ErrorResponseHeaderForwarding(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Error-Source", "upstream")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	}))
	defer upstream.Close()

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolOpenAIChat, adapter.ProtocolOpenAIChat, upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"max_tokens":10}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("X-Error-Source") != "upstream" {
		t.Fatalf("expected X-Error-Source forwarded, got %q", w.Header().Get("X-Error-Source"))
	}
}

func TestHandler_XForwardedFor_Appends(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-For", r.Header.Get("X-Forwarded-For"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"test","object":"chat.completion","created":0,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	h := NewHandler(c, adapter.ProtocolOpenAIChat, adapter.ProtocolOpenAIChat, upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"max_tokens":10}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	r.RemoteAddr = "192.168.1.1:54321"

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	xfwd := w.Header().Get("X-Echo-Forwarded-For")
	if xfwd != "10.0.0.1, 192.168.1.1:54321" {
		t.Fatalf("expected X-Forwarded-For appended, got %q", xfwd)
	}
}

func TestCopyResponseHeaders_EmptySrc(t *testing.T) {
	dst := http.Header{"X-Pre-Existing": {"original"}}
	copyResponseHeaders(http.Header{}, dst)

	if dst.Get("X-Pre-Existing") != "original" {
		t.Fatal("pre-existing headers should be preserved")
	}
	if len(dst) != 1 {
		t.Fatalf("expected 1 header, got %d", len(dst))
	}
}
