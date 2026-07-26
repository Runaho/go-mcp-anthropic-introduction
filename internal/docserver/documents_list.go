package docserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DocumentsList is the docs://documents resource — a JSON array of all document IDs.
type DocumentsList struct {
	spec *mcp.Resource
}

// NewDocumentsList builds a configured DocumentsList feature.
func NewDocumentsList() *DocumentsList {
	return &DocumentsList{
		spec: &mcp.Resource{
			Name:     "documents",
			URI:      "docs://documents",
			MIMEType: "application/json",
		},
	}
}

// Register attaches the resource to s.
func (d *DocumentsList) Register(s *mcp.Server) {
	s.AddResource(d.spec, d.read)
}

func (d *DocumentsList) read(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	body, err := json.Marshal(listIDs())
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      d.spec.URI,
			MIMEType: d.spec.MIMEType,
			Text:     string(body),
		}},
	}, nil
}
