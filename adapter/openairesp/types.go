package openairesp

import "encoding/json"

type responseRequest struct {
	Model               string          `json:"model"`
	Input               json.RawMessage `json:"input"`
	Instructions        string          `json:"instructions,omitempty"`
	Tools               []toolDef       `json:"tools,omitempty"`
	ToolChoice          json.RawMessage `json:"tool_choice,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	TopP                *float64        `json:"top_p,omitempty"`
	MaxOutputTokens     *int64          `json:"max_output_tokens,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	Store               *bool           `json:"store,omitempty"`
	PreviousResponseID  string          `json:"previous_response_id,omitempty"`
	Reasoning           json.RawMessage `json:"reasoning,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	Truncation          string          `json:"truncation,omitempty"`
	ServiceTier         string          `json:"service_tier,omitempty"`
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"`
	Background          *bool           `json:"background,omitempty"`
	MaxToolCalls        *int64          `json:"max_tool_calls,omitempty"`
}

type inputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
	ID        string          `json:"id,omitempty"`
	Summary   json.RawMessage `json:"summary,omitempty"`
	Status    string          `json:"status,omitempty"`
}

type inputContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
	FileData string `json:"file_data,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type toolDef struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type responseResponse struct {
	ID                 string          `json:"id"`
	Object             string          `json:"object"`
	CreatedAt          int64           `json:"created_at"`
	Model              string          `json:"model"`
	Output             []outputItem    `json:"output"`
	Status             string          `json:"status,omitempty"`
	Usage              *respUsage      `json:"usage,omitempty"`
	Error              *respError      `json:"error,omitempty"`
	IncompleteDetails  json.RawMessage `json:"incomplete_details,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	OutputText         string          `json:"output_text,omitempty"`
}

type outputItem struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Status    string          `json:"status,omitempty"`
	Role      string          `json:"role,omitempty"`
	Content   []outputContent `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
	Summary   json.RawMessage `json:"summary,omitempty"`
}

type outputContent struct {
	Type        string       `json:"type"`
	Text        string       `json:"text,omitempty"`
	Refusal     string       `json:"refusal,omitempty"`
	Annotations []annotation `json:"annotations,omitempty"`
}

type annotation struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"`
}

type respUsage struct {
	InputTokens          int64                 `json:"input_tokens"`
	OutputTokens         int64                 `json:"output_tokens"`
	TotalTokens          int64                 `json:"total_tokens"`
	OutputTokensDetails  *outputTokensDetails  `json:"output_tokens_details,omitempty"`
}

type outputTokensDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`
}

type respError struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type responseStreamEvent struct {
	Type           string          `json:"type"`
	SequenceNumber int64           `json:"sequence_number,omitempty"`
	Response       *responseResponse `json:"response,omitempty"`
	OutputIndex    int64           `json:"output_index,omitempty"`
	Item           *outputItem     `json:"item,omitempty"`
	ItemID         string          `json:"item_id,omitempty"`
	Delta          string          `json:"delta,omitempty"`
	Text           string          `json:"text,omitempty"`
	Arguments      string          `json:"arguments,omitempty"`
	ContentIndex   int64           `json:"content_index,omitempty"`
	Part           *outputContent  `json:"part,omitempty"`
	Error          *respError      `json:"error,omitempty"`
}
