package openairesp

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeErrorResponse(t *testing.T) {
	data := json.RawMessage(`{
		"error": {
			"type": "invalid_request_error",
			"code": "model_not_found",
			"message": "The model 'gpt-999' does not exist"
		}
	}`)

	errResp, report, err := decodeErrorResponse(data, 404)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	if errResp.Type != "invalid_request_error" {
		t.Errorf("expected type invalid_request_error, got %q", errResp.Type)
	}
	if errResp.Code != "model_not_found" {
		t.Errorf("expected code model_not_found, got %q", errResp.Code)
	}
	if errResp.Message != "The model 'gpt-999' does not exist" {
		t.Errorf("expected message, got %q", errResp.Message)
	}
	if errResp.Status != 404 {
		t.Errorf("expected status 404, got %d", errResp.Status)
	}
}

func TestEncodeErrorResponse(t *testing.T) {
	errResp := &uni.ErrorResponse{
		Type:    "server_error",
		Message: "Internal server error",
		Code:    "internal_error",
		Status:  500,
	}

	data, report, err := encodeErrorResponse(errResp)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}

	var result responsesError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if result.Error.Type != "server_error" {
		t.Errorf("expected type server_error, got %q", result.Error.Type)
	}
	if result.Error.Code != "internal_error" {
		t.Errorf("expected code internal_error, got %q", result.Error.Code)
	}
	if result.Error.Message != "Internal server error" {
		t.Errorf("expected message, got %q", result.Error.Message)
	}
}
