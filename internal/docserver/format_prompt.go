package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Format is the `format` prompt — asks the model to reformat a document as Markdown.
type Format struct {
	spec *mcp.Prompt
}

// NewFormat builds a configured Format feature.
func NewFormat() *Format {
	return &Format{
		spec: &mcp.Prompt{
			Name:        "format",
			Description: "Rewrites the contents of the document in Markdown format.",
			Arguments: []*mcp.PromptArgument{
				{Name: "doc_id", Required: true, Description: "Id of the document to format"},
			},
		},
	}
}

// Register attaches the prompt to s.
func (f *Format) Register(s *mcp.Server) {
	s.AddPrompt(f.spec, f.get)
}

func (f *Format) get(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	return userPrompt(
		"Your goal is to reformat a document to be written with markdown syntax.\n\n" +
			"The id of the document you need to reformat is:\n<document_id>\n" + docID + "\n</document_id>\n\n" +
			"Add in headers, bullet points, tables, etc as necessary. Feel free to add in extra text, but don't change the meaning of the report.\n" +
			"Use the 'edit_document' tool to edit the document. After the document has been edited, respond with the final version of the doc. Don't explain your changes.\n"), nil
}
