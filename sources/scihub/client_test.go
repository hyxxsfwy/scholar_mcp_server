package scihub

import (
	"context"
	"testing"
)

func TestSearchPapers_RejectNonDOI(t *testing.T) {
	client := NewSciHubClient()
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"valid DOI", "10.1086/260062", false},
		{"keyword query", "machine learning", true},
		{"title query", "Portfolio Selection", true},
		{"empty query", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SearchPapers(ctx, tt.query, 10)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchPapers(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
			}
		})
	}
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
	}
	for _, tt := range tests {
		if got := isDOI(tt.input); got != tt.want {
			t.Errorf("isDOI(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
