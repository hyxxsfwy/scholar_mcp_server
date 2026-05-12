package sources

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestMain(m *testing.M) {
	semanticPlanAnalyzer = nil
	semanticReranker = nil
	os.Exit(m.Run())
}

func TestBuildSemanticSearchPlanForCTAQuantQuery(t *testing.T) {
	plan := buildSemanticSearchPlan(context.Background(), common.SearchParams{Query: "CTA量化策略"})
	if plan == nil || !plan.Applied {
		t.Fatalf("expected applied semantic plan, got %+v", plan)
	}
	if plan.Context != "quant_finance_trading" {
		t.Fatalf("expected quant finance context, got %+v", plan)
	}
	if !strings.Contains(plan.SearchQuery, "managed futures") || !strings.Contains(plan.SearchQuery, "time series momentum") {
		t.Fatalf("expected rewritten query to include finance terms, got %q", plan.SearchQuery)
	}
	if !reflect.DeepEqual(plan.ArxivCategories, []string{"q-fin.TR", "q-fin.PM", "q-fin.ST"}) {
		t.Fatalf("expected q-fin arXiv categories, got %#v", plan.ArxivCategories)
	}
}

func TestApplySmartArxivCategoriesForQuantFinance(t *testing.T) {
	params := applySmartArxivCategories(context.Background(), common.SearchParams{Query: "CTA量化策略"})
	if !reflect.DeepEqual(params.Categories, []string{"q-fin.TR", "q-fin.PM", "q-fin.ST"}) {
		t.Fatalf("expected q-fin categories, got %#v", params.Categories)
	}

	explicit := applySmartArxivCategories(context.Background(), common.SearchParams{Query: "CTA量化策略", Categories: []string{"cs.LG"}})
	if !reflect.DeepEqual(explicit.Categories, []string{"cs.LG"}) {
		t.Fatalf("expected explicit categories to be preserved, got %#v", explicit.Categories)
	}
}

func TestSemanticRerankPromotesManagedFutures(t *testing.T) {
	plan := buildSemanticSearchPlan(context.Background(), common.SearchParams{Query: "CTA量化策略"})
	papers := []common.UnifiedPaper{
		{
			Title:         "Paris Agreement climate proposals need a boost to keep warming well below 2 C",
			Abstract:      "Climate warming emissions and sustainability transition policies.",
			CitationCount: 3344,
			Source:        "openalex",
		},
		{
			Title:         "Coronary CTA for computed tomography angiography",
			Abstract:      "Clinical patient artery imaging study.",
			CitationCount: 900,
			Source:        "openalex",
		},
		{
			Title:         "Demystifying Managed Futures",
			Abstract:      "Managed Futures funds and CTAs can be explained by simple trend-following strategies, specifically time series momentum strategies.",
			CitationCount: 51,
			Categories:    []string{"Finance"},
			Source:        "openalex",
		},
		{
			Title:         "From Efficient Markets Theory to Behavioral Finance",
			Abstract:      "A review of financial economics and behavioral finance.",
			CitationCount: 1745,
			Categories:    []string{"Finance"},
			Source:        "openalex",
		},
	}

	rerankTopSemanticWindow(context.Background(), papers, plan, semanticRerankWindow)
	if papers[0].Title != "Demystifying Managed Futures" {
		t.Fatalf("expected managed futures paper first, got %q", papers[0].Title)
	}
}

func TestSemanticRerankUsesLLMOrder(t *testing.T) {
	originalReranker := semanticReranker
	semanticReranker = func(ctx context.Context, plan *SemanticSearchPlan, papers []common.UnifiedPaper) ([]string, error) {
		if len(papers) != 3 {
			t.Fatalf("expected 3 rerank candidates, got %d", len(papers))
		}
		return []string{"p3", "p1", "p2"}, nil
	}
	t.Cleanup(func() { semanticReranker = originalReranker })

	plan := &SemanticSearchPlan{Applied: true, SearchQuery: "risk parity", RequiredTerms: []string{"risk parity"}}
	papers := []common.UnifiedPaper{
		{ID: "first", Title: "First"},
		{ID: "second", Title: "Second"},
		{ID: "third", Title: "Third"},
	}

	rerankTopSemanticWindow(context.Background(), papers, plan, semanticRerankWindow)
	if papers[0].ID != "third" || papers[1].ID != "first" || papers[2].ID != "second" {
		t.Fatalf("expected LLM order to be applied, got %+v", papers)
	}
	if plan.RerankAnalyzer != "llm" || plan.RerankFallback != "" {
		t.Fatalf("expected LLM rerank metadata, got %+v", plan)
	}
}

func TestParseLLMSemanticRerankOrderSanitizesIDs(t *testing.T) {
	orderedIDs, err := parseLLMSemanticRerankOrder(`{"ordered_ids":["p3","p2","p2","bad","p1"]}`, 3)
	if err != nil {
		t.Fatalf("parse LLM rerank order failed: %v", err)
	}
	expected := []string{"p3", "p2", "p1"}
	if !reflect.DeepEqual(orderedIDs, expected) {
		t.Fatalf("expected sanitized ordered IDs %v, got %v", expected, orderedIDs)
	}
}

func TestBuildSemanticSearchPlanUsesLLMAnalyzer(t *testing.T) {
	originalAnalyzer := semanticPlanAnalyzer
	semanticPlanAnalyzer = func(ctx context.Context, params common.SearchParams, input string) (*SemanticSearchPlan, error) {
		return &SemanticSearchPlan{
			Applied:         true,
			OriginalQuery:   params.Query,
			SearchQuery:     "managed futures trend following time series momentum",
			Context:         "quant_finance_trading",
			ContextLabel:    "量化金融/CTA策略",
			Confidence:      0.92,
			Keywords:        []string{"managed futures", "trend following"},
			RequiredTerms:   []string{"managed futures"},
			ArxivCategories: []string{"q-fin.TR"},
			Analyzer:        "llm",
		}, nil
	}
	t.Cleanup(func() { semanticPlanAnalyzer = originalAnalyzer })

	plan := buildSemanticSearchPlan(context.Background(), common.SearchParams{Query: "CTA量化策略"})
	if plan == nil || plan.Analyzer != "llm" {
		t.Fatalf("expected LLM semantic plan, got %+v", plan)
	}
	if plan.SearchQuery != "managed futures trend following time series momentum" {
		t.Fatalf("expected LLM search query, got %q", plan.SearchQuery)
	}
	if !reflect.DeepEqual(plan.ArxivCategories, []string{"q-fin.TR"}) {
		t.Fatalf("expected LLM arXiv categories, got %#v", plan.ArxivCategories)
	}
}

func TestParseLLMSemanticSearchPlanSanitizesOutput(t *testing.T) {
	plan, err := parseLLMSemanticSearchPlan("```json\n{\"applied\":true,\"context\":\"Quant Finance!\",\"context_label\":\" 量化金融 \",\"confidence\":1.5,\"search_query\":\" managed futures   trend following \",\"keywords\":[\"managed futures\",\"\"],\"required_terms\":[\"trend following\"],\"arxiv_categories\":[\"q-fin.TR\",\"bad category;drop\"],\"notes\":[\"ok\"]}\n```", common.SearchParams{Query: "CTA量化策略"})
	if err != nil {
		t.Fatalf("parse LLM semantic plan failed: %v", err)
	}
	if plan.Analyzer != "llm" || plan.Context != "quantfinance" || plan.Confidence != 1 {
		t.Fatalf("expected sanitized LLM plan, got %+v", plan)
	}
	if plan.SearchQuery != "managed futures trend following" {
		t.Fatalf("expected normalized search query, got %q", plan.SearchQuery)
	}
	if !reflect.DeepEqual(plan.ArxivCategories, []string{"q-fin.TR"}) {
		t.Fatalf("expected sanitized arXiv categories, got %#v", plan.ArxivCategories)
	}
}

func TestSearchScholarPapersUsesSemanticPlanAndCandidateLimit(t *testing.T) {
	handler, source := newFakeHandler()
	handler.enricher = NewSearchResultEnricher(ResultEnrichmentConfig{Enabled: false})
	source.papers = []common.UnifiedPaper{
		{
			ID:            "climate",
			Title:         "Paris Agreement climate proposals need a boost",
			Abstract:      "Climate warming emissions and sustainability transition policies.",
			CitationCount: 3344,
			Source:        "fake",
		},
		{
			ID:            "managed-futures",
			Title:         "Demystifying Managed Futures",
			Abstract:      "Managed Futures funds and CTAs can be explained by trend-following and time series momentum strategies.",
			CitationCount: 51,
			Categories:    []string{"Finance"},
			Source:        "fake",
		},
	}

	params := ScholarSearchParam{Query: "CTA量化策略", Limit: 1}
	result, _, err := handler.SearchScholarPapers(context.Background(), nil, &params)
	if err != nil {
		t.Fatalf("SearchScholarPapers returned error: %v", err)
	}
	if !strings.Contains(source.lastParams.Query, "managed futures") {
		t.Fatalf("expected semantic query expansion, got %q", source.lastParams.Query)
	}
	if source.lastParams.Offset != 0 || source.lastParams.Limit != semanticRerankWindow {
		t.Fatalf("expected first %d candidates, got offset=%d limit=%d", semanticRerankWindow, source.lastParams.Offset, source.lastParams.Limit)
	}

	structuredData, ok := result.Meta["structured_data"].(string)
	if !ok || structuredData == "" {
		t.Fatalf("expected structured data, got %#v", result.Meta["structured_data"])
	}
	var structured AggregatedSearchResult
	if err := json.Unmarshal([]byte(structuredData), &structured); err != nil {
		t.Fatalf("decode structured data failed: %v", err)
	}
	if structured.SearchPlan == nil || structured.SearchPlan.Context != "quant_finance_trading" {
		t.Fatalf("expected quant finance search plan, got %+v", structured.SearchPlan)
	}
	if len(structured.Papers) != 1 || structured.Papers[0].ID != "managed-futures" {
		t.Fatalf("expected reranked top paper to be managed futures, got %+v", structured.Papers)
	}
}
