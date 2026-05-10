package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestGoogleScholarSourceSearchPapers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "search_information": {"total_results": 1},
  "organic_results": [{
    "position": 1,
    "result_id": "rfs-hhaa009",
    "title": "Empirical Asset Pricing via Machine Learning",
    "link": "https://doi.org/10.1093/rfs/hhaa009",
    "snippet": "DOI: 10.1093/rfs/hhaa009",
    "publication_info": {
      "summary": "S Gu, B Kelly, D Xiu - The Review of Financial Studies, 2020 - academic.oup.com"
    },
    "inline_links": {
      "cited_by": {"total": 2043, "link": "https://scholar.google.com/scholar?cites=abc", "cites_id": "abc"},
      "versions": {"total": 5, "link": "https://scholar.google.com/scholar?cluster=abc"}
    },
    "resources": [{"title": "[PDF] example.com", "file_format": "PDF", "link": "https://example.com/paper.pdf"}]
  }]
}`))
	}))
	defer server.Close()

	source := NewGoogleScholarSource(SourceConfig{APIKey: "test-key", BaseURL: server.URL})
	papers, total, err := source.SearchPapers(context.Background(), common.SearchParams{
		Query:   "asset pricing machine learning",
		Author:  "Gu",
		Journal: "Review of Financial Studies",
		Year:    "2020",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("SearchPapers failed: %v", err)
	}
	if total != 1 || len(papers) != 1 {
		t.Fatalf("expected one result, got total=%d len=%d", total, len(papers))
	}

	paper := papers[0]
	if paper.Source != "google_scholar" {
		t.Fatalf("expected google_scholar source, got %q", paper.Source)
	}
	if paper.DOI != "10.1093/rfs/hhaa009" {
		t.Fatalf("expected DOI extraction, got %q", paper.DOI)
	}
	if paper.PublishedDate != "2020" {
		t.Fatalf("expected publication year, got %q", paper.PublishedDate)
	}
	if paper.Journal != "The Review of Financial Studies" {
		t.Fatalf("expected journal extraction, got %q", paper.Journal)
	}
	if paper.CitationCount != 2043 || paper.PDFURL == "" || !paper.IsOpenAccess {
		t.Fatalf("expected citations and PDF fields, got %+v", paper)
	}
	if len(paper.Authors) != 3 {
		t.Fatalf("expected authors parsed from summary, got %#v", paper.Authors)
	}
}
