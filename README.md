# lmm-adapter-go

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/skys-mission/lmm-adapter-go)](https://goreportcard.com/report/github.com/skys-mission/lmm-adapter-go)

[中文文档](README.zh.md)

LLM API protocol conversion library for Go — bidirectional translation between Claude Messages, OpenAI Chat Completions, and OpenAI Responses through a type-safe unified intermediate representation. **Zero external dependencies.**

## Why?

Building an LLM-powered app means dealing with multiple API formats. Every provider has its own request/response structure, streaming protocol, and error format. lmm-adapter-go gives you a single, type-safe internal API — write your application logic once, talk to any LLM provider.

```
┌──────────────────────────────────────────────────────┐
│                   Your Application                   │
├──────────────────────────────────────────────────────┤
│                  Unified IR (uni/)                   │
├──────────┬──────────────────────┬───────────────────┤
│  Claude  │   OpenAI Chat        │  OpenAI Responses  │
│ Messages │   Completions        │                    │
└──────────┴──────────────────────┴───────────────────┘
```

## Installation

```bash
go get github.com/skys-mission/lmm-adapter-go
```

## Quick Start

```go
import (
    "github.com/skys-mission/lmm-adapter-go/adapter"
    "github.com/skys-mission/lmm-adapter-go/adapter/claude"
    "github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
)

c := adapter.NewConverter(
    adapter.WithAdapter(claude.New()),
    adapter.WithAdapter(openaichat.New()),
)

// Claude-format request → OpenAI Chat format
openaiReq, report, _ := c.ConvertRequest(
    adapter.ProtocolClaudeMessages,
    adapter.ProtocolOpenAIChat,
    claudeReqJSON,
)

// Check what was lost during conversion
for _, f := range report.LostFields {
    log.Printf("lost field: %s.%s — %s", f.Source, f.Field, f.Reason)
}
```

### One-liner proxy server

```go
// Expose /v1/messages as a Claude-compatible endpoint
// that forwards to OpenAI behind the scenes
handler := proxy.NewHandler(
    adapter.NewConverter(
        adapter.WithAdapter(claude.New()),
        adapter.WithAdapter(openaichat.New()),
    ),
    adapter.ProtocolClaudeMessages,
    adapter.ProtocolOpenAIChat,
    "https://api.openai.com/v1/chat/completions",
)
http.Handle("/v1/messages", handler)
http.ListenAndServe(":8080", nil)
```

Streaming, auth conversion, and error mapping are handled automatically.

## Features

### Core
- Bidirectional conversion between **Claude Messages**, **OpenAI Chat Completions**, and **OpenAI Responses**
- Type-safe unified IR with **9 sealed ContentPart types** — exhaustive type switches, no surprises
- Request, response, error, and streaming event conversion
- **Conversion quality report** — lost fields and warnings, never silent data loss

### Streaming
- SSE reader/writer with standard protocol compliance
- Stateful `StreamConverter` auto-generates missing events (start, role, content-stop)
- `Accumulator` collects streaming deltas into a complete response
- `Pipeline` wires everything together: SSE in → convert → SSE out

### Proxy
- `POST`-only HTTP handler with automatic streaming detection
- Error responses (4xx/5xx) automatically converted back to client's protocol
- Header policy: **passthrough**, **mapping**, **strip**, and **auth conversion** (Bearer ↔ x-api-key)
- Lost-field summary via `X-Conversion-Lost-Fields` response header

### Quality
- **Zero external dependencies** — standard library only
- Token estimation with CJK-aware ratios (Chinese/Japanese/Korean)
- Structured logging hooks via `middleware.ConversionHook`
- `go build ./...` / `go vet ./...` clean, 11 packages passing tests

## Package Map

| Package | Responsibility |
|---------|---------------|
| `uni/` | Unified intermediate representation: `ContentPart`, `Message`, `RequestParams`, `Response`, etc. |
| `adapter/` | `Converter` registry, `Adapter` and sub-interfaces, `Report` |
| `adapter/claude/` | Claude Messages ↔ unified IR |
| `adapter/openaichat/` | OpenAI Chat Completions ↔ unified IR |
| `adapter/openairesp/` | OpenAI Responses ↔ unified IR |
| `stream/` | `SSEReader`/`SSEWriter`, `StreamConverter`, `Pipeline`, `Accumulator` |
| `proxy/` | Production-ready HTTP proxy handler |
| `header/` | Header policy and auth format conversion |
| `middleware/` | Logging hooks for conversion lifecycle |
| `token/` | CJK-aware token estimation and truncation |

## Usage Patterns

### Direct adapter

```go
uniReq, _, _ := claude.New().DecodeRequest(claudeJSON)
openaiJSON, _, _ := openaichat.New().EncodeRequest(uniReq)
```

### Build requests in the unified IR

```go
req := &uni.RequestParams{
    Model: "claude-sonnet-4-20250514",
    Messages: []uni.Message{
        uni.SystemMessage(uni.TextPart{Text: "You are a helpful assistant."}),
        uni.UserMessage(
            uni.TextPart{Text: "Describe this image:"},
            uni.ImagePart{URL: "https://example.com/photo.png", Detail: "high"},
        ),
    },
}
// Encode to any supported protocol
claudeJSON, _, _  := claude.New().EncodeRequest(req)
openaiJSON, _, _  := openaichat.New().EncodeRequest(req)
```

### Stream accumulation

```go
acc := stream.NewAccumulator()
for event := range sseEvents {
    acc.Accumulate(event)
}
complete := acc.Response() // full uni.Response
```

### Header policy

```go
policy := header.NewPolicy().
    WithPassthrough("User-Agent", "X-Request-ID").
    WithMapping("Authorization", "x-api-key").   // Bearer token → x-api-key
    WithStrip("Cookie").
    WithDefault(header.ActionStrip)
```

## Supported Protocols

| Protocol | Adapter | Stream | Error |
|----------|---------|--------|-------|
| Claude Messages | `adapter/claude` | ✅ SSE events | ✅ |
| OpenAI Chat Completions | `adapter/openaichat` | ✅ SSE chunks | ✅ |
| OpenAI Responses | `adapter/openairesp` | ✅ SSE events | ✅ |

## Performance

Apple M2, Go 1.23:

| Operation | Latency | Alloc |
|-----------|---------|-------|
| Single adapter decode + encode | 0.5 – 4.5 μs | 1 – 6 KB |
| Cross-protocol round trip | 5.7 – 8.7 μs | 4 – 7 KB |

```bash
go test -bench=. -benchmem ./adapter/
```

## Vibe Coding

> Vibe coding composition: **>95%**

Tools: Claude Code · OpenCode
Models: Qwen3.7 Plus · DeepSeek V4 Pro

## Contributing

Issues and PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and PR guidelines.

## License

[Apache License 2.0](LICENSE)
