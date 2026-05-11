package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestBrokerResearchSourceSearchPapers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "reports": [{
    "id": "citic-quant-2024-01",
    "title": "量化配置月报",
    "summary": "覆盖多资产配置与风险预算模型。",
    "authors": ["中信证券金工团队"],
	"broker": "中信证券",
    "published_at": "2024-04-02T09:30:00Z",
    "url": "https://research.example.com/citic/monthly",
    "categories": ["金融工程", "资产配置"],
    "read_count": 321
  }]
}`))
	}))
	defer server.Close()

	source := NewBrokerResearchSource(SourceConfig{BaseURL: server.URL, RequestTimeout: 1})
	papers, total, err := source.SearchPapers(context.Background(), common.SearchParams{
		Query:     "风险预算",
		Publisher: "中信证券",
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("SearchPapers failed: %v", err)
	}
	if total != 1 || len(papers) != 1 {
		t.Fatalf("expected one paper, got total=%d len=%d", total, len(papers))
	}

	paper := papers[0]
	if paper.Source != "broker_research" || paper.ID != "citic-quant-2024-01" {
		t.Fatalf("expected broker_research source and id, got %+v", paper)
	}
	if paper.Publisher != "中信证券" || paper.Journal != "中信证券" {
		t.Fatalf("expected broker metadata to map to publisher/journal, got %+v", paper)
	}
	if paper.PublishedDate != "2024-04-02" || paper.Type != "research-report" {
		t.Fatalf("expected normalized date and report type, got %+v", paper)
	}
	if !paper.IsOpenAccess || paper.ReadCount != 321 || paper.URL == "" {
		t.Fatalf("expected open link and read count, got %+v", paper)
	}
}
