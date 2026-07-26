package docserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runClient(t *testing.T, fn func(ctx context.Context, cs *mcp.ClientSession, init *mcp.InitializeResult)) {
	t.Helper()
	st, ct := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() { _, _ = New().Connect(ctx, st, nil) }()
	c := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()
	fn(ctx, cs, nil) // init result is captured separately via Session.InitializeResult if needed
}

func toolByName(tools []*mcp.Tool, name string) *mcp.Tool {
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	return nil
}

// ---------- tools ----------

func TestToolsList(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		want := []string{"read_doc_contents", "edit_document", "list_docs_meta"}
		got := map[string]bool{}
		for _, tool := range res.Tools {
			got[tool.Name] = true
		}
		for _, name := range want {
			if !got[name] {
				t.Errorf("missing tool %q", name)
			}
		}
	})
}

func TestToolAnnotations(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}

		read := toolByName(res.Tools, "read_doc_contents")
		if read == nil || read.Annotations == nil || !read.Annotations.ReadOnlyHint {
			t.Error("read_doc_contents should have ReadOnlyHint=true")
		}

		edit := toolByName(res.Tools, "edit_document")
		if edit == nil || edit.Annotations == nil {
			t.Fatal("edit_document missing annotations")
		}
		if edit.Annotations.DestructiveHint == nil || !*edit.Annotations.DestructiveHint {
			t.Error("edit_document should have DestructiveHint=true")
		}
		if edit.Annotations.IdempotentHint {
			t.Error("edit_document should not be IdempotentHint")
		}
	})
}

func TestReadDoc(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "read_doc_contents",
			Arguments: map[string]any{"doc_id": "deposition.md"},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		tc, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", res.Content[0])
		}
		if !strings.Contains(tc.Text, "Angela Smith") {
			t.Errorf("unexpected text: %q", tc.Text)
		}
	})
}

func TestReadDocMissing(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "read_doc_contents",
			Arguments: map[string]any{"doc_id": "nope.md"},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if !res.IsError {
			t.Fatal("expected IsError=true for missing doc")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(tc.Text, "not found") {
			t.Errorf("unexpected text: %q", tc.Text)
		}
	})
}

func TestEditDoc(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		editRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "edit_document",
			Arguments: map[string]any{
				"doc_id":  "deposition.md",
				"old_str": "Angela Smith, P.E.",
				"new_str": "John Doe, Ph.D.",
			},
		})
		if err != nil {
			t.Fatalf("edit: %v", err)
		}
		if editRes.IsError {
			t.Fatalf("unexpected IsError: %v", editRes.Content)
		}
		if tc, ok := editRes.Content[0].(*mcp.TextContent); !ok || tc.Text != "edited deposition.md (replaced 1 occurrence)" {
			t.Errorf("unexpected confirmation text: %q", tc.Text)
		}

		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "read_doc_contents",
			Arguments: map[string]any{"doc_id": "deposition.md"},
		})
		if err != nil {
			t.Fatalf("read after edit: %v", err)
		}
		tc := res.Content[0].(*mcp.TextContent)
		if strings.Contains(tc.Text, "Angela Smith") {
			t.Error("edit did not replace old string")
		}
		if !strings.Contains(tc.Text, "John Doe") {
			t.Error("edit did not insert new string")
		}
	})
}

func TestEditDocOldStrNotFound(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		const untouched = "20m condenser tower"
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "edit_document",
			Arguments: map[string]any{
				"doc_id":  "report.pdf",
				"old_str": "string that does not exist anywhere in the document",
				"new_str": "replacement",
			},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if !res.IsError {
			t.Fatal("expected IsError=true when old_str not found")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(tc.Text, "not found") {
			t.Errorf("expected 'not found' in error text, got %q", tc.Text)
		}

		// verify the document body is unchanged
		readRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "read_doc_contents",
			Arguments: map[string]any{"doc_id": "report.pdf"},
		})
		if err != nil {
			t.Fatalf("read after failed edit: %v", err)
		}
		if !strings.Contains(readRes.Content[0].(*mcp.TextContent).Text, untouched) {
			t.Errorf("document was mutated despite failed edit")
		}
	})
}

func TestListDocsMeta(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_docs_meta", Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		var meta map[string]int
		if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &meta); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if meta["count"] != 6 {
			t.Errorf("expected count=6, got %d", meta["count"])
		}
		if meta["total_chars"] == 0 || meta["avg_chars"] == 0 {
			t.Errorf("expected non-zero stats, got %+v", meta)
		}
	})
}

// ---------- resources ----------

func TestListResources(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ListResources(ctx, nil)
		if err != nil {
			t.Fatalf("ListResources: %v", err)
		}
		got := map[string]bool{}
		for _, r := range res.Resources {
			got[r.URI] = true
		}
		for _, want := range []string{"docs://documents", "docs://stats"} {
			if !got[want] {
				t.Errorf("missing resource %q", want)
			}
		}
	})
}

func TestListResourcesContent(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "docs://documents"})
		if err != nil {
			t.Fatalf("ReadResource: %v", err)
		}
		var ids []string
		if err := json.Unmarshal([]byte(res.Contents[0].Text), &ids); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(ids) != 6 {
			t.Errorf("expected 6 docs, got %d", len(ids))
		}
	})
}

func TestStatsResource(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "docs://stats"})
		if err != nil {
			t.Fatalf("ReadResource: %v", err)
		}
		var stats map[string]int
		if err := json.Unmarshal([]byte(res.Contents[0].Text), &stats); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if stats["count"] != 6 {
			t.Errorf("expected count=6, got %d", stats["count"])
		}
	})
}

func TestReadResourceTemplate(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "docs://documents/plan.md"})
		if err != nil {
			t.Fatalf("ReadResource: %v", err)
		}
		if !strings.Contains(res.Contents[0].Text, "implementation") {
			t.Errorf("unexpected body: %q", res.Contents[0].Text)
		}
	})
}

// ---------- prompts ----------

func TestListPrompts(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ListPrompts(ctx, nil)
		if err != nil {
			t.Fatalf("ListPrompts: %v", err)
		}
		got := map[string]bool{}
		for _, p := range res.Prompts {
			got[p.Name] = true
		}
		for _, want := range []string{"format", "summarize", "translate"} {
			if !got[want] {
				t.Errorf("missing prompt %q", want)
			}
		}
	})
}

func TestPromptArguments(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.ListPrompts(ctx, nil)
		if err != nil {
			t.Fatalf("ListPrompts: %v", err)
		}
		var translate *mcp.Prompt
		for _, p := range res.Prompts {
			if p.Name == "translate" {
				translate = p
				break
			}
		}
		if translate == nil {
			t.Fatal("translate prompt not found")
		}
		if len(translate.Arguments) != 2 {
			t.Fatalf("expected 2 args, got %d", len(translate.Arguments))
		}
		for _, arg := range translate.Arguments {
			if !arg.Required {
				t.Errorf("argument %q should be required", arg.Name)
			}
		}
	})
}

func TestGetFormatPrompt(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "format",
			Arguments: map[string]string{"doc_id": "spec.txt"},
		})
		if err != nil {
			t.Fatalf("GetPrompt: %v", err)
		}
		tc, ok := res.Messages[0].Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", res.Messages[0].Content)
		}
		if !strings.Contains(tc.Text, "spec.txt") {
			t.Errorf("prompt missing doc_id: %q", tc.Text)
		}
	})
}

func TestGetSummarizePrompt(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "summarize",
			Arguments: map[string]string{"doc_id": "report.pdf"},
		})
		if err != nil {
			t.Fatalf("GetPrompt: %v", err)
		}
		tc := res.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "summary") || !strings.Contains(tc.Text, "report.pdf") {
			t.Errorf("unexpected prompt: %q", tc.Text)
		}
	})
}

func TestGetTranslatePrompt(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "translate",
			Arguments: map[string]string{"doc_id": "plan.md", "target_language": "Turkish"},
		})
		if err != nil {
			t.Fatalf("GetPrompt: %v", err)
		}
		tc := res.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "Turkish") || !strings.Contains(tc.Text, "plan.md") {
			t.Errorf("unexpected prompt: %q", tc.Text)
		}
	})
}

// ---------- completion ----------

func TestCompleteResourceDocID(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.Complete(ctx, &mcp.CompleteParams{
			Ref: &mcp.CompleteReference{
				Type: "ref/resource",
				URI:  "docs://documents/{doc_id}",
			},
			Argument: mcp.CompleteParamsArgument{Name: "doc_id", Value: "dep"},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if len(res.Completion.Values) == 0 {
			t.Fatal("expected matches for prefix 'dep'")
		}
		found := false
		for _, v := range res.Completion.Values {
			if v == "deposition.md" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'deposition.md' in matches, got %v", res.Completion.Values)
		}
	})
}

func TestCompletePromptDocID(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.Complete(ctx, &mcp.CompleteParams{
			Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "translate"},
			Argument: mcp.CompleteParamsArgument{Name: "doc_id", Value: "fin"},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		found := false
		for _, v := range res.Completion.Values {
			if v == "financials.docx" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'financials.docx', got %v", res.Completion.Values)
		}
	})
}

func TestCompleteTargetLanguage(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.Complete(ctx, &mcp.CompleteParams{
			Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "translate"},
			Argument: mcp.CompleteParamsArgument{Name: "target_language", Value: "tur"},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		found := false
		for _, v := range res.Completion.Values {
			if v == "Turkish" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'Turkish' in matches, got %v", res.Completion.Values)
		}
	})
}

func TestCompleteEmptyPrefix(t *testing.T) {
	runClient(t, func(ctx context.Context, cs *mcp.ClientSession, _ *mcp.InitializeResult) {
		res, err := cs.Complete(ctx, &mcp.CompleteParams{
			Ref:      &mcp.CompleteReference{Type: "ref/resource", URI: "docs://documents/{doc_id}"},
			Argument: mcp.CompleteParamsArgument{Name: "doc_id", Value: ""},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if len(res.Completion.Values) != 6 {
			t.Errorf("expected all 6 docs on empty prefix, got %d", len(res.Completion.Values))
		}
	})
}
