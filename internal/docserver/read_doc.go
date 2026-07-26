package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadDoc returns the contents of a single document. Read-only.
type ReadDoc struct {
	spec *mcp.Tool
}

// NewReadDoc builds a configured ReadDoc feature.
func NewReadDoc() *ReadDoc {
	return &ReadDoc{
		spec: &mcp.Tool{
			Name:        "read_doc_contents",
			Description: "Read the contents of a document and return it as a string.",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
	}
}

// Register attaches the tool to s.
func (r *ReadDoc) Register(s *mcp.Server) {
	mcp.AddTool(s, r.spec, r.handle)
}

type readArgs struct {
	DocID string `json:"doc_id" jsonschema:"Id of the document to read"`
}

func (r *ReadDoc) handle(_ context.Context, _ *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, any, error) {
	body, ok := getDoc(in.DocID)
	if !ok {
		return toolErr(in.DocID), nil, nil
	}
	return textResult(body), nil, nil
}
