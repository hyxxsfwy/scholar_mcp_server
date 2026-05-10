package arxiv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPapers(t *testing.T) {
	xmlResp := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <totalResults>1</totalResults>
  <entry>
    <id>http://arxiv.org/abs/2301.07041v1</id>
    <title>Test Paper</title>
    <summary>Test abstract</summary>
    <author><name>John Doe</name></author>
    <category term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <published>2023-01-17T00:00:00Z</published>
    <updated>2023-01-17T00:00:00Z</updated>
    <link href="http://arxiv.org/abs/2301.07041v1" rel="alternate" type="text/html"/>
    <link href="http://arxiv.org/pdf/2301.07041v1" rel="related" type="application/pdf"/>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewArxivClient()
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

func TestSearchPapers_EmptyQuery(t *testing.T) {
	client := NewArxivClient()
	_, err := client.SearchPapers(context.Background(), "", 0, 10)
	if err == nil {
		t.Error("expected error for empty query")
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
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><totalResults>0</totalResults></feed>`))
	}))
	defer server.Close()

	client := NewArxivClient()
	client.baseURL = server.URL

	_, err := client.SearchPapers(context.Background(), "test", 0, 1)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
