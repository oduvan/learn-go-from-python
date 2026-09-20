# MCP servers with mcp-go

The Model Context Protocol lets an AI assistant call your code. You
expose tools; a client discovers them, decides when to call them, and
feeds the results back to the model.

> **Module:** `github.com/mark3labs/mcp-go`.

```go
srv := mcpserver.NewMCPServer("demo", "1.0.0",
    mcpserver.WithToolCapabilities(true),
)
srv.AddTool(searchTool(), s.handleSearch)
```

## What MCP is

A JSON-RPC protocol between a **client** (Claude Desktop, an IDE, an
agent) and a **server** (your process). The server offers three kinds
of thing:

| Primitive | Meaning |
|---|---|
| **Tools** | functions the model may call, with a JSON Schema |
| **Resources** | readable content the client can fetch |
| **Prompts** | reusable prompt templates the user can invoke |

Tools are what most servers implement, and what this article covers.
The model chooses when to call one, so the description and schema are
not documentation — they are the interface the model reasons about.

## A tool is a schema

```go
func searchTool() mcpgo.Tool {
    return mcpgo.NewTool("search",
        mcpgo.WithDescription("Search the knowledge base."),
        mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("What to search for")),
        mcpgo.WithNumber("limit", mcpgo.Description("Max results, default 10")),
    )
}
```

That produces JSON Schema:

```json
{"properties":{"limit":{"description":"Max results, default 10","type":"number"},
 "query":{"description":"What to search for","type":"string"}},
 "required":["query"],"type":"object"}
```

**Write the descriptions carefully.** They are the only thing telling
the model what a tool does and when to use it. "Search" is useless;
"Search the knowledge base by keyword; returns up to `limit` matching
documents with their ids" produces far better tool selection. Say what
it returns and when *not* to use it.

## Separate definition from behaviour

A convention worth adopting early: schemas in one file, handlers in
another.

```
internal/mcp/
  tools_search.go      → func searchTool() mcpgo.Tool
  handle_search.go     → func (s *Server) handleSearch(...)
  server.go            → registration
```

Once there are more than a handful of tools, a file mixing both is
unreadable — and the schema is the part you re-read most, since it is
what the model sees.

## Handlers, and the untyped argument map

```go
func (s *Server) handleSearch(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
    q := argString(req, "query")
    if q == "" {
        return toolError("query is required"), nil
    }
    return toolJSON(map[string]any{"query": q, "hits": []string{"a", "b"}}), nil
}
```

Arguments arrive as `map[string]any`, so every handler needs
accessors. Write them once:

```go
func argString(req mcpgo.CallToolRequest, key string) string {
    args, ok := req.Params.Arguments.(map[string]any)
    if !ok {
        return ""
    }
    v, _ := args[key].(string)
    return v
}
```

The comma-ok assertion returns `""` for both "absent" and "wrong
type" — so if you need to tell those apart, write a second accessor
returning `(string, bool)`. Numbers arrive as `float64`, exactly as in
[encoding JSON](../06-text-time-and-data/07-encoding-json.md).

## The key idiom: errors are data

This is the thing to internalise.

```go
res, err := s.handleSearch(ctx, req)
// Go err: <nil>   isError: true   content: query is required
```

A validation failure returns `(toolError(msg), nil)` — a result marked
as an error, with a **nil** Go error. The message goes back to the
model, which can read it and try again with a better argument.

Return a non-nil Go error only for a genuine transport or protocol
failure. Doing it for a bad argument turns something the model could
have fixed into a broken call.

The corollary: write tool error messages **for the model**. "query is
required" is actionable. "invalid input" is not.

```go
func toolError(msg string) *mcpgo.CallToolResult {
    return mcpgo.NewToolResultError(msg)
}

func toolJSON(v any) *mcpgo.CallToolResult {
    b, err := json.Marshal(v)
    if err != nil {
        return toolError("encoding result: " + err.Error())
    }
    return mcpgo.NewToolResultText(string(b))
}
```

Funnel every result through two or three helpers like these, and the
output shape stays consistent across a hundred tools.

## Wrapping every tool

Registration is the place to add cross-cutting behaviour, using the
decorator shape from
[middleware](../08-http-with-net-http/03-middleware.md):

```go
func (s *Server) tracked(name string, h mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
    return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
        ctx, span := tracer.Start(ctx, "mcp."+name)
        defer span.End()

        start := time.Now()
        res, err := h(ctx, req)
        s.hist.Record(ctx, time.Since(start).Seconds())
        return res, err
    }
}

srv.AddTool(searchTool(), s.tracked("search", s.handleSearch))
```

```
[tracked] tool=search dur<1s=true err=<nil>
```

One wrapper gives every tool tracing, metrics, logging and an audit
trail. Add a `recover` in there too — a panic in a handler otherwise
takes the server down.

## Transports

```go
// stdio: the client launches your binary
mcpserver.NewStdioServer(srv).ServeStdio(ctx)

// HTTP: a long-running service
mcpserver.NewStreamableHTTPServer(srv)
```

**stdio** is how a desktop client runs a local server: it spawns your
process and talks over stdin and stdout. Which means **nothing else
may write to stdout** — a stray `fmt.Println` corrupts the protocol
stream. Send logs to stderr, and configure `slog` accordingly.

**Streamable HTTP** suits a shared, deployed server. It mounts as a
handler, so it sits alongside your existing routes.

A `--mode` flag switching between them lets one binary do both.

## Authentication

stdio inherits the trust of whoever launched the process. HTTP does
not, and an MCP endpoint is an API that executes code on request.

At minimum require a bearer token. For a multi-user deployment,
OAuth 2.1 with the discovery endpoints is the specified approach, built
on the pieces from [OIDC and OAuth](17-oidc-and-oauth.md).

And design the tools defensively. A model can be talked into calling
things — so a tool that runs caller-supplied SQL should parse and
restrict it, one that writes should be scoped, and destructive
operations should not be exposed at all.

## Designing tools the model can use

- **Few, broad tools beat many narrow ones.** A model choosing among
  twenty tools does better than among two hundred.
- **Return structured JSON**, not prose. The model parses it.
- **Keep responses small.** Everything returned enters the context
  window; paginate, and return ids the model can follow up on.
- **Make them idempotent** where possible. A retried call should be
  safe.
- **Say what a tool does not do.** Preventing a wrong call is as
  valuable as enabling a right one.

> **From Python:** the protocol is identical and the official Python
> SDK is more magical — decorators inferring the schema from type
> hints. Here you declare the schema explicitly, which is more typing
> and leaves no doubt about what the model is being shown.

## Quick reference

| Task | Form |
|---|---|
| a server | `mcpserver.NewMCPServer(name, version, opts...)` |
| a tool schema | `mcpgo.NewTool(name, WithDescription, WithString(...))` |
| required argument | `mcpgo.Required()` |
| register | `srv.AddTool(def, handler)` |
| handler | `func(ctx, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error)` |
| read arguments | assert `req.Params.Arguments.(map[string]any)` |
| **a bad argument** | `return toolError(msg), nil` — nil Go error |
| a real failure | a non-nil Go error |
| results | one `toolJSON`/`toolText` helper, used everywhere |
| cross-cutting concerns | wrap the handler at registration |
| local client | `ServeStdio` — **nothing on stdout but the protocol** |
| deployed | `NewStreamableHTTPServer`, behind auth |

## Sources

- [Model Context Protocol — modelcontextprotocol.io](https://modelcontextprotocol.io/)
- [MCP specification — spec.modelcontextprotocol.io](https://spec.modelcontextprotocol.io/)
- [`mcp-go` — pkg.go.dev/github.com/mark3labs/mcp-go](https://pkg.go.dev/github.com/mark3labs/mcp-go)
- [mcp-go repository — github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [MCP transports — modelcontextprotocol.io/docs/concepts/transports](https://modelcontextprotocol.io/docs/concepts/transports)
