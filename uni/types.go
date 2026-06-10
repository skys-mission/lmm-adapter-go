package uni

import (
	"encoding/json"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleDeveloper Role = "developer"
	RoleTool      Role = "tool"
)

type Message struct {
	Role    Role          `json:"role"`
	Content []ContentPart `json:"content"`
	Ext     ExtData       `json:"ext,omitempty"`
}

type ContentPartType string

const (
	ContentPartText             ContentPartType = "text"
	ContentPartImage            ContentPartType = "image"
	ContentPartAudio            ContentPartType = "audio"
	ContentPartFile             ContentPartType = "file"
	ContentPartThinking         ContentPartType = "thinking"
	ContentPartRedactedThinking ContentPartType = "redacted_thinking"
	ContentPartToolUse          ContentPartType = "tool_use"
	ContentPartToolResult       ContentPartType = "tool_result"
	ContentPartRefusal          ContentPartType = "refusal"
)

type ContentPart interface {
	ContentType() ContentPartType
	contentPart()
}

type TextPart struct {
	Text string `json:"text"`
}

func (TextPart) ContentType() ContentPartType { return ContentPartText }
func (TextPart) contentPart()                 {}

type ImagePart struct {
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

func (ImagePart) ContentType() ContentPartType { return ContentPartImage }
func (ImagePart) contentPart()                 {}

type AudioPart struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

func (AudioPart) ContentType() ContentPartType { return ContentPartAudio }
func (AudioPart) contentPart()                 {}

type FilePart struct {
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
	Name      string `json:"name,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

func (FilePart) ContentType() ContentPartType { return ContentPartFile }
func (FilePart) contentPart()                 {}

type ThinkingPart struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

func (ThinkingPart) ContentType() ContentPartType { return ContentPartThinking }
func (ThinkingPart) contentPart()                 {}

type RedactedThinkingPart struct {
	Data string `json:"data"`
}

func (RedactedThinkingPart) ContentType() ContentPartType { return ContentPartRedactedThinking }
func (RedactedThinkingPart) contentPart()                 {}

type ToolUsePart struct {
	ToolCallID string          `json:"tool_call_id"`
	ToolName   string          `json:"tool_name"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

func (ToolUsePart) ContentType() ContentPartType { return ContentPartToolUse }
func (ToolUsePart) contentPart()                 {}

type ToolResultPart struct {
	ToolCallID string        `json:"tool_call_id"`
	Content    []ContentPart `json:"content,omitempty"`
	IsError    bool          `json:"is_error,omitempty"`
}

func (ToolResultPart) ContentType() ContentPartType { return ContentPartToolResult }
func (ToolResultPart) contentPart()                 {}

type RefusalPart struct {
	Refusal string `json:"refusal"`
}

func (RefusalPart) ContentType() ContentPartType { return ContentPartRefusal }
func (RefusalPart) contentPart()                 {}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
	Ext         ExtData         `json:"ext,omitempty"`
}

type ToolChoice struct {
	Type     ToolChoiceType `json:"type"`
	ToolName string         `json:"tool_name,omitempty"`
}

type ToolChoiceType string

const (
	ToolChoiceAuto     ToolChoiceType = "auto"
	ToolChoiceRequired ToolChoiceType = "required"
	ToolChoiceNone     ToolChoiceType = "none"
	ToolChoiceSpecific ToolChoiceType = "specific"
)

type Usage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	TotalTokens              int64 `json:"total_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
}

type StopReason string

const (
	StopReasonEndTurn       StopReason = "end_turn"
	StopReasonMaxTokens     StopReason = "max_tokens"
	StopReasonToolCalls     StopReason = "tool_calls"
	StopReasonStopSequence  StopReason = "stop_sequence"
	StopReasonContentFilter StopReason = "content_filter"
	StopReasonRefusal       StopReason = "refusal"
)

type RequestParams struct {
	Model             string      `json:"model"`
	Messages          []Message   `json:"messages"`
	Tools             []Tool      `json:"tools,omitempty"`
	ToolChoice        *ToolChoice `json:"tool_choice,omitempty"`
	MaxTokens         *int64      `json:"max_tokens,omitempty"`
	Temperature       *float64    `json:"temperature,omitempty"`
	TopP              *float64    `json:"top_p,omitempty"`
	TopK              *int64      `json:"top_k,omitempty"`
	FrequencyPenalty  *float64    `json:"frequency_penalty,omitempty"`
	PresencePenalty   *float64    `json:"presence_penalty,omitempty"`
	StopSequences     []string    `json:"stop_sequences,omitempty"`
	Stream            bool        `json:"stream,omitempty"`
	ParallelToolCalls *bool       `json:"parallel_tool_calls,omitempty"`
	Seed              *int64      `json:"seed,omitempty"`
	Ext               ExtData     `json:"ext,omitempty"`
}

type Response struct {
	ID           string     `json:"id"`
	Model        string     `json:"model"`
	Messages     []Message  `json:"messages"`
	Usage        Usage      `json:"usage"`
	StopReason   StopReason `json:"stop_reason"`
	StopSequence string     `json:"stop_sequence,omitempty"`
	Created      int64      `json:"created,omitempty"`
	Ext          ExtData    `json:"ext,omitempty"`
}

type StreamEvent struct {
	Type         StreamEventType `json:"type"`
	ID           string          `json:"id,omitempty"`
	Created      int64           `json:"created,omitempty"`
	Model        string          `json:"model,omitempty"`
	Choices      []StreamChoice  `json:"choices,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
	StopReason   *StopReason     `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Error        *StreamError    `json:"error,omitempty"`
	Ext          ExtData         `json:"ext,omitempty"`
}

type StreamEventType string

const (
	StreamEventStart        StreamEventType = "start"
	StreamEventDelta        StreamEventType = "delta"
	StreamEventStop         StreamEventType = "stop"
	StreamEventError        StreamEventType = "error"
	StreamEventToolCall     StreamEventType = "tool_call"
	StreamEventContentStart StreamEventType = "content_start"
	StreamEventContentStop  StreamEventType = "content_stop"
)

type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *StopReason `json:"finish_reason,omitempty"`
}

type StreamDelta struct {
	Role      Role            `json:"role,omitempty"`
	Content   []ContentPart   `json:"content,omitempty"`
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

type ToolCallDelta struct {
	Index      int    `json:"index"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
}

type StreamError struct {
	Type    string  `json:"type"`
	Message string  `json:"message"`
	Code    string  `json:"code,omitempty"`
	Ext     ExtData `json:"ext,omitempty"`
}

type ErrorResponse struct {
	Type    string  `json:"type"`
	Message string  `json:"message"`
	Code    string  `json:"code,omitempty"`
	Param   string  `json:"param,omitempty"`
	Status  int     `json:"status,omitempty"`
	Ext     ExtData `json:"ext,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

type ExtData map[string]json.RawMessage

func (e ExtData) Get(key string, v any) error {
	raw, ok := e[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

func (e *ExtData) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if *e == nil {
		*e = make(ExtData)
	}
	(*e)[key] = raw
	return nil
}

func (e ExtData) Has(key string) bool {
	_, ok := e[key]
	return ok
}

func UserMessage(content ...ContentPart) Message {
	return Message{Role: RoleUser, Content: content}
}

func AssistantMessage(content ...ContentPart) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func SystemMessage(content ...ContentPart) Message {
	return Message{Role: RoleSystem, Content: content}
}

func DeveloperMessage(content ...ContentPart) Message {
	return Message{Role: RoleDeveloper, Content: content}
}

func ToolMessage(callID string, content []ContentPart, isError bool) Message {
	return Message{
		Role:    RoleTool,
		Content: []ContentPart{ToolResultPart{ToolCallID: callID, Content: content, IsError: isError}},
	}
}
