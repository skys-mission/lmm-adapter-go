package header

import (
	"net/http"
	"testing"
)

func TestPolicyPassthrough(t *testing.T) {
	policy := NewPolicy().
		WithPassthrough("User-Agent", "X-Request-ID")

	src := http.Header{}
	src.Set("User-Agent", "TestClient/1.0")
	src.Set("X-Request-ID", "req-123")
	src.Set("Cookie", "session=abc")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("User-Agent") != "TestClient/1.0" {
		t.Errorf("expected User-Agent to pass through, got %q", dst.Get("User-Agent"))
	}
	if dst.Get("X-Request-ID") != "req-123" {
		t.Errorf("expected X-Request-ID to pass through, got %q", dst.Get("X-Request-ID"))
	}
	if dst.Get("Cookie") != "" {
		t.Errorf("expected Cookie to be stripped, got %q", dst.Get("Cookie"))
	}
}

func TestPolicyMapping(t *testing.T) {
	policy := NewPolicy().
		WithMapping("Authorization", "x-api-key")

	src := http.Header{}
	src.Set("Authorization", "Bearer token123")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("x-api-key") != "Bearer token123" {
		t.Errorf("expected x-api-key to be set, got %q", dst.Get("x-api-key"))
	}
	if dst.Get("Authorization") != "" {
		t.Errorf("expected Authorization to be removed, got %q", dst.Get("Authorization"))
	}
}

func TestPolicyStrip(t *testing.T) {
	policy := NewPolicy().
		WithStrip("Cookie", "X-Forwarded-For").
		WithDefault(ActionPassthrough)

	src := http.Header{}
	src.Set("Cookie", "session=abc")
	src.Set("X-Forwarded-For", "1.2.3.4")
	src.Set("User-Agent", "TestClient/1.0")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("Cookie") != "" {
		t.Errorf("expected Cookie to be stripped, got %q", dst.Get("Cookie"))
	}
	if dst.Get("X-Forwarded-For") != "" {
		t.Errorf("expected X-Forwarded-For to be stripped, got %q", dst.Get("X-Forwarded-For"))
	}
	if dst.Get("User-Agent") != "TestClient/1.0" {
		t.Errorf("expected User-Agent to pass through, got %q", dst.Get("User-Agent"))
	}
}

func TestPolicyDefaultPassthrough(t *testing.T) {
	policy := NewPolicy().
		WithDefault(ActionPassthrough)

	src := http.Header{}
	src.Set("User-Agent", "TestClient/1.0")
	src.Set("X-Custom", "value")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("User-Agent") != "TestClient/1.0" {
		t.Errorf("expected User-Agent to pass through, got %q", dst.Get("User-Agent"))
	}
	if dst.Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom to pass through, got %q", dst.Get("X-Custom"))
	}
}

func TestPolicyDefaultStrip(t *testing.T) {
	policy := NewPolicy().
		WithDefault(ActionStrip)

	src := http.Header{}
	src.Set("User-Agent", "TestClient/1.0")
	src.Set("X-Custom", "value")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("User-Agent") != "" {
		t.Errorf("expected User-Agent to be stripped, got %q", dst.Get("User-Agent"))
	}
	if dst.Get("X-Custom") != "" {
		t.Errorf("expected X-Custom to be stripped, got %q", dst.Get("X-Custom"))
	}
}

func TestApplyAuthClaude(t *testing.T) {
	src := http.Header{}
	src.Set("Authorization", "Bearer sk-ant-123")

	dst := http.Header{}
	NewPolicy().ApplyAuth(src, dst, "claude_messages")

	if dst.Get("x-api-key") != "sk-ant-123" {
		t.Errorf("expected x-api-key to be set, got %q", dst.Get("x-api-key"))
	}
}

func TestApplyAuthOpenAI(t *testing.T) {
	src := http.Header{}
	src.Set("x-api-key", "sk-123")

	dst := http.Header{}
	NewPolicy().ApplyAuth(src, dst, "openai_chat")

	if dst.Get("Authorization") != "Bearer sk-123" {
		t.Errorf("expected Authorization to be set, got %q", dst.Get("Authorization"))
	}
}

func TestApplyAuthOpenAIBearer(t *testing.T) {
	src := http.Header{}
	src.Set("Authorization", "Bearer sk-123")

	dst := http.Header{}
	NewPolicy().ApplyAuth(src, dst, "openai_chat")

	if dst.Get("Authorization") != "Bearer sk-123" {
		t.Errorf("expected Authorization to pass through, got %q", dst.Get("Authorization"))
	}
}

func TestDefaultProxyPolicy(t *testing.T) {
	policy := DefaultProxyPolicy()

	src := http.Header{}
	src.Set("User-Agent", "TestClient/1.0")
	src.Set("X-Request-ID", "req-123")
	src.Set("Cookie", "session=abc")
	src.Set("X-Forwarded-For", "1.2.3.4")

	dst := http.Header{}
	policy.Apply(src, dst)

	if dst.Get("User-Agent") != "TestClient/1.0" {
		t.Errorf("expected User-Agent to pass through, got %q", dst.Get("User-Agent"))
	}
	if dst.Get("X-Request-ID") != "req-123" {
		t.Errorf("expected X-Request-ID to pass through, got %q", dst.Get("X-Request-ID"))
	}
	if dst.Get("Cookie") != "" {
		t.Errorf("expected Cookie to be stripped, got %q", dst.Get("Cookie"))
	}
	if dst.Get("X-Forwarded-For") != "" {
		t.Errorf("expected X-Forwarded-For to be stripped, got %q", dst.Get("X-Forwarded-For"))
	}
}
