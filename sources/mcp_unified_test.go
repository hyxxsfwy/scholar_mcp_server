package sources

import (
	"context"
	"fmt"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

type fakePaperSource struct {
	name       string
	lastParams common.SearchParams
	searchErr  error
	papers     []common.UnifiedPaper
}

func (f *fakePaperSource) GetSourceName() string {
	return f.name
}

func (f *fakePaperSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	f.lastParams = params
	if f.searchErr != nil {
		return nil, 0, f.searchErr
	}
	if len(f.papers) == 0 {
		f.papers = []common.UnifiedPaper{{
			ID:     "paper-1",
			Title:  "Test Paper",
			Source: f.name,
		}}
	}

	return f.papers, len(f.papers), nil
}

func (f *fakePaperSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	return &common.UnifiedPaper{ID: identifier, Title: "Test Paper", Source: f.name}, nil
}

func (f *fakePaperSource) IsAvailable(ctx context.Context) bool {
	return true
}

func (f *fakePaperSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{SupportsFullText: true, SupportsAuthorSearch: true}
}

func TestUnifiedMCPHandlerSearchScholarPapersStructuredOnly(t *testing.T) {
	tests := []struct {
		name         string
		params       ScholarSearchParam
		wantQuery    string
		assertParams func(t *testing.T, got common.SearchParams)
	}{
		{
			name:      "title only",
			params:    ScholarSearchParam{Title: "Portfolio Selection"},
			wantQuery: "Portfolio Selection",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Title != "Portfolio Selection" {
					t.Fatalf("expected title filter to be preserved, got %+v", got)
				}
			},
		},
		{
			name:      "author only",
			params:    ScholarSearchParam{Author: "Harry Markowitz"},
			wantQuery: "Harry Markowitz",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Author != "Harry Markowitz" {
					t.Fatalf("expected author filter to be preserved, got %+v", got)
				}
			},
		},
		{
			name:      "journal only",
			params:    ScholarSearchParam{Journal: "Journal of Finance"},
			wantQuery: "Journal of Finance",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Journal != "Journal of Finance" {
					t.Fatalf("expected journal filter to be preserved, got %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSourceManager()
			source := &fakePaperSource{name: "fake"}
			manager.RegisterSource("fake", source, SourceConfig{Enabled: true})

			handler := NewUnifiedMCPHandlerWithManager(manager)
			params := tt.params

			result, _, err := handler.SearchScholarPapers(context.Background(), nil, &params)
			if err != nil {
				t.Fatalf("SearchScholarPapers returned error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if source.lastParams.Query != tt.wantQuery {
				t.Fatalf("expected derived query %q, got %q", tt.wantQuery, source.lastParams.Query)
			}
			if params.Query != tt.wantQuery {
				t.Fatalf("expected params.Query to be normalized to %q, got %q", tt.wantQuery, params.Query)
			}
			tt.assertParams(t, source.lastParams)
		})
	}
}

func TestUnifiedMCPHandlerSearchSourcePapersStructuredOnly(t *testing.T) {
	tests := []struct {
		name         string
		params       SourceSearchParam
		wantQuery    string
		assertParams func(t *testing.T, got common.SearchParams)
	}{
		{
			name:      "title only",
			params:    SourceSearchParam{Source: "fake", ScholarSearchParam: ScholarSearchParam{Title: "DeepLOB"}},
			wantQuery: "DeepLOB",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Title != "DeepLOB" {
					t.Fatalf("expected title filter to be preserved, got %+v", got)
				}
			},
		},
		{
			name:      "author only",
			params:    SourceSearchParam{Source: "fake", ScholarSearchParam: ScholarSearchParam{Author: "Marco Avellaneda"}},
			wantQuery: "Marco Avellaneda",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Author != "Marco Avellaneda" {
					t.Fatalf("expected author filter to be preserved, got %+v", got)
				}
			},
		},
		{
			name:      "journal only",
			params:    SourceSearchParam{Source: "fake", ScholarSearchParam: ScholarSearchParam{Journal: "Quantitative Finance"}},
			wantQuery: "Quantitative Finance",
			assertParams: func(t *testing.T, got common.SearchParams) {
				t.Helper()
				if got.Journal != "Quantitative Finance" {
					t.Fatalf("expected journal filter to be preserved, got %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSourceManager()
			source := &fakePaperSource{name: "fake"}
			manager.RegisterSource("fake", source, SourceConfig{Enabled: true})

			handler := NewUnifiedMCPHandlerWithManager(manager)
			params := tt.params

			result, _, err := handler.SearchSourcePapers(context.Background(), nil, &params)
			if err != nil {
				t.Fatalf("SearchSourcePapers returned error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if source.lastParams.Query != tt.wantQuery {
				t.Fatalf("expected derived query %q, got %q", tt.wantQuery, source.lastParams.Query)
			}
			if params.Query != tt.wantQuery {
				t.Fatalf("expected params.Query to be normalized to %q, got %q", tt.wantQuery, params.Query)
			}
			tt.assertParams(t, source.lastParams)
		})
	}
}

func TestUnifiedMCPHandlerSearchScholarPapersRejectsEmptyStructuredSearch(t *testing.T) {
	manager := NewSourceManager()
	manager.RegisterSource("fake", &fakePaperSource{name: "fake", searchErr: fmt.Errorf("should not be called")}, SourceConfig{Enabled: true})

	handler := NewUnifiedMCPHandlerWithManager(manager)
	_, _, err := handler.SearchScholarPapers(context.Background(), nil, &ScholarSearchParam{})
	if err == nil {
		t.Fatal("expected empty structured search to fail")
	}
}
