# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands
- Build: `go build ./cmd/proxy`
- Run: `./start.sh` or `./proxy --blacklist=./cmd/proxy/blacklist.txt`
- Test: `go test ./...`
- Test specific: `go test ./path/to/package -run TestName`

## Code Style Guidelines
- Formatting: Use `gofmt` or `go fmt ./...` to format code
- Imports: Group standard library, then third-party, then local imports
- Error handling: Always check errors and log or return them appropriately
- Logging: Use the internal logger package (`logger.Log()`)
- Naming: Use CamelCase for exported names, camelCase for unexported
- Files: Keep files under 500 lines; one package per directory
- Comments: Document all exported functions, types, and constants
- Error handling pattern: `if err := funcCall(); err != nil { ... }`
- Context: Use context for handling timeouts and cancellation in new code