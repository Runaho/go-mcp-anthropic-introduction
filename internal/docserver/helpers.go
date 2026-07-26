package docserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

// textResult builds a successful CallToolResult wrapping s as a TextContent.
func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// toolErr builds a failed CallToolResult for a missing document.
func toolErr(id string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: errMissing(id).Error()}},
		IsError: true,
	}
}

// toolErrMsg builds a failed CallToolResult with an arbitrary message.
func toolErrMsg(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

// errMissing returns a "not found" error for a document id.
func errMissing(id string) error {
	return &mcpError{msg: "Doc with id " + id + " not found"}
}

// mcpError satisfies the error interface for tool-level failures.
type mcpError struct{ msg string }

func (e *mcpError) Error() string { return e.msg }
