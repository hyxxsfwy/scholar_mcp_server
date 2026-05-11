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
		assertSearchRequest(t, request)

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
	assertSearchResult(t, result)
}

func assertSearchRequest(t *testing.T, request *http.Request) {
	t.Helper()
	assertQueryValue(t, request, "search", "quantitative finance")
	assertQueryValue(t, request, "mailto", "test@example.com")
	assertQueryValue(t, request, "per-page", "20")
	assertQueryValue(t, request, "page", "1")
	assertQueryValue(t, request, "sort", "cited_by_count:desc")
	assertFilterContains(t, request, "from_publication_date:2020-01-01")
	assertFilterContains(t, request, "to_publication_date:2023-12-31")
	assertFilterContains(t, request, "type:article")
	assertFilterContains(t, request, "is_oa:true")
	assertFilterContains(t, request, "cited_by_count:>100")
}

func assertSearchResult(t *testing.T, result *SearchResult) {
	t.Helper()
	if result.Total != 123 || len(result.Papers) != 1 {
		t.Fatalf("expected one result with total 123, got total=%d len=%d", result.Total, len(result.Papers))
	}
	if result.Papers[0].DisplayName != "Empirical Asset Pricing via Machine Learning" {
		t.Fatalf("unexpected paper: %+v", result.Papers[0])
	}
}

func assertQueryValue(t *testing.T, request *http.Request, key, expected string) {
	t.Helper()
	if actual := request.URL.Query().Get(key); actual != expected {
		t.Fatalf("expected query parameter %s=%q, got %q", key, expected, actual)
	}
}

func assertFilterContains(t *testing.T, request *http.Request, expected string) {
	t.Helper()
	filter := request.URL.Query().Get("filter")
	if !strings.Contains(filter, expected) {
		t.Fatalf("expected filter %q in %q", expected, filter)
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
