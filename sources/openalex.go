package sources

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/openalex"
)

type OpenAlexSource struct {
	client *openalex.OpenAlexClient
	config SourceConfig
}

func NewOpenAlexSource(config SourceConfig) *OpenAlexSource {
	client := openalex.NewOpenAlexClient(config.APIKey)
	if config.BaseURL != "" {
		client = openalex.NewOpenAlexClientWithBaseURL(config.APIKey, config.BaseURL)
	}

	return &OpenAlexSource{client: client, config: config}
}

func (source *OpenAlexSource) GetSourceName() string {
	return "openalex"
}

func (source *OpenAlexSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	result, err := source.client.SearchPapersWithParams(ctx, buildOpenAlexSearchParam(params))
	if err != nil {
		return nil, 0, err
	}

	papers := make([]common.UnifiedPaper, 0, len(result.Papers))
	for _, paper := range result.Papers {
		papers = append(papers, newOpenAlexUnifiedPaper(paper))
	}

	return papers, result.Total, nil
}

func (source *OpenAlexSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := source.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	unified := newOpenAlexUnifiedPaper(*paper)
	return &unified, nil
}

func (source *OpenAlexSource) IsAvailable(ctx context.Context) bool {
	_, err := source.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

func (source *OpenAlexSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          true,
		RequiresAPIKey:       false,
		SupportedFields:      []string{"query", "title", "author", "journal", "year", "type", "open_access"},
		MaxResultsPerPage:    openalex.MaxLimit,
	}
}

func newOpenAlexUnifiedPaper(paper openalex.Paper) common.UnifiedPaper {
	doi := normalizeOpenAlexDOI(firstNonEmpty(paper.DOI, paper.IDs["doi"]))
	pdfURL := firstOpenAlexPDFURL(paper)
	url := firstNonEmpty(openAlexLandingPageURL(paper), paper.OpenAccess.OAURL, doiURL(doi), paper.ID)

	return common.UnifiedPaper{
		ID:            firstNonEmpty(paper.ID, paper.IDs["openalex"]),
		DOI:           doi,
		Title:         firstNonEmpty(paper.Title, paper.DisplayName),
		Abstract:      openAlexAbstract(paper.AbstractInvertedIndex),
		Authors:       openAlexAuthors(paper.Authorships),
		Journal:       openAlexJournal(paper),
		PublishedDate: openAlexPublishedDate(paper),
		Volume:        paper.Biblio.Volume,
		Issue:         paper.Biblio.Issue,
		Pages:         openAlexPages(paper.Biblio),
		Categories:    openAlexConcepts(paper.Concepts),
		Keywords:      openAlexKeywords(paper.Keywords),
		CitationCount: paper.CitedByCount,
		URL:           url,
		PDFURL:        pdfURL,
		IsOpenAccess:  paper.OpenAccess.IsOA || pdfURL != "" || hasOpenAlexOALocation(paper),
		Type:          firstNonEmpty(paper.Type, paper.TypeCrossref),
		Source:        "openalex",
		ExternalIDs:   openAlexExternalIDs(paper, doi),
		Metadata: map[string]interface{}{
			"open_access":       paper.OpenAccess,
			"primary_location":  paper.PrimaryLocation,
			"best_oa_location":  paper.BestOALocation,
			"locations":         paper.Locations,
			"type_crossref":     paper.TypeCrossref,
			"publication_year":  paper.PublicationYear,
			"concepts":          paper.Concepts,
			"institutions":      openAlexInstitutions(paper.Authorships),
			"source_record_ids": paper.IDs,
		},
	}
}

func normalizeOpenAlexDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "https://dx.doi.org/", "http://dx.doi.org/", "doi:"} {
		if strings.HasPrefix(strings.ToLower(doi), prefix) {
			return strings.TrimSpace(doi[len(prefix):])
		}
	}

	return doi
}

func doiURL(doi string) string {
	if doi == "" {
		return ""
	}

	return "https://doi.org/" + doi
}

func openAlexPublishedDate(paper openalex.Paper) string {
	if paper.PublicationDate != "" {
		return paper.PublicationDate
	}
	if paper.PublicationYear > 0 {
		return strconv.Itoa(paper.PublicationYear)
	}

	return ""
}

func openAlexJournal(paper openalex.Paper) string {
	if paper.PrimaryLocation.Source != nil && paper.PrimaryLocation.Source.DisplayName != "" {
		return paper.PrimaryLocation.Source.DisplayName
	}
	if paper.BestOALocation != nil && paper.BestOALocation.Source != nil && paper.BestOALocation.Source.DisplayName != "" {
		return paper.BestOALocation.Source.DisplayName
	}
	for _, location := range paper.Locations {
		if location.Source != nil && location.Source.DisplayName != "" {
			return location.Source.DisplayName
		}
	}

	return ""
}

func openAlexLandingPageURL(paper openalex.Paper) string {
	if paper.PrimaryLocation.LandingPageURL != "" {
		return paper.PrimaryLocation.LandingPageURL
	}
	if paper.BestOALocation != nil && paper.BestOALocation.LandingPageURL != "" {
		return paper.BestOALocation.LandingPageURL
	}
	for _, location := range paper.Locations {
		if location.LandingPageURL != "" {
			return location.LandingPageURL
		}
	}

	return ""
}

func firstOpenAlexPDFURL(paper openalex.Paper) string {
	if paper.PrimaryLocation.PDFURL != "" {
		return paper.PrimaryLocation.PDFURL
	}
	if paper.BestOALocation != nil && paper.BestOALocation.PDFURL != "" {
		return paper.BestOALocation.PDFURL
	}
	for _, location := range paper.Locations {
		if location.PDFURL != "" {
			return location.PDFURL
		}
	}

	return ""
}

func hasOpenAlexOALocation(paper openalex.Paper) bool {
	if paper.PrimaryLocation.IsOA || paper.BestOALocation != nil && paper.BestOALocation.IsOA {
		return true
	}
	for _, location := range paper.Locations {
		if location.IsOA {
			return true
		}
	}

	return false
}

func openAlexAuthors(authorships []openalex.Authorship) []string {
	authors := make([]string, 0, len(authorships))
	for _, authorship := range authorships {
		if strings.TrimSpace(authorship.Author.DisplayName) != "" {
			authors = append(authors, strings.TrimSpace(authorship.Author.DisplayName))
		}
	}

	return authors
}

func openAlexInstitutions(authorships []openalex.Authorship) []string {
	seen := make(map[string]struct{})
	var institutions []string
	for _, authorship := range authorships {
		for _, institution := range authorship.Institutions {
			name := strings.TrimSpace(institution.DisplayName)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			institutions = append(institutions, name)
		}
	}

	return institutions
}

func openAlexConcepts(concepts []openalex.Concept) []string {
	categories := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if strings.TrimSpace(concept.DisplayName) != "" {
			categories = append(categories, concept.DisplayName)
		}
	}

	return categories
}

func openAlexKeywords(keywords []openalex.Keyword) []string {
	values := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if strings.TrimSpace(keyword.DisplayName) != "" {
			values = append(values, keyword.DisplayName)
		}
	}

	return values
}

func openAlexPages(biblio openalex.Biblio) string {
	switch {
	case biblio.FirstPage != "" && biblio.LastPage != "":
		return biblio.FirstPage + "-" + biblio.LastPage
	case biblio.FirstPage != "":
		return biblio.FirstPage
	default:
		return ""
	}
}

func openAlexExternalIDs(paper openalex.Paper, doi string) map[string]string {
	externalIDs := make(map[string]string)
	for key, value := range paper.IDs {
		if strings.TrimSpace(value) != "" {
			externalIDs[key] = value
		}
	}
	if paper.ID != "" {
		externalIDs["openalex"] = paper.ID
	}
	if doi != "" {
		externalIDs["doi"] = doi
	}

	return externalIDs
}

func openAlexAbstract(invertedIndex map[string][]int) string {
	if len(invertedIndex) == 0 {
		return ""
	}

	positionToWord := make(map[int]string)
	positions := make([]int, 0)
	for word, wordPositions := range invertedIndex {
		for _, position := range wordPositions {
			positionToWord[position] = word
			positions = append(positions, position)
		}
	}
	sort.Ints(positions)

	words := make([]string, 0, len(positions))
	for _, position := range positions {
		words = append(words, positionToWord[position])
	}

	return strings.Join(words, " ")
}

func buildOpenAlexSearchParam(params common.SearchParams) *openalex.OpenAlexSearchParam {
	return &openalex.OpenAlexSearchParam{
		Query:          buildOpenAlexQuery(params),
		Year:           params.Year,
		YearRange:      params.YearRange,
		Type:           params.Type,
		OpenAccessOnly: params.OpenAccessOnly,
		MinCitations:   params.MinCitations,
		Offset:         params.Offset,
		Limit:          params.Limit,
		SortBy:         params.SortBy,
		SortOrder:      params.SortOrder,
	}
}

func buildOpenAlexQuery(params common.SearchParams) string {
	var terms []string
	derivedQuery := common.DeriveSearchQuery(params)
	if params.Query != "" && params.Query != derivedQuery {
		terms = append(terms, params.Query)
	}
	for _, value := range []string{params.Title, params.Author, params.Abstract, params.Journal, strings.Join(params.Categories, " ")} {
		value = strings.TrimSpace(value)
		if value != "" {
			terms = append(terms, value)
		}
	}
	if len(terms) == 0 {
		return firstNonEmpty(params.Query, derivedQuery)
	}

	return strings.Join(terms, " ")
}
