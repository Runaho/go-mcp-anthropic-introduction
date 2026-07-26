// Package docserver implements the DocumentMCP server: an in-memory document
// store exposed over MCP stdio.
//
// Tools (with annotations):
//   - read_doc_contents(doc_id)            [read-only]   -> text
//   - edit_document(doc_id, old, new)      [destructive] -> mutates
//   - list_docs_meta()                     [read-only]   -> JSON stats
//
// Resources:
//   - docs://documents                     -> JSON list of doc IDs
//   - docs://documents/{doc_id}            -> text/plain doc body
//   - docs://stats                         -> JSON document count + avg length
//
// Prompts:
//   - format(doc_id)        -> reformat doc as Markdown
//   - summarize(doc_id)     -> short summary
//   - translate(doc_id, target_language)  -> translate doc contents
package docserver

import (
	"context"
	"encoding/json"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ServerName    = "DocumentMCP"
	ServerVersion = "0.2.0"
)

var docs = map[string]string{
	"deposition.md":   "This deposition covers the testimony of Angela Smith, P.E.",
	"report.pdf":      "The report details the state of a 20m condenser tower.",
	"financials.docx": "These financials outline the project's budget and expenditures.",
	"outlook.pdf":     "This document presents the projected future performance of the system.",
	"plan.md":         "The plan outlines the steps for the project's implementation.",
	"spec.txt":        "These specifications define the technical requirements for the equipment.",
}

var docsMu sync.RWMutex

// New creates a DocumentMCP server with all tools, resources, and prompts registered.
func New() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: ServerName, Version: ServerVersion},
		&mcp.ServerOptions{
			Instructions: "DocumentMCP — in-memory document store. Use the tools and " +
				"resources to read, edit, and inspect documents. Use prompts (format, " +
				"summarize, translate) to give the model a structured starting point.",
			CompletionHandler: complete,
		},
	)
	AddTools(s)
	AddResources(s)
	AddPrompts(s)
	return s
}

// ---------- completion ----------

// Languages used as completions for the `translate` prompt's target_language arg.
var languages = []string{"English", "French", "German", "Italian", "Japanese", "Spanish", "Turkish"}

func complete(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if req.Params.Ref == nil {
		return &mcp.CompleteResult{}, nil
	}
	prefix := strings.ToLower(req.Params.Argument.Value)
	var pool []string

	switch {
	case req.Params.Ref.Type == "ref/resource" && strings.HasPrefix(req.Params.Ref.URI, "docs://documents/{doc_id}"):
		pool = docIDs()
	case req.Params.Ref.Type == "ref/prompt" && req.Params.Argument.Name == "doc_id":
		pool = docIDs()
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

func docIDs() []string {
	docsMu.RLock()
	ids := make([]string, 0, len(docs))
	for id := range docs {
		ids = append(ids, id)
	}
	docsMu.RUnlock()
	sort.Strings(ids)
	return ids
}

// ---------- tools ----------

type readArgs struct {
	DocID string `json:"doc_id" jsonschema:"Id of the document to read"`
}

func readDoc(_ context.Context, _ *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, any, error) {
	docsMu.RLock()
	body, ok := docs[in.DocID]
	docsMu.RUnlock()
	if !ok {
		return toolErr(in.DocID), nil, nil
	}
	return text(body), nil, nil
}

type editArgs struct {
	DocID  string `json:"doc_id"  jsonschema:"Id of the document that will be edited"`
	OldStr string `json:"old_str" jsonschema:"The text to replace. Must match exactly, including whitespace"`
	NewStr string `json:"new_str" jsonschema:"The new text to insert in place of the old text"`
}

func editDoc(_ context.Context, _ *mcp.CallToolRequest, in editArgs) (*mcp.CallToolResult, any, error) {
	docsMu.Lock()
	defer docsMu.Unlock()
	body, ok := docs[in.DocID]
	if !ok {
		return toolErr(in.DocID), nil, nil
	}
	docs[in.DocID] = replaceAll(body, in.OldStr, in.NewStr)
	return nil, nil, nil
}

func listDocsMeta(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	docsMu.RLock()
	count := len(docs)
	total := 0
	for _, body := range docs {
		total += len(body)
	}
	docsMu.RUnlock()
	avg := 0
	if count > 0 {
		avg = total / count
	}
	out, _ := json.Marshal(map[string]int{
		"count":          count,
		"total_chars":    total,
		"avg_chars":      avg,
	})
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(out)}},
	}, nil, nil
}

func AddTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "read_doc_contents",
		Description: "Read the contents of a document and return it as a string.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, readDoc)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "edit_document",
		Description: "Edit a document by replacing a string in the documents content with a new string.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr(true),
			IdempotentHint:  false,
			OpenWorldHint:   ptr(false),
		},
	}, editDoc)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_docs_meta",
		Description: "Return aggregate stats (count, total chars, average chars) for all documents.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, listDocsMeta)
}

// ---------- resources ----------

func listDocs(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	body, _ := json.Marshal(docIDs())
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "docs://documents",
			MIMEType: "application/json",
			Text:     string(body),
		}},
	}, nil
}

func fetchDoc(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := path.Base(req.Params.URI)
	docsMu.RLock()
	body, ok := docs[id]
	docsMu.RUnlock()
	if !ok {
		return nil, errMissing(id)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     body,
		}},
	}, nil
}

func docStats(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	docsMu.RLock()
	defer docsMu.RUnlock()
	count := len(docs)
	total := 0
	for _, body := range docs {
		total += len(body)
	}
	avg := 0
	if count > 0 {
		avg = total / count
	}
	body, _ := json.Marshal(map[string]int{
		"count":       count,
		"total_chars": total,
		"avg_chars":   avg,
	})
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "docs://stats",
			MIMEType: "application/json",
			Text:     string(body),
		}},
	}, nil
}

func AddResources(s *mcp.Server) {
	s.AddResource(&mcp.Resource{
		Name:     "documents",
		URI:      "docs://documents",
		MIMEType: "application/json",
	}, listDocs)

	s.AddResource(&mcp.Resource{
		Name:     "stats",
		URI:      "docs://stats",
		MIMEType: "application/json",
	}, docStats)

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "document",
		URITemplate: "docs://documents/{doc_id}",
		MIMEType:    "text/plain",
	}, fetchDoc)
}

// ---------- prompts ----------

func userPrompt(text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
	}
}

func formatDoc(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	return userPrompt(
		"Your goal is to reformat a document to be written with markdown syntax.\n\n" +
			"The id of the document you need to reformat is:\n<document_id>\n" + docID + "\n</document_id>\n\n" +
			"Add in headers, bullet points, tables, etc as necessary. Feel free to add in extra text, but don't change the meaning of the report.\n" +
			"Use the 'edit_document' tool to edit the document. After the document has been edited, respond with the final version of the doc. Don't explain your changes.\n"), nil
}

func summarizeDoc(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	return userPrompt(
		"Please provide a concise summary of the following document.\n\n" +
			"Document id: " + docID + "\n\n" +
			"Read the document with the 'read_doc_contents' tool, then respond with a 3-5 sentence summary highlighting the key points.\n" +
			"Do not edit the document. Keep the response under 150 words.\n"), nil
}

func translateDoc(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	docID := req.Params.Arguments["doc_id"]
	lang := req.Params.Arguments["target_language"]
	return userPrompt(
		"Please translate the following document into " + lang + ".\n\n" +
			"Document id: " + docID + "\n\n" +
			"Read the document with the 'read_doc_contents' tool, then respond with the full translation.\n" +
			"Preserve technical terms, names, and formatting (Markdown, lists) where possible.\n"), nil
}

func AddPrompts(s *mcp.Server) {
	docID := func(desc string) *mcp.PromptArgument {
		return &mcp.PromptArgument{Name: "doc_id", Required: true, Description: desc}
	}
	s.AddPrompt(&mcp.Prompt{
		Name:        "format",
		Description: "Rewrites the contents of the document in Markdown format.",
		Arguments:   []*mcp.PromptArgument{docID("Id of the document to format")},
	}, formatDoc)

	s.AddPrompt(&mcp.Prompt{
		Name:        "summarize",
		Description: "Produces a 3-5 sentence summary of the document.",
		Arguments:   []*mcp.PromptArgument{docID("Id of the document to summarize")},
	}, summarizeDoc)

	s.AddPrompt(&mcp.Prompt{
		Name:        "translate",
		Description: "Translates the document into a target language.",
		Arguments: []*mcp.PromptArgument{
			docID("Id of the document to translate"),
			{Name: "target_language", Required: true, Description: "Target language (e.g. Turkish, French)"},
		},
	}, translateDoc)
}

// ---------- helpers ----------

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

func toolErr(id string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: errMissing(id).Error()}},
		IsError: true,
	}
}

func errMissing(id string) error {
	return &mcpError{msg: "Doc with id " + id + " not found"}
}

func ptr[T any](v T) *T { return &v }

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := make([]byte, 0, len(s))
	i := 0
	for {
		j := indexAt(s, old, i)
		if j < 0 {
			out = append(out, s[i:]...)
			return string(out)
		}
		out = append(out, s[i:j]...)
		out = append(out, new...)
		i = j + len(old)
	}
}

func indexAt(s, sub string, from int) int {
	if from >= len(s) {
		return -1
	}
	if sub == "" {
		return from
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

type mcpError struct{ msg string }

func (e *mcpError) Error() string { return e.msg }
