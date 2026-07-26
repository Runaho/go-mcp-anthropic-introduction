package docserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Translate is the `translate` prompt — asks the model to translate a document
// into a target language.
type Translate struct {
	spec *mcp.Prompt
}

// NewTranslate builds a configured Translate feature.
func NewTranslate() *Translate {
	return &Translate{
		spec: &mcp.Prompt{
			Name:        "translate",
			Description: "Translates the document into a target language.",
			Arguments: []*mcp.PromptArgument{
				{Name: "doc_id", Required: true, Description: "Id of the document to translate"},
				{Name: "target_language", Required: true, Description: "Target language (e.g. Turkish, French)"},
			},
		},
	}
}

// Register attaches the prompt to s.
func (t *Translate) Register(s *mcp.Server) {
	s.AddPrompt(t.spec, t.get)
}

func (t *Translate) get(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	lang := req.Params.Arguments["target_language"]
	return userPrompt(
		"Please translate the following document into " + lang + ".\n\n" +
			"Document id: " + docID + "\n\n" +
			"Read the document with the 'read_doc_contents' tool, then respond with the full translation.\n" +
			"Preserve technical terms, names, and formatting (Markdown, lists) where possible.\n"), nil
}
