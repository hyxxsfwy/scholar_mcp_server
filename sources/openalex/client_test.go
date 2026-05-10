package openalex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchPapersWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		queryValues := request.URL.Query()
		if queryValues.Get("search") != "quantitative finance" {
			t.Fatalf("expected search query, got %q", queryValues.Get("search"))
		}
		if queryValues.Get("mailto") != "test@example.com" {
			t.Fatalf("expected mailto polite pool parameter")
		}
		if queryValues.Get("per-page") != "20" || queryValues.Get("page") != "1" {
			t.Fatalf("expected paging parameters, got page=%q per-page=%q", queryValues.Get("page"), queryValues.Get("per-page"))
		}
		filter := queryValues.Get("filter")
		for _, expected := range []string{"from_publication_date:2020-01-01", "to_publication_date:2023-12-31", "type:article", "is_oa:true", "cited_by_count:>100"} {
			if !strings.Contains(filter, expected) {
				t.Fatalf("expected filter %q in %q", expected, filter)
			}
		}
		if queryValues.Get("sort") != "cited_by_count:desc" {
			t.Fatalf("expected citation sort, got %q", queryValues.Get("sort"))
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "meta": {"count": 123, "page": 1, "per_page": 20},
  "results": [{
    "id": "https://openalex.org/W123",
    "doi": "https://doi.org/10.1093/rfs/hhaa009",
    "display_name": "Empirical Asset Pricing via Machine Learning",
    "publication_year": 2020,
    "publication_date": "2020-02-01",
    "type": "article",
    "cited_by_count": 2043
  }]
}`))
	}))
	defer server.Close()

	client := NewOpenAlexClientWithBaseURL("test@example.com", server.URL)
	result, err := client.SearchPapersWithParams(context.Background(), &OpenAlexSearchParam{
		Query:          "quantitative finance",
		YearRange:      "2020-2023",
		Type:           "article",
		OpenAccessOnly: true,
		MinCitations:   100,
		Limit:          20,
		SortBy:         "citation_count",
		SortOrder:      "desc",
	})
	if err != nil {
		t.Fatalf("SearchPapersWithParams failed: %v", err)
	}
	if result.Total != 123 || len(result.Papers) != 1 {
		t.Fatalf("expected one result with total 123, got total=%d len=%d", result.Total, len(result.Papers))
	}
	if result.Papers[0].DisplayName != "Empirical Asset Pricing via Machine Learning" {
		t.Fatalf("unexpected paper: %+v", result.Papers[0])
	}
}

func TestSearchPapersWithParamsRejectsEmptyQuery(t *testing.T) {
	client := NewOpenAlexClient("")
	_, err := client.SearchPapers(context.Background(), "", 0, 10)
	if err == nil {
		t.Fatal("expected empty query error")
	}
}

func TestGetPaperNormalizesDOI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.URL.EscapedPath(), "doi:10.1093%2Frfs%2Fhhaa009") {
			t.Fatalf("expected encoded DOI identifier path, got %q", request.URL.EscapedPath())
		}
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{"id":"https://openalex.org/W123","display_name":"Test Paper"}`))
	}))
	defer server.Close()

	client := NewOpenAlexClientWithBaseURL("", server.URL)
	paper, err := client.GetPaper(context.Background(), "https://doi.org/10.1093/rfs/hhaa009")
	if err != nil {
		t.Fatalf("GetPaper failed: %v", err)
	}
	if paper.ID != "https://openalex.org/W123" {
		t.Fatalf("unexpected paper: %+v", paper)
	}
}
