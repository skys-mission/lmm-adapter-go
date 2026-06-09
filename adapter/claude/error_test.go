package claude

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeError(t *testing.T) {
	data := json.RawMessage(`{
		"type": "error",
		"error": {
			"type": "invalid_request_error",
			"message": "max_tokens: field required"
		}
	}`)

	errResp, report, err := decodeError(data, 400)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	if errResp.Type != "invalid_request_error" {
		t.Errorf("expected type invalid_request_error, got %q", errResp.Type)
	}
	if errResp.Message != "max_tokens: field required" {
		t.Errorf("expected message 'max_tokens: field required', got %q", errResp.Message)
	}
	if errResp.Status != 400 {
		t.Errorf("expected status 400, got %d", errResp.Status)
	}
}

func TestEncodeError(t *testing.T) {
	errResp := &uni.ErrorResponse{
		Type:    "rate_limit_error",
		Message: "Rate limit exceeded",
		Status:  429,
	}

	data, report, err := encodeError(errResp)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	var result claudeError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if result.Type != "error" {
		t.Errorf("expected type 'error', got %q", result.Type)
	}
	if result.Error.Type != "rate_limit_error" {
		t.Errorf("expected error type rate_limit_error, got %q", result.Error.Type)
	}
	if result.Error.Message != "Rate limit exceeded" {
		t.Errorf("expected message 'Rate limit exceeded', got %q", result.Error.Message)
	}
}
