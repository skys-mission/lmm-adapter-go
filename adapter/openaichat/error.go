package openaichat

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type openaiError struct {
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

func decodeError(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *adapter.Report, error) {
	report := adapter.NewReport()

	var errResp openaiError
	if err := json.Unmarshal(data, &errResp); err != nil {
		return nil, report, fmt.Errorf("unmarshal openai error: %w", err)
	}

	code := ""
	if errResp.Error.Code != nil {
		if c, ok := errResp.Error.Code.(string); ok {
			code = c
		}
	}

	param := ""
	if errResp.Error.Param != nil {
		if p, ok := errResp.Error.Param.(string); ok {
			param = p
		}
	}

	return &uni.ErrorResponse{
		Type:    errResp.Error.Type,
		Message: errResp.Error.Message,
		Code:    code,
		Param:   param,
		Status:  statusCode,
		Ext:     make(uni.ExtData),
	}, report, nil
}

func encodeError(err *uni.ErrorResponse) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	errResp := openaiError{}
	errResp.Error.Type = err.Type
	errResp.Error.Message = err.Message

	if err.Code != "" {
		errResp.Error.Code = err.Code
	}
	if err.Param != "" {
		errResp.Error.Param = err.Param
	}

	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return nil, report, fmt.Errorf("marshal openai error: %w", marshalErr)
	}

	return data, report, nil
}
