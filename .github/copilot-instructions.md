# Copilot instructions for imap-mcp-go

## Build, test, and lint commands

- Build: `go build ./...`
- Full tests: `go test ./...`
- Package tests during iteration: `go test ./internal/...`
- Single test example: `go test ./internal/imapclient -run TestExtractMessageBodyPrefersPlainText`
- Format changed Go files with `gofmt -w <paths>` before finishing work

There is no dedicated lint command in the repository at the moment.

## High-level architecture

The repository now implements the README-defined structure:

- `cmd/server/main.go`: stdio MCP server entrypoint that reads environment configuration and starts the server.
- `internal/config/config.go`: environment-variable loading and validation.
- `internal/mcpserver/tools.go`: MCP tool registration and thin handlers.
- `internal/imapclient/`: IMAP behavior split by concern:
  - `client.go`: connection, authentication, and folder selection
  - `list.go`: `list_unread` / `list_read`
  - `search.go`: `search_by_sender`
  - `body.go`: `get_message_body` fetch and MIME parsing
  - `mutate.go`: `mark_read` and `move_to_folder`

Keep the MCP layer thin. Tool handlers should delegate to `internal/imapclient` rather than embedding IMAP protocol logic in the server package.

## Key conventions

- This project is intentionally **security-first and narrow in scope**: no SMTP/send support, no delete tool, and no bulk mutation tool. Mutating actions should stay limited to a single message UID per call.
- Preserve a strict separation between **read** and **act** flows. The server should never chain a `get_message_body` result into a mutation automatically; all writes must come from an explicit tool call.
- `get_message_body` is the only tool that returns attacker-controlled free text. Returned content should be clearly framed as untrusted, prefer `text/plain`, and only fall back to stripped/sanitized HTML.
- All mailbox-oriented tools should accept an optional `folder` parameter and default to `INBOX`.
- `list_unread`, `list_read`, and `search_by_sender` are expected to support pagination from the start; avoid unbounded list responses.
- `move_to_folder` should use the go-imap v2 MOVE path, which already falls back to `COPY` + `STORE \\Deleted` + `EXPUNGE` when the server lacks MOVE support.
- Configuration is environment-only. The current implementation uses `IMAP_HOST`, `IMAP_PORT`, `IMAP_USER`, `IMAP_PASSWORD`, `IMAP_TLS`, optional `IMAP_DEFAULT_FOLDER`, and optional `IMAP_AUDIT_LOG`. Do not hardcode mailbox credentials.
- Mutations are auditable: `mark_read` and `move_to_folder` append JSON-line entries to the local audit log.
- The IMAP client is initialized with `go-message/charset` support so subject lines and address display names can decode non-UTF-8 encoded words.
