// Verifies docs/13-third-party-libraries/18-mcp-servers-with-mcp-go.md
package mcpdemo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type Server struct{ srv *mcpserver.MCPServer }

func searchTool() mcpgo.Tool {
	return mcpgo.NewTool("search",
		mcpgo.WithDescription("Search the knowledge base."),
		mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("What to search for")),
		mcpgo.WithNumber("limit", mcpgo.Description("Max results, default 10")),
	)
}

func argString(req mcpgo.CallToolRequest, key string) string {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := args[key].(string)
	return v
}

func toolError(msg string) *mcpgo.CallToolResult { return mcpgo.NewToolResultError(msg) }

func toolJSON(v any) *mcpgo.CallToolResult {
	b, err := json.Marshal(v)
	if err != nil {
		return toolError("encoding result: " + err.Error())
	}
	return mcpgo.NewToolResultText(string(b))
}

func (s *Server) handleSearch(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	q := argString(req, "query")
	if q == "" {
		return toolError("query is required"), nil
	}
	return toolJSON(map[string]any{"query": q, "hits": []string{"a", "b"}}), nil
}

func (s *Server) tracked(name string, h mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		start := time.Now()
		res, err := h(ctx, req)
		_ = time.Since(start) // a real wrapper records this
		return res, err
	}
}

func call(args map[string]any) mcpgo.CallToolRequest {
	var req mcpgo.CallToolRequest
	req.Params.Name = "search"
	req.Params.Arguments = args
	return req
}

func text(t *testing.T, res *mcpgo.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatalf("no text content in %+v", res)
	return ""
}

func TestToolSchema(t *testing.T) {
	b, err := json.Marshal(searchTool().InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	schema := string(b)
	for _, want := range []string{
		`"required":["query"]`,
		`"type":"object"`,
		`"query":{"description":"What to search for","type":"string"}`,
		`"limit":{"description":"Max results, default 10","type":"number"}`,
	} {
		if !strings.Contains(schema, want) {
			t.Errorf("schema missing %s\ngot: %s", want, schema)
		}
	}
}

func TestHandlerHappyPath(t *testing.T) {
	s := &Server{}
	res, err := s.tracked("search", s.handleSearch)(context.Background(), call(map[string]any{"query": "go"}))
	if err != nil {
		t.Fatalf("Go error = %v, want nil", err)
	}
	if res.IsError {
		t.Error("IsError = true, want false")
	}
	if got := text(t, res); got != `{"hits":["a","b"],"query":"go"}` {
		t.Errorf("content = %s", got)
	}
}

// The key idiom: a validation failure is data for the model, not a Go
// error that aborts the call.
func TestValidationFailureIsDataNotError(t *testing.T) {
	s := &Server{}
	res, err := s.handleSearch(context.Background(), call(map[string]any{}))
	if err != nil {
		t.Fatalf("Go error = %v, want nil — a bad argument must not be a transport failure", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true")
	}
	if got := text(t, res); got != "query is required" {
		t.Errorf("message = %q, want an actionable one", got)
	}
}

func TestRegistrationAcceptsWrappedHandler(t *testing.T) {
	srv := mcpserver.NewMCPServer("demo", "1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithInstructions("A demo server."),
	)
	s := &Server{srv: srv}
	srv.AddTool(searchTool(), s.tracked("search", s.handleSearch))
}
