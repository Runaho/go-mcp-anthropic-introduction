package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListDocsMeta returns aggregate stats (count, total chars, average chars) for
// the document store. Read-only.
type ListDocsMeta struct {
	spec *mcp.Tool
}

// NewListDocsMeta builds a configured ListDocsMeta feature.
func NewListDocsMeta() *ListDocsMeta {
	return &ListDocsMeta{
		spec: &mcp.Tool{
			Name:        "list_docs_meta",
			Description: "Return aggregate stats (count, total chars, average chars) for all documents.",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
	}
}

// Register attaches the tool to s.
func (l *ListDocsMeta) Register(s *mcp.Server) {
	mcp.AddTool(s, l.spec, l.handle)
}

func (l *ListDocsMeta) handle(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	out, err := statsJSON()
	if err != nil {
		return nil, nil, err
	}
	return textResult(string(out)), nil, nil
}
