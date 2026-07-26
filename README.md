# DocumentMCP — Go MCP Server

In-memory document store exposed over [Model Context Protocol](https://modelcontextprotocol.io) stdio transport. Go rewrite of the Python `mcp_server.py` in `archive_cli_project_COMPLETE/`, with extra MCP features.

## Layout

```
go-mcp-anthropic-introduction/
├── main.go                          # stdio entry point
├── internal/docserver/              # each feature is a self-registering type
│   ├── server.go                       # New() — registers every feature
│   ├── store.go                        # docs map, mutex, accessors
│   ├── helpers.go                      # textResult, toolErr, replaceAll, etc.
│   ├── completion.go                   # MCP completion handler
│   ├── read_doc.go                     # ReadDoc tool
│   ├── edit_doc.go                     # EditDoc tool
│   ├── list_docs_meta.go               # ListDocsMeta tool
│   ├── documents_list.go               # DocumentsList resource
│   ├── document.go                     # Document resource template
│   ├── doc_stats.go                    # DocStats resource
│   ├── format_prompt.go                # Format prompt
│   ├── summarize_prompt.go             # Summarize prompt
│   ├── translate_prompt.go             # Translate prompt
│   └── docserver_test.go               # 19 unit tests (in-memory transport)
├── go.mod / go.sum
├── Makefile                         # build / run / inspector / test / tidy / clean
├── REFACTOR_PLAN.md                 # migration plan & rationale
└── archive_cli_project_COMPLETE/    # Python arşivi (dokunulmaz)
```

### Feature pattern

Every tool, resource, and prompt follows the same shape — a struct that holds
its own `*mcp.Tool` / `*mcp.Resource` / `*mcp.Prompt` spec, plus a `Register`
method that wires it into a server. `server.go` doesn't know what's inside
any feature; it only calls `New*().Register(s)`.

```go
// read_doc.go (representative)
type ReadDoc struct{ spec *mcp.Tool }

func NewReadDoc() *ReadDoc { return &ReadDoc{spec: &mcp.Tool{Name: "read_doc_contents", ...}} }
func (r *ReadDoc) Register(s *mcp.Server) { mcp.AddTool(s, r.spec, r.handle) }
func (r *ReadDoc) handle(_ context.Context, _ *mcp.CallToolRequest, in readArgs) (...) { ... }
```

Adding a new feature = new file with one type, no changes to `server.go`
required unless you want it listed in `New()`.

## Capabilities

### Tools (with annotations)

| Name | Read-only | Destructive | Description |
|---|---|---|---|
| `read_doc_contents` | ✓ | — | Read a document's contents |
| `edit_document` | — | ✓ | Replace a string in a document |
| `list_docs_meta` | ✓ | — | Return count, total chars, average chars |

### Resources

| URI | Type | Description |
|---|---|---|
| `docs://documents` | `application/json` | List of all document IDs |
| `docs://documents/{doc_id}` | `text/plain` | Body of a specific document |
| `docs://stats` | `application/json` | Aggregate document stats |

### Prompts

| Name | Arguments | Description |
|---|---|---|
| `format` | `doc_id` (required) | Reformat doc as Markdown |
| `summarize` | `doc_id` (required) | 3-5 sentence summary |
| `translate` | `doc_id`, `target_language` (required) | Translate doc to target language |

### Server

- **Instructions:** "DocumentMCP — in-memory document store. Use the tools and resources to read, edit, and inspect documents. Use prompts (format, summarize, translate) to give the model a structured starting point."
- **Version:** 0.3.0
- **Transport:** stdio (newline-delimited JSON)

## Build & Run

```bash
make build               # ./docserver
make run                 # stdio transport
make test                # 20 unit tests
make inspector           # npx @modelcontextprotocol/inspector ./docserver
make inspector-headless  # BROWSER=none — UI link printed to terminal
make inspector-stop      # kill any leftover inspector / vite / proxy
make tidy                # go mod tidy
make clean               # rm docserver
```

### Inspector modes

`make inspector` opens the official MCP Inspector in your browser. There are two ports it uses:

| Port | Role |
|---|---|
| `6277` | Proxy server (between UI and the stdio server) |
| `6274` | Web UI (Vite dev server) |

**Default (`make inspector`)** — spawns `./docserver` as a child process, opens `http://localhost:6274` in your default browser. Both targets set `DANGEROUSLY_OMIT_AUTH=true` so the proxy skips the per-session auth token check (see the warning it prints: "Authentication is disabled. This is not recommended."). This is fine for local dev — never set this on a shared network.

**Headless (`make inspector-headless`)** — same setup, but `BROWSER=none` is set so no browser window opens. The plain URL is printed to the terminal:

```
🚀 MCP Inspector is up and running at:
   http://localhost:6274
🌐 Opening browser...   ← skipped because BROWSER=none
```

Just open `http://localhost:6274` in any browser. Useful over SSH, in containers, or when the default browser is something you don't want spawned (e.g. over a remote desktop session).

**Stop (`make inspector-stop`)** — kills any leftover inspector, Vite, or proxy processes still holding `6274` or `6277`. Run this if you get `Proxy Server PORT IS IN USE at port 6277 ❌`.

Once the UI is open:

- **Tools** tab: list, call `read_doc_contents` with `doc_id: "deposition.md"`, edit, etc.
- **Resources** tab: read `docs://documents`, `docs://stats`, or any `docs://documents/{id}`.
- **Prompts** tab: pick a prompt (e.g. `translate`), fill `doc_id` and `target_language`, submit.

## Tests

```bash
$ go test -v ./...
=== RUN   TestToolsList              --- PASS
=== RUN   TestToolAnnotations        --- PASS  (read-only, destructive hints)
=== RUN   TestReadDoc                --- PASS
=== RUN   TestReadDocMissing         --- PASS  (IsError: true)
=== RUN   TestEditDoc                --- PASS
=== RUN   TestListDocsMeta           --- PASS  (stats tool)
=== RUN   TestListResources          --- PASS
=== RUN   TestListResourcesContent   --- PASS
=== RUN   TestStatsResource          --- PASS  (docs://stats)
=== RUN   TestReadResourceTemplate   --- PASS
=== RUN   TestListPrompts            --- PASS
=== RUN   TestPromptArguments        --- PASS  (required args)
=== RUN   TestGetFormatPrompt        --- PASS
=== RUN   TestGetSummarizePrompt     --- PASS
=== RUN   TestGetTranslatePrompt     --- PASS
=== RUN   TestCompleteResourceDocID  --- PASS  (completion for resource URI)
=== RUN   TestCompletePromptDocID    --- PASS  (completion for prompt arg)
=== RUN   TestCompleteTargetLanguage --- PASS  (completion for language list)
=== RUN   TestCompleteEmptyPrefix    --- PASS  (empty prefix returns all)
PASS
```

Tests use the SDK's `NewInMemoryTransports` — no subprocess, no Inspector, no network.

## Why Go

- Single static binary, no Python toolchain
- Resmi MCP SDK (`modelcontextprotocol/go-sdk`) → spec compliance, type-safe generics
- Stdlib `slog` for stderr, stdlib `sync.RWMutex` for shared state
- ~250 → ~340 lines of code total; 1 production dependency

## See also

- [Introduction to Model Context Protocol by Anthropic (LinkedIn Learning)](https://www.linkedin.com/learning/introduction-to-model-context-protocol-by-anthropic/welcome-to-the-course) — the Python version of the same project. This repo is the Go equivalent: same tools (`read_doc_contents`, `edit_document`), same resources (`docs://documents`, `docs://documents/{doc_id}`), same prompt (`format`), plus a few extra (summarize, translate, list_docs_meta, docs://stats, completion handler). Watch the course for the protocol fundamentals and CLI design; read the Go code for the idiomatic translation.
- `REFACTOR_PLAN.md` — decision log and rejected alternatives
- `archive_cli_project_COMPLETE/mcp_server.py` — original Python implementation (untouched)
roject_COMPLETE/mcp_server.py` — original Python implementation (untouched)
