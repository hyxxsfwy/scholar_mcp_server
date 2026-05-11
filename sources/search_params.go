package sources

import (
	"fmt"
	"strings"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/adsabs"
	"github.com/Seelly/scholar_mcp_server/sources/brokerresearch"
	"github.com/Seelly/scholar_mcp_server/sources/crossref"
	"github.com/Seelly/scholar_mcp_server/sources/googlescholar"
	"github.com/Seelly/scholar_mcp_server/sources/scopus"
	"github.com/Seelly/scholar_mcp_server/sources/semanticscholar"
)

func hasCrossrefStructuredFilters(params common.SearchParams) bool {
	return params.Author != "" || params.Title != "" || params.Journal != "" ||
		params.Publisher != "" || params.Type != "" || params.Year != "" || params.YearRange != ""
}

func hasSemanticScholarStructuredFilters(params common.SearchParams) bool {
	return params.Journal != "" || params.Year != "" || params.YearRange != "" ||
		params.Type != "" || params.MinCitations > 0 || len(params.Categories) > 0
}

func hasScopusStructuredFilters(params common.SearchParams) bool {
	return params.Author != "" || params.Title != "" || params.Journal != "" ||
		params.Year != "" || params.YearRange != "" || params.Language != "" ||
		params.Type != "" || len(params.Categories) > 0
}

func hasAdsabsStructuredFilters(params common.SearchParams) bool {
	return params.Author != "" || params.Title != "" || params.Abstract != "" ||
		params.Journal != "" || params.Year != "" || params.YearRange != "" || params.Type != ""
}

func hasArxivStructuredFilters(params common.SearchParams) bool {
	return params.Title != "" || params.Author != "" || params.Abstract != "" ||
		params.Journal != "" || len(params.Categories) > 0
}

func hasGoogleScholarStructuredFilters(params common.SearchParams) bool {
	return params.Title != "" || params.Author != "" || params.Abstract != "" ||
		params.Journal != "" || params.Year != "" || params.YearRange != "" || len(params.Categories) > 0
}

func buildCrossrefSearchParam(params common.SearchParams) *crossref.CrossrefSearchParam {
	searchParams := &crossref.CrossrefSearchParam{
		Query:     params.Query,
		Author:    params.Author,
		Title:     params.Title,
		Container: params.Journal,
		Publisher: params.Publisher,
		Type:      params.Type,
		Offset:    params.Offset,
		Rows:      params.Limit,
	}

	searchParams.FromDate, searchParams.UntilDate = buildCrossrefDateRange(params)

	return searchParams
}

func buildSemanticScholarSearchParam(params common.SearchParams) *semanticscholar.SemanticScholarSearchParam {
	searchParams := &semanticscholar.SemanticScholarSearchParam{
		Query:            params.Query,
		Year:             firstNonEmpty(params.YearRange, params.Year),
		Venue:            singleValueSlice(params.Journal),
		FieldsOfStudy:    append([]string(nil), params.Categories...),
		PublicationTypes: singleValueSlice(params.Type),
		MinCitationCount: params.MinCitations,
		Offset:           params.Offset,
		Limit:            params.Limit,
	}

	return searchParams
}

func buildScopusSearchParam(params common.SearchParams) *scopus.ScopusSearchParam {
	searchParams := &scopus.ScopusSearchParam{
		Query:       params.Query,
		Author:      params.Author,
		Title:       params.Title,
		Journal:     params.Journal,
		Year:        firstNonEmpty(params.YearRange, params.Year),
		SubjectArea: strings.Join(params.Categories, " OR "),
		Language:    params.Language,
		DocType:     params.Type,
		Start:       params.Offset,
		Count:       params.Limit,
	}

	return searchParams
}

func buildAdsabsSearchParam(params common.SearchParams) *adsabs.AdsabsSearchParam {
	searchParams := &adsabs.AdsabsSearchParam{
		Query:    params.Query,
		Author:   params.Author,
		Title:    params.Title,
		Abstract: params.Abstract,
		Year:     firstNonEmpty(params.YearRange, params.Year),
		Pub:      params.Journal,
		DocType:  params.Type,
		Start:    params.Offset,
		Rows:     params.Limit,
	}

	return searchParams
}

func buildGoogleScholarSearchParam(params common.SearchParams) *googlescholar.GoogleScholarSearchParam {
	return &googlescholar.GoogleScholarSearchParam{
		Query:     buildGoogleScholarQuery(params),
		Author:    params.Author,
		Journal:   params.Journal,
		Year:      params.Year,
		YearRange: params.YearRange,
		Offset:    params.Offset,
		Limit:     params.Limit,
		SortBy:    params.SortBy,
		SortOrder: params.SortOrder,
	}
}

func buildBrokerResearchSearchParam(params common.SearchParams) *brokerresearch.SearchParam {
	return &brokerresearch.SearchParam{
		Query:          firstNonEmpty(params.Query, common.DeriveSearchQuery(params)),
		Author:         params.Author,
		Title:          params.Title,
		Abstract:       params.Abstract,
		Journal:        params.Journal,
		Publisher:      params.Publisher,
		Year:           params.Year,
		YearRange:      params.YearRange,
		Language:       params.Language,
		Type:           params.Type,
		Categories:     append([]string(nil), params.Categories...),
		OpenAccessOnly: params.OpenAccessOnly,
		Offset:         params.Offset,
		Limit:          params.Limit,
		SortBy:         params.SortBy,
		SortOrder:      params.SortOrder,
	}
}

func buildGoogleScholarQuery(params common.SearchParams) string {
	var clauses []string
	derivedQuery := common.DeriveSearchQuery(params)

	if params.Query != "" && (!hasGoogleScholarStructuredFilters(params) || params.Query != derivedQuery) {
		clauses = append(clauses, params.Query)
	}
	if params.Title != "" {
		clauses = append(clauses, fmt.Sprintf("intitle:%q", params.Title))
	}
	if params.Abstract != "" {
		clauses = append(clauses, params.Abstract)
	}
	if len(params.Categories) > 0 {
		clauses = append(clauses, strings.Join(params.Categories, " "))
	}

	if len(clauses) == 0 {
		return firstNonEmpty(params.Query, derivedQuery)
	}

	return strings.Join(clauses, " ")
}

func buildArxivSearchQuery(params common.SearchParams) string {
	var clauses []string

	derivedQuery := deriveArxivTextQuery(params)
	if params.Query != "" && (!hasArxivStructuredFilters(params) || params.Query != derivedQuery) {
		clauses = append(clauses, buildArxivFieldClause("all", params.Query))
	}
	if params.Title != "" {
		clauses = append(clauses, buildArxivFieldClause("ti", params.Title))
	}
	if params.Author != "" {
		clauses = append(clauses, buildArxivFieldClause("au", params.Author))
	}
	if params.Abstract != "" {
		clauses = append(clauses, buildArxivFieldClause("abs", params.Abstract))
	}
	if params.Journal != "" {
		clauses = append(clauses, buildArxivFieldClause("jr", params.Journal))
	}
	if categoryClause := buildArxivCategoryClause(params.Categories); categoryClause != "" {
		clauses = append(clauses, categoryClause)
	}

	return strings.Join(clauses, " AND ")
}

func deriveArxivTextQuery(params common.SearchParams) string {
	parts := []string{params.Title, params.Author, params.Abstract, params.Journal}
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			terms = append(terms, part)
		}
	}

	return strings.Join(terms, " ")
}

func buildArxivFieldClause(field, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	return fmt.Sprintf("%s:%q", field, value)
}

func buildArxivCategoryClause(categories []string) string {
	var clauses []string
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category == "" {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("cat:%s", category))
	}

	switch len(clauses) {
	case 0:
		return ""
	case 1:
		return clauses[0]
	default:
		return fmt.Sprintf("(%s)", strings.Join(clauses, " OR "))
	}
}

func buildCrossrefDateRange(params common.SearchParams) (string, string) {
	year := strings.TrimSpace(firstNonEmpty(params.YearRange, params.Year))
	if year == "" {
		return "", ""
	}

	parts := strings.Split(year, "-")
	if len(parts) == 1 && len(parts[0]) == 4 {
		return parts[0] + "-01-01", parts[0] + "-12-31"
	}
	if len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 4 {
		return parts[0] + "-01-01", parts[1] + "-12-31"
	}

	return "", ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

func singleValueSlice(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return []string{value}
}
