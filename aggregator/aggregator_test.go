package aggregator

import (
	"context"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestSearchPapers_Validation(t *testing.T) {
	agg := NewScholarAggregator()
	ctx := context.Background()

	_, err := agg.SearchPapers(ctx, common.SearchParams{Query: ""})
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestMergePapers_Deduplication(t *testing.T) {
	agg := NewScholarAggregator()

	papers := []common.UnifiedPaper{
		{DOI: "10.1086/260062", Title: "Paper A"},
		{DOI: "10.1086/260062", Title: "Paper A Duplicate"},
		{DOI: "10.1234/test", Title: "Paper B"},
		{Title: "Paper C"},
		{Title: "Paper C"},
	}

	merged := agg.mergePapers(papers)
	if len(merged) != 3 {
		t.Errorf("expected 3 unique papers, got %d", len(merged))
	}
}

func TestSortPapers(t *testing.T) {
	agg := NewScholarAggregator()

	papers := []common.UnifiedPaper{
		{Title: "A", CitationCount: 10},
		{Title: "B", CitationCount: 50},
		{Title: "C", CitationCount: 5},
	}

	agg.sortPapers(papers, "citation_count", "desc")
	if papers[0].CitationCount != 50 {
		t.Errorf("expected highest citation first, got %d", papers[0].CitationCount)
	}

	agg.sortPapers(papers, "citation_count", "asc")
	if papers[0].CitationCount != 5 {
		t.Errorf("expected lowest citation first, got %d", papers[0].CitationCount)
	}
}
