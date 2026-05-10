package sources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/googlescholar"
)

var googleScholarYearPattern = regexp.MustCompile(`\b(?:18|19|20)\d{2}\b`)

type GoogleScholarSource struct {
	client *googlescholar.GoogleScholarClient
	config SourceConfig
}

func NewGoogleScholarSource(config SourceConfig) *GoogleScholarSource {
	client := googlescholar.NewGoogleScholarClient(config.APIKey)
	if config.BaseURL != "" {
		client = googlescholar.NewGoogleScholarClientWithBaseURL(config.APIKey, config.BaseURL)
	}

	return &GoogleScholarSource{
		client: client,
		config: config,
	}
}

func (source *GoogleScholarSource) GetSourceName() string {
	return "google_scholar"
}

func (source *GoogleScholarSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	result, err := source.client.SearchPapersWithParams(ctx, buildGoogleScholarSearchParam(params))
	if err != nil {
		return nil, 0, err
	}

	papers := make([]common.UnifiedPaper, 0, len(result.Papers))
	for _, paper := range result.Papers {
		papers = append(papers, newGoogleScholarUnifiedPaper(paper))
	}

	return papers, result.Total, nil
}

func (source *GoogleScholarSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, fmt.Errorf("论文标识符不能为空")
	}

	result, err := source.client.SearchPapers(ctx, identifier, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(result.Papers) == 0 {
		return nil, fmt.Errorf("未找到标识符为 %s 的论文", identifier)
	}

	unified := newGoogleScholarUnifiedPaper(result.Papers[0])
	return &unified, nil
}

func (source *GoogleScholarSource) IsAvailable(ctx context.Context) bool {
	if strings.TrimSpace(source.config.APIKey) == "" {
		return false
	}

	_, err := source.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

func (source *GoogleScholarSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          true,
		RequiresAPIKey:       true,
		SupportedFields:      []string{"query", "title", "author", "journal", "year"},
		MaxResultsPerPage:    googlescholar.MaxLimit,
	}
}

func newGoogleScholarUnifiedPaper(paper googlescholar.Paper) common.UnifiedPaper {
	pdfURL := firstGoogleScholarPDFURL(paper.Resources)
	doi := common.ExtractDOI(strings.Join([]string{paper.Link, paper.Snippet, paper.Title}, " "))

	return common.UnifiedPaper{
		ID:            googleScholarPaperID(paper),
		DOI:           doi,
		Title:         paper.Title,
		Abstract:      paper.Snippet,
		Authors:       googleScholarAuthors(paper.PublicationInfo),
		Journal:       googleScholarJournal(paper.PublicationInfo.Summary),
		PublishedDate: googleScholarPublishedYear(paper.PublicationInfo.Summary),
		CitationCount: paper.InlineLinks.CitedBy.Total,
		URL:           paper.Link,
		PDFURL:        pdfURL,
		IsOpenAccess:  pdfURL != "",
		Type:          "article",
		Source:        "google_scholar",
		ExternalIDs: map[string]string{
			"google_scholar": paper.ResultID,
			"doi":            doi,
		},
		Metadata: map[string]interface{}{
			"position":            paper.Position,
			"publication_summary": paper.PublicationInfo.Summary,
			"cited_by_url":        paper.InlineLinks.CitedBy.Link,
			"cites_id":            paper.InlineLinks.CitedBy.CitesID,
			"versions_url":        paper.InlineLinks.Versions.Link,
			"versions_count":      paper.InlineLinks.Versions.Total,
			"related_pages_link":  paper.InlineLinks.RelatedPagesLink,
			"serpapi_result_id":   paper.ResultID,
			"available_resources": paper.Resources,
		},
	}
}

func googleScholarPaperID(paper googlescholar.Paper) string {
	if strings.TrimSpace(paper.ResultID) != "" {
		return paper.ResultID
	}
	if strings.TrimSpace(paper.Link) != "" {
		return paper.Link
	}

	return paper.Title
}

func googleScholarAuthors(publicationInfo googlescholar.PublicationInfo) []string {
	authors := make([]string, 0, len(publicationInfo.Authors))
	for _, author := range publicationInfo.Authors {
		if strings.TrimSpace(author.Name) != "" {
			authors = append(authors, strings.TrimSpace(author.Name))
		}
	}
	if len(authors) > 0 {
		return authors
	}

	summaryParts := strings.Split(publicationInfo.Summary, " - ")
	if len(summaryParts) == 0 {
		return nil
	}

	return common.SplitAuthors(summaryParts[0])
}

func googleScholarJournal(summary string) string {
	summaryParts := strings.Split(summary, " - ")
	if len(summaryParts) < 2 {
		return ""
	}

	venue := strings.TrimSpace(summaryParts[1])
	year := googleScholarPublishedYear(summary)
	if year != "" {
		venue = strings.TrimSpace(strings.TrimSuffix(venue, year))
		venue = strings.TrimSpace(strings.TrimSuffix(venue, ","))
	}

	return venue
}

func googleScholarPublishedYear(summary string) string {
	return googleScholarYearPattern.FindString(summary)
}

func firstGoogleScholarPDFURL(resources []googlescholar.Resource) string {
	for _, resource := range resources {
		if resource.Link == "" {
			continue
		}
		fileFormat := strings.ToLower(resource.FileFormat)
		title := strings.ToLower(resource.Title)
		if strings.Contains(fileFormat, "pdf") || strings.Contains(title, "pdf") {
			return resource.Link
		}
	}

	return ""
}
