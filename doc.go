// Package lmm provides protocol conversion between LLM API formats.
//
// # Supported Protocols
//
//   - Claude Messages API (adapter/claude)
//   - OpenAI Chat Completions API (adapter/openaichat)
//   - OpenAI Responses API (adapter/openairesp)
//
// # Quick Start
//
//	import (
//	    "github.com/skys-mission/lmm-adapter-go/adapter"
//	    "github.com/skys-mission/lmm-adapter-go/adapter/claude"
//	    "github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
//	)
//
//	func main() {
//	    c := adapter.NewConverter(
//	        adapter.WithAdapter(claude.New()),
//	        adapter.WithAdapter(openaichat.New()),
//	    )
//
//	    claudeJSON := json.RawMessage(`{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":[{"type":"text","text":"Hello"}]}],"max_tokens":1024}`)
//	    openaiJSON, report, err := c.ConvertRequest(adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, claudeJSON)
//	    _ = openaiJSON
//	    _ = report
//	    _ = err
//	}
//
// # Proxy Server
//
//	import (
//	    "github.com/skys-mission/lmm-adapter-go/adapter"
//	    "github.com/skys-mission/lmm-adapter-go/adapter/claude"
//	    "github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
//	    "github.com/skys-mission/lmm-adapter-go/proxy"
//	)
//
//	func main() {
//	    c := adapter.NewConverter(
//	        adapter.WithAdapter(claude.New()),
//	        adapter.WithAdapter(openaichat.New()),
//	    )
//
//	    handler := proxy.NewHandler(
//	        c,
//	        adapter.ProtocolClaudeMessages,
//	        adapter.ProtocolOpenAIChat,
//	        "https://api.openai.com/v1/chat/completions",
//	    )
//
//	    http.Handle("/v1/messages", handler)
//	    http.ListenAndServe(":8080", nil)
//	}
//
// # Streaming
//
//	import (
//	    "github.com/skys-mission/lmm-adapter-go/stream"
//	)
//
//	func handleStream(w http.ResponseWriter, r *http.Request) {
//	    pipeline := stream.NewPipeline(srcAdapter, dstAdapter)
//	    pipeline.Pipe(r.Context(), r.Body, w)
//	}
//
// # Stream Accumulator
//
//	acc := stream.NewAccumulator()
//	for {
//	    event, err := sseReader.Read()
//	    if err == io.EOF { break }
//	    acc.Accumulate(uniEvent)
//	}
//	response := acc.Response()
//
// # Error Conversion
//
//	errResp, report, err := c.ConvertError(
//	    adapter.ProtocolOpenAIChat,
//	    adapter.ProtocolClaudeMessages,
//	    errorBody,
//	    httpStatusCode,
//	)
//
// # Token Estimation
//
//	import "github.com/skys-mission/lmm-adapter-go/token"
//
//	tokens := token.EstimateTokens("Hello, world!")
//	reqTokens := token.EstimateRequestTokens(uniReq)
//
// # Architecture
//
// # Protocol A ←→ Adapter A ←→ Unified IR ←→ Adapter B ←→ Protocol B
//
// Each adapter implements bidirectional conversion between its protocol format
// and the unified intermediate representation (IR). Cross-protocol conversion
// is achieved by chaining: decode from source protocol to IR, then encode from
// IR to target protocol.
//
// # Package Structure
//
//	uni/          Unified intermediate representation types
//	adapter/      Adapter interfaces, Converter, Report
//	  claude/     Claude Messages API adapter
//	  openaichat/ OpenAI Chat Completions adapter
//	  openairesp/ OpenAI Responses API adapter
//	stream/       SSE reader/writer, StreamConverter, Pipeline, Accumulator
//	proxy/        HTTP handler for proxy scenarios
//	header/       Header policy (passthrough, mapping, strip, auth)
//	middleware/   Logging hook interface
//	token/        Token count estimation
package lmm
