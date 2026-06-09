package openaichat

import "encoding/json"

type chatCompletionRequest struct {
	Model               string              `json:"model"`
	Messages            []chatMessage       `json:"messages"`
	Tools               []toolDef           `json:"tools,omitempty"`
	ToolChoice          json.RawMessage     `json:"tool_choice,omitempty"`
	Temperature         *float64            `json:"temperature,omitempty"`
	TopP                *float64            `json:"top_p,omitempty"`
	FrequencyPenalty    *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64            `json:"presence_penalty,omitempty"`
	MaxCompletionTokens *int64              `json:"max_completion_tokens,omitempty"`
	MaxTokens           *int64              `json:"max_tokens,omitempty"`
	N                   *int64              `json:"n,omitempty"`
	Stream              bool                `json:"stream,omitempty"`
	Stop                json.RawMessage     `json:"stop,omitempty"`
	Seed                *int64              `json:"seed,omitempty"`
	StreamOptions       json.RawMessage     `json:"stream_options,omitempty"`
	User                string              `json:"user,omitempty"`
	ParallelToolCalls   *bool               `json:"parallel_tool_calls,omitempty"`
	ResponseFormat      json.RawMessage     `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role         string          `json:"role"`
	Content      json.RawMessage `json:"content,omitempty"`
	Name         string          `json:"name,omitempty"`
	ToolCalls    []toolCall      `json:"tool_calls,omitempty"`
	ToolCallID   string          `json:"tool_call_id,omitempty"`
	Refusal      string          `json:"refusal,omitempty"`
}

type contentPart struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ImageURL   *imageURLPart   `json:"image_url,omitempty"`
	InputAudio *inputAudioPart `json:"input_audio,omitempty"`
	File       *filePart       `json:"file,omitempty"`
}

type imageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type inputAudioPart struct {
	Data   string `json:"data"`
	Format string `json:"format,omitempty"`
}

type filePart struct {
	FileData string `json:"file_data,omitempty"`
	FileName string `json:"filename,omitempty"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function functionCallData `json:"function"`
}

type functionCallData struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []choice `json:"choices"`
	Usage             *usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	ServiceTier       string   `json:"service_tier,omitempty"`
}

type choice struct {
	Index        int             `json:"index"`
	Message      chatMessage     `json:"message"`
	FinishReason string          `json:"finish_reason"`
	LogProbs     json.RawMessage `json:"logprobs,omitempty"`
}

type usage struct {
	PromptTokens        int64                   `json:"prompt_tokens"`
	CompletionTokens    int64                   `json:"completion_tokens"`
	TotalTokens         int64                   `json:"total_tokens"`
	CompletionDetails   *completionTokenDetails `json:"completion_tokens_details,omitempty"`
}

type completionTokenDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`
}

type chatCompletionChunk struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []chunkChoice `json:"choices"`
	Usage             *usage        `json:"usage,omitempty"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
}

type chunkChoice struct {
	Index        int              `json:"index"`
	Delta        chatMessageDelta `json:"delta"`
	FinishReason string           `json:"finish_reason,omitempty"`
	LogProbs     json.RawMessage  `json:"logprobs,omitempty"`
}

type chatMessageDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []toolCallDelta `json:"tool_calls,omitempty"`
	Refusal   string          `json:"refusal,omitempty"`
}

type toolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function functionCallData `json:"function,omitempty"`
}
