package docserver

import (
	"context"
	"path"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Document is the docs://documents/{doc_id} resource template — text body of one
// document, addressed by the doc_id path segment.
type Document struct {
	spec *mcp.ResourceTemplate
}

// NewDocument builds a configured Document feature.
func NewDocument() *Document {
	return &Document{
		spec: &mcp.ResourceTemplate{
			Name:        "document",
			URITemplate: "docs://documents/{doc_id}",
			MIMEType:    "text/plain",
		},
	}
}

// Register attaches the resource template to s.
func (d *Document) Register(s *mcp.Server) {
	s.AddResourceTemplate(d.spec, d.read)
}

func (d *Document) read(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := path.Base(req.Params.URI)
	body, ok := getDoc(id)
	if !ok {
		return nil, errMissing(id)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: d.spec.MIMEType,
			Text:     body,
		}},
	}, nil
}
