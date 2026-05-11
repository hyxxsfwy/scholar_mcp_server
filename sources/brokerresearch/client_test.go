package brokerresearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBrokerResearchClientSearchJSONFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		if got := request.URL.Query().Get("q"); got != "多因子" {
			t.Fatalf("expected query placeholder to be expanded, got %q", got)
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{
  "items": [
    {
      "id": "htsc-quant-1",
      "title": "多因子选股周报",
      "abstract": "跟踪多因子组合表现与行业暴露。",
      "authors": "华泰金工团队",
      "broker": "华泰证券",
      "team": "金融工程",
      "published_date": "2024-05-10",
      "url": "https://research.example.com/htsc/1",
      "pdf_url": "https://research.example.com/htsc/1.pdf",
      "categories": ["金融工程", "多因子"],
      "keywords": "选股,因子",
      "read_count": "1234",
      "score": 95
    },
    {
      "id": "macro-1",
      "title": "宏观利率点评",
      "broker": "华泰证券",
      "published_date": "2023-01-01"
    }
  ]
}`))
	}))
	defer server.Close()

	feedConfig := fmt.Sprintf(`[{"name":"华泰证券金工","broker":"华泰证券","team":"金融工程","format":"json","url":"%s?q={query}&limit={limit}"}]`, server.URL)
	client := NewBrokerResearchClient(feedConfig, "test-token", time.Second)

	result, err := client.SearchReports(context.Background(), &SearchParam{
		Query:      "多因子",
		Publisher:  "华泰证券",
		Year:       "2024",
		Categories: []string{"金融工程"},
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("SearchReports failed: %v", err)
	}
	if result.Total != 1 || len(result.Reports) != 1 {
		t.Fatalf("expected one filtered report, got total=%d len=%d", result.Total, len(result.Reports))
	}

	report := result.Reports[0]
	if report.ID != "htsc-quant-1" || report.Broker != "华泰证券" || report.Team != "金融工程" {
		t.Fatalf("expected broker defaults to be preserved, got %+v", report)
	}
	if report.PublishedDate != "2024-05-10" || report.URL == "" || report.PDFURL == "" {
		t.Fatalf("expected normalized date and links, got %+v", report)
	}
	if len(report.Authors) != 1 || report.Authors[0] != "华泰金工团队" {
		t.Fatalf("expected string authors to be normalized, got %#v", report.Authors)
	}
	if report.ReadCount != 1234 || report.Score != 95 {
		t.Fatalf("expected flexible numeric fields, got read_count=%d score=%d", report.ReadCount, report.Score)
	}
}
