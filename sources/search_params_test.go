package sources

import (
	"reflect"
	"testing"

	"github.com/Seelly/scholar_mcp_server/common"
)

func TestBuildCrossrefSearchParam(t *testing.T) {
	params := common.SearchParams{
		Query:     "Portfolio Selection",
		Author:    "Harry Markowitz",
		Title:     "Portfolio Selection",
		Journal:   "The Journal of Finance",
		Publisher: "Wiley",
		Type:      "journal-article",
		Year:      "1952",
		Offset:    5,
		Limit:     20,
	}

	searchParams := buildCrossrefSearchParam(params)
	if searchParams.Query != params.Query {
		t.Fatalf("expected query %q, got %q", params.Query, searchParams.Query)
	}
	if searchParams.Author != params.Author || searchParams.Title != params.Title {
		t.Fatalf("expected author/title filters to be preserved: %+v", searchParams)
	}
	if searchParams.Container != params.Journal {
		t.Fatalf("expected container %q, got %q", params.Journal, searchParams.Container)
	}
	if searchParams.FromDate != "1952-01-01" || searchParams.UntilDate != "1952-12-31" {
		t.Fatalf("expected 1952 date window, got %q - %q", searchParams.FromDate, searchParams.UntilDate)
	}
	if searchParams.Offset != 5 || searchParams.Rows != 20 {
		t.Fatalf("expected paging to be preserved, got offset=%d rows=%d", searchParams.Offset, searchParams.Rows)
	}
}

func TestBuildSemanticScholarSearchParam(t *testing.T) {
	params := common.SearchParams{
		Query:        "deep learning finance",
		Journal:      "Quantitative Finance",
		Categories:   []string{"Finance", "Machine Learning"},
		Year:         "2022",
		Type:         "Review",
		MinCitations: 10,
		Offset:       3,
		Limit:        7,
	}

	searchParams := buildSemanticScholarSearchParam(params)
	if searchParams.Query != params.Query || searchParams.Year != params.Year {
		t.Fatalf("expected query/year to be preserved: %+v", searchParams)
	}
	if !reflect.DeepEqual(searchParams.Venue, []string{"Quantitative Finance"}) {
		t.Fatalf("expected venue filter, got %#v", searchParams.Venue)
	}
	if !reflect.DeepEqual(searchParams.FieldsOfStudy, params.Categories) {
		t.Fatalf("expected fields of study %v, got %v", params.Categories, searchParams.FieldsOfStudy)
	}
	if !reflect.DeepEqual(searchParams.PublicationTypes, []string{"Review"}) {
		t.Fatalf("expected publication type filter, got %#v", searchParams.PublicationTypes)
	}
	if searchParams.MinCitationCount != 10 || searchParams.Offset != 3 || searchParams.Limit != 7 {
		t.Fatalf("expected min citations and paging to be preserved: %+v", searchParams)
	}
}

func TestBuildScopusSearchParam(t *testing.T) {
	params := common.SearchParams{
		Query:      "limit order book",
		Author:     "Zihao Zhang",
		Title:      "DeepLOB",
		Journal:    "IEEE Transactions on Signal Processing",
		Categories: []string{"COMP", "MATH"},
		Year:       "2019",
		Language:   "en",
		Type:       "ar",
		Offset:     2,
		Limit:      15,
	}

	searchParams := buildScopusSearchParam(params)
	if searchParams.Author != params.Author || searchParams.Title != params.Title {
		t.Fatalf("expected author/title filters to be preserved: %+v", searchParams)
	}
	if searchParams.SubjectArea != "COMP OR MATH" {
		t.Fatalf("expected subject area clause, got %q", searchParams.SubjectArea)
	}
	if searchParams.Journal != params.Journal || searchParams.Year != params.Year {
		t.Fatalf("expected journal/year filters to be preserved: %+v", searchParams)
	}
	if searchParams.Language != "en" || searchParams.DocType != "ar" {
		t.Fatalf("expected language/type filters to be preserved: %+v", searchParams)
	}
	if searchParams.Start != 2 || searchParams.Count != 15 {
		t.Fatalf("expected paging to be preserved: %+v", searchParams)
	}
}

func TestBuildAdsabsSearchParam(t *testing.T) {
	params := common.SearchParams{
		Query:    "stochastic volatility",
		Author:   "Steven Heston",
		Title:    "Closed-Form Solution",
		Abstract: "bond and currency options",
		Journal:  "Review of Financial Studies",
		Year:     "1993",
		Type:     "article",
		Offset:   1,
		Limit:    12,
	}

	searchParams := buildAdsabsSearchParam(params)
	if searchParams.Query != params.Query || searchParams.Author != params.Author {
		t.Fatalf("expected query/author filters to be preserved: %+v", searchParams)
	}
	if searchParams.Title != params.Title || searchParams.Abstract != params.Abstract {
		t.Fatalf("expected title/abstract filters to be preserved: %+v", searchParams)
	}
	if searchParams.Pub != params.Journal || searchParams.DocType != params.Type || searchParams.Year != params.Year {
		t.Fatalf("expected journal/type/year filters to be preserved: %+v", searchParams)
	}
	if searchParams.Start != 1 || searchParams.Rows != 12 {
		t.Fatalf("expected paging to be preserved: %+v", searchParams)
	}
}

func TestBuildGoogleScholarSearchParam(t *testing.T) {
	params := common.SearchParams{
		Query:      "machine learning",
		Title:      "Empirical Asset Pricing",
		Author:     "Shihao Gu",
		Journal:    "Review of Financial Studies",
		Categories: []string{"Finance", "Asset Pricing"},
		YearRange:  "2020-2023",
		Offset:     4,
		Limit:      8,
		SortBy:     "published_date",
		SortOrder:  "desc",
	}

	searchParams := buildGoogleScholarSearchParam(params)
	if searchParams.Query != `machine learning intitle:"Empirical Asset Pricing" Finance Asset Pricing` {
		t.Fatalf("expected Google Scholar query to preserve explicit and structured terms, got %q", searchParams.Query)
	}
	if searchParams.Author != params.Author || searchParams.Journal != params.Journal || searchParams.YearRange != params.YearRange {
		t.Fatalf("expected structured filters to be preserved: %+v", searchParams)
	}
	if searchParams.Offset != 4 || searchParams.Limit != 8 || searchParams.SortBy != "published_date" {
		t.Fatalf("expected paging and sort to be preserved: %+v", searchParams)
	}
}

func TestBuildGoogleScholarSearchParamDerivesQueryForAuthorOnly(t *testing.T) {
	params := common.SearchParams{Author: "Harry Markowitz", Limit: 5}

	searchParams := buildGoogleScholarSearchParam(params)
	if searchParams.Query != "Harry Markowitz" {
		t.Fatalf("expected author-only query to be derived, got %q", searchParams.Query)
	}
	if searchParams.Author != "Harry Markowitz" {
		t.Fatalf("expected author filter to be preserved, got %q", searchParams.Author)
	}
}

func TestBuildArxivSearchQuery(t *testing.T) {
	t.Run("uses fielded clauses for structured filters", func(t *testing.T) {
		params := common.SearchParams{
			Query:      "Portfolio Selection Harry Markowitz",
			Title:      "Portfolio Selection",
			Author:     "Harry Markowitz",
			Categories: []string{"q-fin.PM", "q-fin.TR"},
		}

		query := buildArxivSearchQuery(params)
		expected := `ti:"Portfolio Selection" AND au:"Harry Markowitz" AND (cat:q-fin.PM OR cat:q-fin.TR)`
		if query != expected {
			t.Fatalf("expected %q, got %q", expected, query)
		}
	})

	t.Run("keeps explicit query when it differs from derived filters", func(t *testing.T) {
		params := common.SearchParams{
			Query:  "stochastic process",
			Title:  "volatility model",
			Author: "Jane Doe",
		}

		query := buildArxivSearchQuery(params)
		expected := `all:"stochastic process" AND ti:"volatility model" AND au:"Jane Doe"`
		if query != expected {
			t.Fatalf("expected %q, got %q", expected, query)
		}
	})
}
