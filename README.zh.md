# lmm-adapter-go

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/skys-mission/lmm-adapter-go)](https://goreportcard.com/report/github.com/skys-mission/lmm-adapter-go)

[English](README.md)

LLM API 协议转换库 —— 通过类型安全的统一中间表示，在 Claude Messages、OpenAI Chat Completions 和 OpenAI Responses 之间双向转换。**零外部依赖。**

## 为什么需要它？

构建 LLM 应用意味着要同时对接多种 API 格式。每个提供商都有自己的一套请求/响应结构、流式协议和错误格式。lmm-adapter-go 提供单一、类型安全的内部 API —— 业务逻辑只写一次，底层 LLM 随意切换。

```
┌──────────────────────────────────────────────────────┐
│                     你的应用程序                      │
├──────────────────────────────────────────────────────┤
│                  统一 IR (uni/)                       │
├──────────┬──────────────────────┬───────────────────┤
│  Claude  │   OpenAI Chat        │  OpenAI Responses  │
│ Messages │   Completions        │                    │
└──────────┴──────────────────────┴───────────────────┘
```

## 安装

```bash
go get github.com/skys-mission/lmm-adapter-go
```

## 快速开始

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

// Claude 格式请求 → OpenAI Chat 格式
openaiReq, report, _ := c.ConvertRequest(
    adapter.ProtocolClaudeMessages,
    adapter.ProtocolOpenAIChat,
    claudeReqJSON,
)

// 查看转换过程中丢失了哪些字段
for _, f := range report.LostFields {
    log.Printf("丢失字段: %s.%s — %s", f.Source, f.Field, f.Reason)
}
```

### 一行代码启动代理

```go
// 对外暴露 /v1/messages（Claude 兼容接口），
// 背后转发到 OpenAI 并自动转换格式
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

流式传输、Auth 转换、错误映射全部自动处理。

## 特性

### 核心
- **Claude Messages**、**OpenAI Chat Completions**、**OpenAI Responses** 三协议双向转换
- 类型安全的统一 IR，**9 种 sealed ContentPart 类型** —— 穷举类型切换，不会漏
- 请求、响应、错误、流式事件全覆盖
- **转换质量报告** —— 丢失字段和警告，绝不静默丢弃数据

### 流式处理
- 标准 SSE 读写器
- 有状态 `StreamConverter`，自动补全缺失事件（start、role、content-stop）
- `Accumulator` 将流式增量收集为完整响应
- `Pipeline` 串联全流程：SSE 输入 → 转换 → SSE 输出

### 代理
- `POST`-only HTTP handler，自动检测流式请求
- 错误响应（4xx/5xx）自动转换回客户端协议
- Header 策略：**透传**、**映射**、**移除**、**Auth 转换**（Bearer ↔ x-api-key）
- 丢失字段概要通过 `X-Conversion-Lost-Fields` 响应头返回

### 质量
- **零外部依赖** —— 仅用标准库
- Token 估算，区分中/日/韩文字符比例
- 结构化日志 Hook（`middleware.ConversionHook`）
- `go build ./...` / `go vet ./...` 零警告，11 个包测试通过

## 包结构

| 包 | 职责 |
|----|------|
| `uni/` | 统一中间表示：`ContentPart`、`Message`、`RequestParams`、`Response` 等 |
| `adapter/` | `Converter` 注册中心、`Adapter` 及子接口、`Report` |
| `adapter/claude/` | Claude Messages ↔ 统一 IR |
| `adapter/openaichat/` | OpenAI Chat Completions ↔ 统一 IR |
| `adapter/openairesp/` | OpenAI Responses ↔ 统一 IR |
| `stream/` | `SSEReader`/`SSEWriter`、`StreamConverter`、`Pipeline`、`Accumulator` |
| `proxy/` | 生产就绪 HTTP 代理 handler |
| `header/` | Header 策略和 Auth 格式转换 |
| `middleware/` | 转换生命周期的日志 Hook |
| `token/` | 中/日/韩感知的 Token 估算和截断 |

## 使用模式

### 直接使用适配器

```go
uniReq, _, _ := claude.New().DecodeRequest(claudeJSON)
openaiJSON, _, _ := openaichat.New().EncodeRequest(uniReq)
```

### 用统一 IR 构建请求

```go
req := &uni.RequestParams{
    Model: "claude-sonnet-4-20250514",
    Messages: []uni.Message{
        uni.SystemMessage(uni.TextPart{Text: "你是一个有用的助手。"}),
        uni.UserMessage(
            uni.TextPart{Text: "描述这张图片："},
            uni.ImagePart{URL: "https://example.com/photo.png", Detail: "high"},
        ),
    },
}
// 编码为任意支持的协议
claudeJSON, _, _  := claude.New().EncodeRequest(req)
openaiJSON, _, _  := openaichat.New().EncodeRequest(req)
```

### 流式累积

```go
acc := stream.NewAccumulator()
for event := range sseEvents {
    acc.Accumulate(event)
}
complete := acc.Response() // 完整 uni.Response
```

### Header 策略

```go
policy := header.NewPolicy().
    WithPassthrough("User-Agent", "X-Request-ID").
    WithMapping("Authorization", "x-api-key").   // Bearer token → x-api-key
    WithStrip("Cookie").
    WithDefault(header.ActionStrip)
```

## 支持的协议

| 协议 | 适配器 | 流式 | 错误 |
|------|--------|------|------|
| Claude Messages | `adapter/claude` | ✅ SSE 事件 | ✅ |
| OpenAI Chat Completions | `adapter/openaichat` | ✅ SSE 分片 | ✅ |
| OpenAI Responses | `adapter/openairesp` | ✅ SSE 事件 | ✅ |

## 性能

Apple M2 / Go 1.23：

| 操作 | 延迟 | 内存 |
|------|------|------|
| 单适配器解码 + 编码 | 0.5 – 4.5 μs | 1 – 6 KB |
| 跨协议往返转换 | 5.7 – 8.7 μs | 4 – 7 KB |

```bash
go test -bench=. -benchmem ./adapter/
```

## Vibe Coding

> Vibe coding 成分：**>95%**

工具：Claude Code · OpenCode
模型：Qwen3.7 Plus · DeepSeek V4 Pro

## 贡献

欢迎 Issue 和 PR。开发环境和 PR 规范见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可证

[Apache License 2.0](LICENSE)
