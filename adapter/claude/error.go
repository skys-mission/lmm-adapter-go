package claude

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type claudeError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *adapter.Report, error) {
	report := adapter.NewReport()

	var errResp claudeError
	if err := json.Unmarshal(data, &errResp); err != nil {
		return nil, report, fmt.Errorf("unmarshal claude error: %w", err)
	}

	return &uni.ErrorResponse{
		Type:    errResp.Error.Type,
		Message: errResp.Error.Message,
		Status:  statusCode,
		Ext:     make(uni.ExtData),
	}, report, nil
}

func encodeError(err *uni.ErrorResponse) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	errResp := claudeError{
		Type: "error",
	}
	errResp.Error.Type = err.Type
	errResp.Error.Message = err.Message

	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return nil, report, fmt.Errorf("marshal claude error: %w", marshalErr)
	}

	return data, report, nil
}
