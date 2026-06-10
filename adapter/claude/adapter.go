package claude

import (
	"encoding/json"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type Adapter struct{}

func New() adapter.Adapter {
	return &Adapter{}
}

func (a *Adapter) Protocol() adapter.Protocol {
	return adapter.ProtocolClaudeMessages
}

func (a *Adapter) DecodeRequest(data json.RawMessage) (*uni.RequestParams, *adapter.Report, error) {
	return decodeRequest(data)
}

func (a *Adapter) EncodeRequest(params *uni.RequestParams) (json.RawMessage, *adapter.Report, error) {
	return encodeRequest(params)
}

func (a *Adapter) DecodeResponse(data json.RawMessage) (*uni.Response, *adapter.Report, error) {
	return decodeResponse(data)
}

func (a *Adapter) EncodeResponse(resp *uni.Response) (json.RawMessage, *adapter.Report, error) {
	return encodeResponse(resp)
}

func (a *Adapter) DecodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *adapter.Report, error) {
	return decodeStreamEvent(data)
}

func (a *Adapter) EncodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *adapter.Report, error) {
	return encodeStreamEvent(event)
}

func (a *Adapter) DecodeError(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *adapter.Report, error) {
	return decodeError(data, statusCode)
}

func (a *Adapter) EncodeError(err *uni.ErrorResponse) (json.RawMessage, *adapter.Report, error) {
	return encodeError(err)
}
