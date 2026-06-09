# Changelog

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
