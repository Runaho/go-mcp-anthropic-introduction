package docserver

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// languages is the completion pool for the `translate` prompt's target_language arg.
var languages = []string{
	"English", "French", "German", "Italian", "Japanese", "Spanish", "Turkish",
}

// complete is the MCP completion handler. It serves three reference kinds:
//
//   - ref/resource on docs://documents/{doc_id}     -> doc_id prefix match
//   - ref/prompt with arg doc_id                    -> doc_id prefix match
//   - ref/prompt with arg target_language           -> language list
func complete(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if req.Params.Ref == nil {
		return &mcp.CompleteResult{}, nil
	}
	prefix := strings.ToLower(req.Params.Argument.Value)
	var pool []string

	switch {
	case req.Params.Ref.Type == "ref/resource" && strings.HasPrefix(req.Params.Ref.URI, "docs://documents/{doc_id}"):
		pool = listIDs()
	case req.Params.Ref.Type == "ref/prompt" && req.Params.Argument.Name == "doc_id":
		pool = listIDs()
	case req.Params.Ref.Type == "ref/prompt" && req.Params.Argument.Name == "target_language":
		pool = languages
	default:
		return &mcp.CompleteResult{}, nil
	}

	var matches []string
	for _, v := range pool {
		if prefix == "" || strings.HasPrefix(strings.ToLower(v), prefix) {
			matches = append(matches, v)
		}
	}
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values: matches,
			Total:  len(matches),
		},
	}, nil
}

// userPrompt wraps a single user-role TextContent in a GetPromptResult.
func userPrompt(text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
	}
}
