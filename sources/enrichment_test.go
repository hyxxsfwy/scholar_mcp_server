package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
	openalexapi "github.com/Seelly/scholar_mcp_server/sources/openalex"
)

func TestSearchResultEnricherEnrichesTopNWithLLM(t *testing.T) {
	var requestCount int32
	server := newEnrichmentLLMTestServer(t, &requestCount)
	defer server.Close()

	enricher := NewSearchResultEnricher(ResultEnrichmentConfig{
		Enabled:         true,
		TopN:            1,
		PrefetchPDF:     false,
		MaxPDFBytes:     1024,
		PDFTextMaxChars: 500,
		MaxInputChars:   4000,
		TimeoutSeconds:  5,
		LLM: LLMConfig{
			APIKey:         "test-key",
			BaseURL:        server.URL,
			Model:          "deepseek-v4-flash",
			MaxTokens:      300,
			Temperature:    0.1,
			MaxConcurrency: 2,
		},
	})
	papers := []common.UnifiedPaper{
		{Title: "First Paper", Abstract: "因子投资组合构建。"},
		{Title: "Second Paper", Abstract: "风险模型。"},
	}

	enricher.EnrichPapers(context.Background(), papers)

	assertTopNEnrichment(t, requestCount, papers)
}

func newEnrichmentLLMTestServer(t *testing.T, requestCount *int32) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(requestCount, 1)
		assertLLMRequest(t, request)
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"## 详细大纲\n- 研究问题\n\n## 梗概\n本文讨论因子投资。"}}]}`))
	}))
}

func assertLLMRequest(t *testing.T, request *http.Request) {
	t.Helper()
	if request.URL.Path != "/chat/completions" {
		t.Fatalf("expected chat completions path, got %s", request.URL.Path)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
		t.Fatalf("expected bearer token, got %q", got)
	}

	var payload openAIChatCompletionRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request failed: %v", err)
	}
	if payload.Model != "deepseek-v4-flash" {
		t.Fatalf("expected configured model, got %q", payload.Model)
	}
	prompt := payload.Messages[len(payload.Messages)-1].Content
	if !strings.Contains(prompt, "First Paper") || !strings.Contains(prompt, "因子投资") {
		t.Fatalf("expected prompt to include first paper title and abstract, got %q", prompt)
	}
}

func assertTopNEnrichment(t *testing.T, requestCount int32, papers []common.UnifiedPaper) {
	t.Helper()
	if requestCount != 1 {
		t.Fatalf("expected one LLM request, got %d", requestCount)
	}
	if papers[0].Enrichment == nil || !strings.Contains(papers[0].Enrichment.Content, "因子投资") {
		t.Fatalf("expected first paper enrichment, got %+v", papers[0].Enrichment)
	}
	if papers[0].Enrichment.PDFPrefetched || papers[0].Enrichment.PDFTextChars != 0 {
		t.Fatalf("expected no PDF prefetch, got %+v", papers[0].Enrichment)
	}
	if papers[1].Enrichment != nil {
		t.Fatalf("expected second paper not to be enriched, got %+v", papers[1].Enrichment)
	}
}

func TestSearchResultEnricherDisabledWithoutAPIKey(t *testing.T) {
	enricher := NewSearchResultEnricher(ResultEnrichmentConfig{
		Enabled: true,
		LLM:     LLMConfig{BaseURL: "https://llm.example.com/v1", Model: "deepseek-v4-flash"},
	})
	papers := []common.UnifiedPaper{{Title: "Paper", Abstract: "Abstract"}}

	enricher.EnrichPapers(context.Background(), papers)

	if papers[0].Enrichment != nil {
		t.Fatalf("expected enrichment to stay disabled without API key, got %+v", papers[0].Enrichment)
	}
}

func TestFirstOpenAlexResolvedPDFURLUsesOpenAccessLocation(t *testing.T) {
	paper := openalexapi.Paper{
		PrimaryLocation: openalexapi.Location{IsOA: false, PDFURL: "https://closed.example.com/paper.pdf"},
		BestOALocation:  &openalexapi.Location{IsOA: true, PDFURL: "https://oa.example.com/best.pdf"},
		Locations: []openalexapi.Location{
			{IsOA: true, PDFURL: "https://oa.example.com/other.pdf"},
		},
	}

	got := firstOpenAlexResolvedPDFURL(paper)
	if got != "https://oa.example.com/best.pdf" {
		t.Fatalf("expected best OA PDF URL, got %q", got)
	}
}
