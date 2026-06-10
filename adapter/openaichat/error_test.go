package openaichat

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeError(t *testing.T) {
	data := json.RawMessage(`{
		"error": {
			"message": "The model does not exist",
			"type": "invalid_request_error",
			"param": null,
			"code": "model_not_found"
		}
	}`)

	errResp, report, err := decodeError(data, 404)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	if errResp.Type != "invalid_request_error" {
		t.Errorf("expected type invalid_request_error, got %q", errResp.Type)
	}
	if errResp.Message != "The model does not exist" {
		t.Errorf("expected message, got %q", errResp.Message)
	}
	if errResp.Code != "model_not_found" {
		t.Errorf("expected code model_not_found, got %q", errResp.Code)
	}
	if errResp.Status != 404 {
		t.Errorf("expected status 404, got %d", errResp.Status)
	}
}

func TestEncodeError(t *testing.T) {
	errResp := &uni.ErrorResponse{
		Type:    "insufficient_quota",
		Message: "You exceeded your current quota",
		Code:    "insufficient_quota",
		Status:  429,
	}

	data, report, err := encodeError(errResp)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	var result openaiError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if result.Error.Type != "insufficient_quota" {
		t.Errorf("expected type insufficient_quota, got %q", result.Error.Type)
	}
	if result.Error.Message != "You exceeded your current quota" {
		t.Errorf("expected message, got %q", result.Error.Message)
	}
	if result.Error.Code != "insufficient_quota" {
		t.Errorf("expected code, got %v", result.Error.Code)
	}
}
