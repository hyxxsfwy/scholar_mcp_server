package common

import (
	"testing"
)

func TestValidateSearchParams(t *testing.T) {
	tests := []struct {
		name    string
		params  SearchParams
		wantErr bool
		errField string
	}{
		{"valid", SearchParams{Query: "machine learning"}, false, ""},
		{"empty query", SearchParams{Query: ""}, true, "query"},
		{"negative offset", SearchParams{Query: "test", Offset: -1}, true, "offset"},
		{"negative limit", SearchParams{Query: "test", Limit: -1}, true, "limit"},
		{"limit too large", SearchParams{Query: "test", Limit: 1001}, true, "limit"},
		{"invalid sort field", SearchParams{Query: "test", SortBy: "citations"}, false, ""},
		{"invalid sort field bad", SearchParams{Query: "test", SortBy: "unknown_field"}, true, "sort_by"},
		{"invalid sort order", SearchParams{Query: "test", SortOrder: "random"}, true, "sort_order"},
		{"valid year", SearchParams{Query: "test", Year: "2020"}, false, ""},
		{"valid year range", SearchParams{Query: "test", Year: "2018-2023"}, false, ""},
		{"invalid year", SearchParams{Query: "test", Year: "20xx"}, true, "year"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSearchParams(tt.params)
			if tt.wantErr && result.Valid {
				t.Errorf("expected error but got valid")
			}
			if !tt.wantErr && !result.Valid {
				t.Errorf("expected valid but got errors: %v", result.Errors)
			}
			if tt.errField != "" {
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error on field %q, got %v", tt.errField, result.Errors)
				}
			}
		})
	}
}

func TestNormalizeSearchParams(t *testing.T) {
	t.Run("citations alias normalized", func(t *testing.T) {
		p := SearchParams{Query: "test", SortBy: "citations"}
		NormalizeSearchParams(&p)
		if p.SortBy != "citation_count" {
			t.Errorf("expected citation_count, got %s", p.SortBy)
		}
	})

	t.Run("defaults applied", func(t *testing.T) {
		p := SearchParams{Query: "  test  "}
		NormalizeSearchParams(&p)
		if p.Query != "test" {
			t.Errorf("expected trimmed query, got %q", p.Query)
		}
		if p.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", p.Limit)
		}
		if p.SortBy != "relevance" {
			t.Errorf("expected default sort relevance, got %s", p.SortBy)
		}
		if p.SortOrder != "desc" {
			t.Errorf("expected default sort order desc, got %s", p.SortOrder)
		}
	})
}

func TestIsDOI(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"10.1086/260062", true},
		{"10.1234/test.paper", true},
		{"not-a-doi", false},
		{"", false},
		{"10.abc/test", false},
	}
	for _, tt := range tests {
		if got := IsDOI(tt.input); got != tt.want {
			t.Errorf("IsDOI(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsArxivID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2301.07041", true},
		{"2301.07041v2", true},
		{"hep-th/9711200", true},
		{"not-arxiv", false},
		{"10.1086/260062", false},
	}
	for _, tt := range tests {
		if got := IsArxivID(tt.input); got != tt.want {
			t.Errorf("IsArxivID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestExtractDOI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://doi.org/10.1086/260062", "10.1086/260062"},
		{"doi:10.1234/test", "10.1234/test"},
		{"10.1234/direct", "10.1234/direct"},
		{"no doi here", ""},
	}
	for _, tt := range tests {
		if got := ExtractDOI(tt.input); got != tt.want {
			t.Errorf("ExtractDOI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
