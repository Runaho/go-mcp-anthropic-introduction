// Package docserver implements the DocumentMCP server: an in-memory document
// store exposed over MCP stdio.
//
// Each tool, resource, and prompt lives in its own file as a self-registering
// feature type. server.go only orchestrates registration — it knows nothing
// about the internals of any individual feature.
//
// Capabilities:
//
//	Tools:
//	  - read_doc_contents(doc_id)            [read-only]   -> text
//	  - edit_document(doc_id, old, new)      [destructive] -> mutates
//	  - list_docs_meta()                     [read-only]   -> JSON stats
//
//	Resources:
//	  - docs://documents                     -> JSON list of doc IDs
//	  - docs://documents/{doc_id}            -> text/plain doc body
//	  - docs://stats                         -> JSON document count + avg length
//
//	Prompts:
//	  - format(doc_id)        -> reformat doc as Markdown
//	  - summarize(doc_id)     -> 3-5 sentence summary
//	  - translate(doc_id, target_language)  -> translate doc contents
package docserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

const (
	ServerName    = "DocumentMCP"
	ServerVersion = "0.3.0"
)

// New builds a DocumentMCP server with every feature registered.
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

	// tools
	NewReadDoc().Register(s)
	NewEditDoc().Register(s)
	NewListDocsMeta().Register(s)

	// resources
	NewDocumentsList().Register(s)
	NewDocument().Register(s)
	NewDocStats().Register(s)

	// prompts
	NewFormat().Register(s)
	NewSummarize().Register(s)
	NewTranslate().Register(s)

	return s
}
