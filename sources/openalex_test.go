package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestOpenAlexSourceSearchPapers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "meta": {"count": 1},
  "results": [{
    "id": "https://openalex.org/W123",
    "doi": "https://doi.org/10.1093/rfs/hhaa009",
    "display_name": "Empirical Asset Pricing via Machine Learning",
    "publication_year": 2020,
    "publication_date": "2020-02-01",
    "type": "article",
    "type_crossref": "journal-article",
    "cited_by_count": 2043,
    "ids": {"openalex": "https://openalex.org/W123", "doi": "https://doi.org/10.1093/rfs/hhaa009"},
    "authorships": [{"author": {"display_name": "Shihao Gu"}, "institutions": [{"display_name": "University of Chicago"}]}],
    "primary_location": {
      "is_oa": true,
      "landing_page_url": "https://academic.oup.com/rfs/article/33/5/2223/5758276",
      "pdf_url": "https://example.com/open.pdf",
      "source": {"display_name": "The Review of Financial Studies", "type": "journal"}
    },
    "open_access": {"is_oa": true, "oa_status": "green", "oa_url": "https://example.com/open.pdf"},
    "biblio": {"volume": "33", "issue": "5", "first_page": "2223", "last_page": "2273"},
    "concepts": [{"display_name": "Finance"}],
    "keywords": [{"display_name": "asset pricing"}],
    "abstract_inverted_index": {"Machine": [0], "learning": [1], "works.": [2]}
  }]
}`))
	}))
	defer server.Close()

	source := NewOpenAlexSource(SourceConfig{APIKey: "test@example.com", BaseURL: server.URL})
	papers, total, err := source.SearchPapers(context.Background(), common.SearchParams{Query: "asset pricing", Limit: 1})
	if err != nil {
		t.Fatalf("SearchPapers failed: %v", err)
	}
	if total != 1 || len(papers) != 1 {
		t.Fatalf("expected one result, got total=%d len=%d", total, len(papers))
	}

	paper := papers[0]
	if paper.Source != "openalex" || paper.DOI != "10.1093/rfs/hhaa009" {
		t.Fatalf("expected OpenAlex source and normalized DOI, got %+v", paper)
	}
	if paper.Journal != "The Review of Financial Studies" || paper.PublishedDate != "2020-02-01" {
		t.Fatalf("expected journal and date, got %+v", paper)
	}
	if paper.Abstract != "Machine learning works." {
		t.Fatalf("expected reconstructed abstract, got %q", paper.Abstract)
	}
	if paper.PDFURL == "" || !paper.IsOpenAccess || paper.CitationCount != 2043 {
		t.Fatalf("expected OA PDF and citations, got %+v", paper)
	}
	if paper.Pages != "2223-2273" || len(paper.Authors) != 1 || len(paper.Categories) != 1 || len(paper.Keywords) != 1 {
		t.Fatalf("expected bibliographic fields, got %+v", paper)
	}
}
