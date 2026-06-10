package adapter

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

type mockAdapter struct {
	protocol Protocol
}

func (m *mockAdapter) Protocol() Protocol { return m.protocol }

func (m *mockAdapter) DecodeRequest(data json.RawMessage) (*uni.RequestParams, *Report, error) {
	return &uni.RequestParams{Model: "decode-req"}, NewReport(), nil
}

func (m *mockAdapter) EncodeRequest(params *uni.RequestParams) (json.RawMessage, *Report, error) {
	return json.RawMessage(`{"encoded":"request"}`), NewReport(), nil
}

func (m *mockAdapter) DecodeResponse(data json.RawMessage) (*uni.Response, *Report, error) {
	return &uni.Response{Model: "decode-resp"}, NewReport(), nil
}

func (m *mockAdapter) EncodeResponse(resp *uni.Response) (json.RawMessage, *Report, error) {
	return json.RawMessage(`{"encoded":"response"}`), NewReport(), nil
}

func (m *mockAdapter) DecodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *Report, error) {
	return &uni.StreamEvent{Type: uni.StreamEventDelta}, NewReport(), nil
}

func (m *mockAdapter) EncodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *Report, error) {
	return json.RawMessage(`{"encoded":"stream"}`), NewReport(), nil
}

func (m *mockAdapter) DecodeError(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *Report, error) {
	return &uni.ErrorResponse{Message: "decode-err", Status: statusCode}, NewReport(), nil
}

func (m *mockAdapter) EncodeError(err *uni.ErrorResponse) (json.RawMessage, *Report, error) {
	return json.RawMessage(`{"encoded":"error"}`), NewReport(), nil
}

func TestNewConverter(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("mock_a")}
	b := &mockAdapter{protocol: Protocol("mock_b")}

	c := NewConverter(WithAdapter(a), WithAdapter(b))

	if len(c.List()) != 2 {
		t.Fatalf("expected 2 protocols, got %d", len(c.List()))
	}
}

func TestConverter_RegisterAndGet(t *testing.T) {
	c := NewConverter()
	m := &mockAdapter{protocol: Protocol("mock")}

	c.Register(m)

	got, err := c.Get(Protocol("mock"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Protocol() != m.protocol {
		t.Fatalf("expected protocol %s, got %s", m.protocol, got.Protocol())
	}

	_, err = c.Get(Protocol("missing"))
	if err == nil {
		t.Fatal("expected error for missing adapter")
	}
}

func TestConverter_List(t *testing.T) {
	c := NewConverter()
	if len(c.List()) != 0 {
		t.Fatalf("expected empty list, got %d", len(c.List()))
	}

	c.Register(&mockAdapter{protocol: Protocol("x")})
	c.Register(&mockAdapter{protocol: Protocol("y")})

	list := c.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 protocols, got %d", len(list))
	}
}

func TestConverter_ConvertRequest(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("src")}
	b := &mockAdapter{protocol: Protocol("dst")}
	c := NewConverter(WithAdapter(a), WithAdapter(b))

	result, report, err := c.ConvertRequest(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}
	if string(result) != `{"encoded":"request"}` {
		t.Fatalf("unexpected result: %s", string(result))
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestConverter_ConvertRequest_UnregisteredSource(t *testing.T) {
	c := NewConverter()
	_, report, err := c.ConvertRequest(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unregistered source")
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestConverter_ConvertRequest_UnregisteredTarget(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("src")}
	c := NewConverter(WithAdapter(a))
	_, report, err := c.ConvertRequest(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unregistered target")
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestConverter_ConvertResponse(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("src")}
	b := &mockAdapter{protocol: Protocol("dst")}
	c := NewConverter(WithAdapter(a), WithAdapter(b))

	result, report, err := c.ConvertResponse(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ConvertResponse failed: %v", err)
	}
	if string(result) != `{"encoded":"response"}` {
		t.Fatalf("unexpected result: %s", string(result))
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestConverter_ConvertStreamEvent(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("src")}
	b := &mockAdapter{protocol: Protocol("dst")}
	c := NewConverter(WithAdapter(a), WithAdapter(b))

	result, report, err := c.ConvertStreamEvent(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ConvertStreamEvent failed: %v", err)
	}
	if string(result) != `{"encoded":"stream"}` {
		t.Fatalf("unexpected result: %s", string(result))
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestConverter_ConvertError(t *testing.T) {
	a := &mockAdapter{protocol: Protocol("src")}
	b := &mockAdapter{protocol: Protocol("dst")}
	c := NewConverter(WithAdapter(a), WithAdapter(b))

	result, report, err := c.ConvertError(Protocol("src"), Protocol("dst"), json.RawMessage(`{}`), 400)
	if err != nil {
		t.Fatalf("ConvertError failed: %v", err)
	}
	if string(result) != `{"encoded":"error"}` {
		t.Fatalf("unexpected result: %s", string(result))
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestReport_AddWarning(t *testing.T) {
	r := NewReport()
	r.AddWarning(SeverityWarning, "field", "message")

	if len(r.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(r.Warnings))
	}
	if r.Warnings[0].Severity != SeverityWarning {
		t.Fatalf("expected severity warning, got %d", r.Warnings[0].Severity)
	}
	if r.Warnings[0].Field != "field" {
		t.Fatalf("expected field, got %s", r.Warnings[0].Field)
	}
}

func TestReport_AddLostField(t *testing.T) {
	r := NewReport()
	r.AddLostField("src", "field", "reason")

	if len(r.LostFields) != 1 {
		t.Fatalf("expected 1 lost field, got %d", len(r.LostFields))
	}
	if r.LostFields[0].Source != "src" {
		t.Fatalf("expected source src, got %s", r.LostFields[0].Source)
	}
}

func TestReport_Merge(t *testing.T) {
	r1 := NewReport()
	r1.AddWarning(SeverityInfo, "a", "b")
	r1.AddLostField("s", "f", "r")

	r2 := NewReport()
	r2.AddWarning(SeverityWarning, "c", "d")
	r2.AddLostField("s2", "f2", "r2")

	r1.Merge(r2)

	if len(r1.Warnings) != 2 {
		t.Fatalf("expected 2 warnings after merge, got %d", len(r1.Warnings))
	}
	if len(r1.LostFields) != 2 {
		t.Fatalf("expected 2 lost fields after merge, got %d", len(r1.LostFields))
	}
}

func TestReport_MergeNil(t *testing.T) {
	r := NewReport()
	r.Merge(nil)

	if len(r.Warnings) != 0 || len(r.LostFields) != 0 {
		t.Fatal("expected no changes after merging nil")
	}
}

func TestWarningSeverity_Constants(t *testing.T) {
	if SeverityInfo != 0 {
		t.Fatalf("expected SeverityInfo == 0, got %d", SeverityInfo)
	}
	if SeverityWarning != 1 {
		t.Fatalf("expected SeverityWarning == 1, got %d", SeverityWarning)
	}
	if SeverityError != 2 {
		t.Fatalf("expected SeverityError == 2, got %d", SeverityError)
	}
}
