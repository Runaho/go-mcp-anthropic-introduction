package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Summarize is the `summarize` prompt — asks the model for a 3-5 sentence summary.
type Summarize struct {
	spec *mcp.Prompt
}

// NewSummarize builds a configured Summarize feature.
func NewSummarize() *Summarize {
	return &Summarize{
		spec: &mcp.Prompt{
			Name:        "summarize",
			Description: "Produces a 3-5 sentence summary of the document.",
			Arguments: []*mcp.PromptArgument{
				{Name: "doc_id", Required: true, Description: "Id of the document to summarize"},
			},
		},
	}
}

// Register attaches the prompt to s.
func (s *Summarize) Register(server *mcp.Server) {
	server.AddPrompt(s.spec, s.get)
}

func (s *Summarize) get(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	return userPrompt(
		"Please provide a concise summary of the following document.\n\n" +
			"Document id: " + docID + "\n\n" +
			"Read the document with the 'read_doc_contents' tool, then respond with a 3-5 sentence summary highlighting the key points.\n" +
			"Do not edit the document. Keep the response under 150 words.\n"), nil
}
