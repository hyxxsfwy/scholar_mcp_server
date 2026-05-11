package googlescholar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPapersWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		assertSearchRequest(t, request)

		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "search_information": {"total_results": 123},
  "organic_results": [{
    "position": 1,
    "result_id": "abc123",
    "title": "Empirical Asset Pricing via Machine Learning",
    "link": "https://doi.org/10.1093/rfs/hhaa009",
    "snippet": "A machine learning study in empirical asset pricing.",
    "publication_info": {
      "summary": "S Gu, B Kelly, D Xiu - The Review of Financial Studies, 2020 - academic.oup.com",
      "authors": [{"name": "Shihao Gu"}, {"name": "Bryan Kelly"}]
    },
    "inline_links": {
      "cited_by": {"total": 2043, "link": "https://scholar.google.com/scholar?cites=abc", "cites_id": "abc"},
      "versions": {"total": 5, "link": "https://scholar.google.com/scholar?cluster=abc"}
    },
    "resources": [{"title": "PDF", "file_format": "PDF", "link": "https://example.com/paper.pdf"}]
  }]
}`))
	}))
	defer server.Close()

	client := NewGoogleScholarClientWithBaseURL("test-key", server.URL)
	result, err := client.SearchPapersWithParams(context.Background(), &GoogleScholarSearchParam{
		Query:     "quantitative finance",
		Author:    "Shihao Gu",
		Journal:   "Review of Financial Studies",
		YearRange: "2020-2023",
		Limit:     50,
		SortBy:    "published_date",
	})
	if err != nil {
		t.Fatalf("SearchPapersWithParams failed: %v", err)
	}
	assertSearchResult(t, result)
}

func assertSearchRequest(t *testing.T, request *http.Request) {
	t.Helper()
	assertQueryValue(t, request, "engine", "google_scholar")
	assertQueryValue(t, request, "api_key", "test-key")
	assertQueryValue(t, request, "q", "quantitative finance")
	assertQueryValue(t, request, "as_sauthors", "Shihao Gu")
	assertQueryValue(t, request, "as_publication", "Review of Financial Studies")
	assertQueryValue(t, request, "as_ylo", "2020")
	assertQueryValue(t, request, "as_yhi", "2023")
	assertQueryValue(t, request, "scisbd", "1")
}

func assertSearchResult(t *testing.T, result *SearchResult) {
	t.Helper()
	if result.Total != 123 {
		t.Fatalf("expected total 123, got %d", result.Total)
	}
	if result.Limit != MaxLimit {
		t.Fatalf("expected limit to be capped at %d, got %d", MaxLimit, result.Limit)
	}
	if len(result.Papers) != 1 || result.Papers[0].Title != "Empirical Asset Pricing via Machine Learning" {
		t.Fatalf("unexpected papers: %+v", result.Papers)
	}
}

func assertQueryValue(t *testing.T, request *http.Request, key, expected string) {
	t.Helper()
	if actual := request.URL.Query().Get(key); actual != expected {
		t.Fatalf("expected query parameter %s=%q, got %q", key, expected, actual)
	}
}

func TestSearchPapersWithParamsRequiresAPIKey(t *testing.T) {
	client := NewGoogleScholarClient("")
	_, err := client.SearchPapers(context.Background(), "quantitative finance", 0, 1)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestSearchPapersWithParamsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	client := NewGoogleScholarClientWithBaseURL("bad-key", server.URL)
	_, err := client.SearchPapers(context.Background(), "quantitative finance", 0, 1)
	if err == nil {
		t.Fatal("expected API error")
	}
}
