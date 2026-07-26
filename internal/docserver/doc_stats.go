package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DocStats is the docs://stats resource — JSON object with document count,
// total characters, and average characters per document.
type DocStats struct {
	spec *mcp.Resource
}

// NewDocStats builds a configured DocStats feature.
func NewDocStats() *DocStats {
	return &DocStats{
		spec: &mcp.Resource{
			Name:     "stats",
			URI:      "docs://stats",
			MIMEType: "application/json",
		},
	}
}

// Register attaches the resource to s.
func (d *DocStats) Register(s *mcp.Server) {
	s.AddResource(d.spec, d.read)
}

func (d *DocStats) read(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	body, err := statsJSON()
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
