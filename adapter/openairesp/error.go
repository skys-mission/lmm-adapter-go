package openairesp

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type responsesError struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeErrorResponse(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *adapter.Report, error) {
	report := adapter.NewReport()

	var errResp responsesError
	if err := json.Unmarshal(data, &errResp); err != nil {
		return nil, report, fmt.Errorf("unmarshal responses error: %w", err)
	}

	return &uni.ErrorResponse{
		Type:    errResp.Error.Type,
		Message: errResp.Error.Message,
		Code:    errResp.Error.Code,
		Status:  statusCode,
		Ext:     make(uni.ExtData),
	}, report, nil
}

func encodeErrorResponse(err *uni.ErrorResponse) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	errResp := responsesError{}
	errResp.Error.Type = err.Type
	errResp.Error.Message = err.Message
	errResp.Error.Code = err.Code

	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return nil, report, fmt.Errorf("marshal responses error: %w", marshalErr)
	}

	return data, report, nil
}
