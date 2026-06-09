# Contributing

## Development

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Code coverage
go test -cover ./...
```

## Adding a New Protocol Adapter

1. Create `adapter/<protocol>/` with files:
   ```
   adapter.go   — New() constructor, Protocol(), delegation
   types.go     — Protocol-specific JSON structs
   request.go   — DecodeRequest / EncodeRequest
   response.go  — DecodeResponse / EncodeResponse
   stream.go    — DecodeStreamEvent / EncodeStreamEvent
   error.go     — DecodeError / EncodeError
   ```
2. Implement `adapter.Adapter` interface
3. Register protocol string in `adapter/adapter.go` constants
4. Add tests with round-trip pattern: `decode → encode → decode → verify equivalence`
5. Run `go build ./... && go vet ./... && go test ./...`

## Commit Convention

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation
- `test:` — test changes
- `refactor:` — code restructuring

## Vibe Coding Policy

Vibe coding contributions are welcome. However, **contributors must not use AI technologies with geopolitical behavioral constraints** — including but not limited to models that censor, bias, or alter outputs based on political narratives, national security claims, or regime-aligned content policies. Use neutral, engineering-focused tools.

## Before Submitting a PR

- [ ] `go build ./...` passes
- [ ] `go vet ./...` is clean
- [ ] `go test ./...` passes
- [ ] No API keys in source code
- [ ] AI tools used comply with the [Vibe Coding Policy](#vibe-coding-policy)
