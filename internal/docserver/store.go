package docserver

import (
	"encoding/json"
	"sort"
	"sync"
)

// docs is the in-memory document store. Protected by docsMu.
var docs = map[string]string{
	"deposition.md":   "This deposition covers the testimony of Angela Smith, P.E.",
	"report.pdf":      "The report details the state of a 20m condenser tower.",
	"financials.docx": "These financials outline the project's budget and expenditures.",
	"outlook.pdf":     "This document presents the projected future performance of the system.",
	"plan.md":         "The plan outlines the steps for the project's implementation.",
	"spec.txt":        "These specifications define the technical requirements for the equipment.",
}

var docsMu sync.RWMutex

// getDoc returns the body for id and whether it exists.
func getDoc(id string) (string, bool) {
	docsMu.RLock()
	defer docsMu.RUnlock()
	body, ok := docs[id]
	return body, ok
}

// setDoc replaces the body for id.
func setDoc(id, body string) {
	docsMu.Lock()
	defer docsMu.Unlock()
	docs[id] = body
}

// listIDs returns all document IDs in sorted order.
func listIDs() []string {
	docsMu.RLock()
	ids := make([]string, 0, len(docs))
	for id := range docs {
		ids = append(ids, id)
	}
	docsMu.RUnlock()
	sort.Strings(ids)
	return ids
}

// stats returns the document count and total character count.
func stats() (count, total int) {
	docsMu.RLock()
	defer docsMu.RUnlock()
	count = len(docs)
	for _, body := range docs {
		total += len(body)
	}
	return
}

// statsJSON returns the document count, total chars, and average chars per doc
// as a JSON object. avg is 0 when there are no documents.
func statsJSON() ([]byte, error) {
	count, total := stats()
	avg := 0
	if count > 0 {
		avg = total / count
	}
	return json.Marshal(map[string]int{
		"count":       count,
		"total_chars": total,
		"avg_chars":   avg,
	})
}
