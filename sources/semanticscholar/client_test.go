package semanticscholar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPapers(t *testing.T) {
	jsonResp := `{
		"total": 1,
		"data": [{
			"paperId": "abc123",
			"title": "Test Paper",
			"abstract": "Test abstract",
			"year": 2023,
			"citationCount": 10,
			"authors": [{"name": "John Doe"}]
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonResp))
	}))
	defer server.Close()

	client := NewSemanticScholarClient()
	client.baseURL = server.URL

	result, err := client.SearchPapers(context.Background(), "test", 0, 10)
	if err != nil {
		t.Fatalf("SearchPapers failed: %v", err)
	}
	if len(result.Papers) != 1 {
		t.Errorf("expected 1 paper, got %d", len(result.Papers))
	}
	if result.Papers[0].Title != "Test Paper" {
		t.Errorf("expected title 'Test Paper', got %q", result.Papers[0].Title)
	}
}

func TestSearchPapers_Retry429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"message": "Too Many Requests"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total": 0, "data": []}`))
	}))
	defer server.Close()

	client := NewSemanticScholarClient()
	client.baseURL = server.URL

	_, err := client.SearchPapers(context.Background(), "test", 0, 1)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
