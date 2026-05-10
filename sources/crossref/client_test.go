package crossref

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPapers(t *testing.T) {
	jsonResp := `{
		"status": "ok",
		"message": {
			"total-results": 1,
			"items": [{
				"DOI": "10.1086/260062",
				"title": ["Test Paper"],
				"author": [{"given": "John", "family": "Doe"}],
				"is-referenced-by-count": 100
			}]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonResp))
	}))
	defer server.Close()

	client := NewCrossrefClient()
	client.baseURL = server.URL

	result, err := client.SearchPapers(context.Background(), "test", 0, 10)
	if err != nil {
		t.Fatalf("SearchPapers failed: %v", err)
	}
	if len(result.Papers) != 1 {
		t.Errorf("expected 1 paper, got %d", len(result.Papers))
	}
	if result.Papers[0].DOI != "10.1086/260062" {
		t.Errorf("expected DOI '10.1086/260062', got %q", result.Papers[0].DOI)
	}
}

func TestSearchPapers_Retry429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "message": {"total-results": 0, "items": []}}`))
	}))
	defer server.Close()

	client := NewCrossrefClient()
	client.baseURL = server.URL

	_, err := client.SearchPapers(context.Background(), "test", 0, 1)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
