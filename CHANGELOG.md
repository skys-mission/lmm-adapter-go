# Changelog

## [0.1.1] — 2026-06-10

### Fixed

- **Claude streaming stop reason decoding**: `message_delta` events now correctly read `stop_reason` from the nested `delta` object (was incorrectly looking at the top level, causing `StopReason` to always be empty in streaming accumulators)
- **OpenAI Chat streaming stop reason encoding**: `StreamEventStop` now correctly maps `event.StopReason` to `finish_reason` on the chunk when choices do not already carry a `FinishReason`
- **Claude `tool_result` decoding**: non-text content parts inside `tool_result` are now reported as `LostField` instead of being silently dropped
- **Claude `tool_choice` unknown type handling**: unknown `tool_choice.type` values now report a `LostField` and default to `auto` instead of silently producing an empty result
- **OpenAI Chat unknown content part handling**: unknown `content_part.type` values now report a `LostField` before falling back to `TextPart`
- **OpenAI Responses extension restoration errors**: `previous_response_id`, `store`, `truncation`, `service_tier`, `background`, and `max_tool_calls` restoration now propagates `json.Unmarshal` errors instead of silently ignoring them
- **OpenAI Responses unknown `tool_choice` string**: reports a `LostField` instead of silently defaulting to `auto`
- **Accumulator tool call matching**: `finalizeToolCalls` now matches by `ToolCallID` instead of by name, eliminating collision bugs when multiple tool calls share the same name
- **Proxy streaming error handling**: `pipeline.Pipe` errors are now captured and written as best-effort SSE error events to the client instead of being silently dropped

### Added

- **Converter unit tests**: `adapter/adapter_test.go` with comprehensive coverage of `Converter`, `Report`, and severity constants (adapter package coverage: 84.8%)
- **Adapter package test coverage**: targeted tests for previously uncovered stream encode/decode paths in `adapter/claude`, `adapter/openairesp`, and `adapter/openaichat`

## [0.1.0] — 2026-06-10

### Added

- **Unified IR** (`uni/`): Type-safe intermediate representation with 9 sealed `ContentPart` types
- **3 protocol adapters**: Claude Messages, OpenAI Chat Completions, OpenAI Responses
- **Bidirectional conversion**: Request, response, error, and streaming event conversion
- **Stream pipeline**: SSE reader/writer, stateful `StreamConverter`, `Pipeline`, `Accumulator`
- **HTTP proxy handler**: Auto error detection, streaming detection, header policy
- **Header policy**: Passthrough, mapping, strip, auth conversion (Bearer ↔ x-api-key)
- **Middleware hooks**: `ConversionHook` interface with structured logging
- **Token estimation**: CJK-aware heuristic counting, `TruncateToTokenLimit`
- **Zero external dependencies**: Standard library only
