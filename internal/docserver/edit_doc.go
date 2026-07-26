package docserver

import (
	"context"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EditDoc replaces a string in a document's contents. Destructive but not idempotent.
type EditDoc struct {
	spec *mcp.Tool
}

// NewEditDoc builds a configured EditDoc feature.
func NewEditDoc() *EditDoc {
	f := false
	t := true
	return &EditDoc{
		spec: &mcp.Tool{
			Name:        "edit_document",
			Description: "Edit a document by replacing a string in the documents content with a new string.",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &t,
				IdempotentHint:  false,
				OpenWorldHint:   &f,
			},
		},
	}
}

// Register attaches the tool to s.
func (e *EditDoc) Register(s *mcp.Server) {
	mcp.AddTool(s, e.spec, e.handle)
}

type editArgs struct {
	DocID  string `json:"doc_id"  jsonschema:"Id of the document that will be edited"`
	OldStr string `json:"old_str" jsonschema:"The text to replace. Must match exactly, including whitespace"`
	NewStr string `json:"new_str" jsonschema:"The new text to insert in place of the old text"`
}

func (e *EditDoc) handle(_ context.Context, _ *mcp.CallToolRequest, in editArgs) (*mcp.CallToolResult, any, error) {
	body, ok := getDoc(in.DocID)
	if !ok {
		return toolErr(in.DocID), nil, nil
	}
	if !strings.Contains(body, in.OldStr) {
		return toolErrMsg("old_str not found in document " + in.DocID + "; no changes made"), nil, nil
	}
	n := strings.Count(body, in.OldStr)
	setDoc(in.DocID, strings.ReplaceAll(body, in.OldStr, in.NewStr))
	word := "occurrence"
	if n != 1 {
		word = "occurrences"
	}
	return textResult("edited " + in.DocID + " (replaced " + strconv.Itoa(n) + " " + word + ")"), nil, nil
}
