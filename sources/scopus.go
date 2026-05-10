package sources

import (
	"context"
	"fmt"

	"github.com/Seelly/scholar_mcp_server/sources/scopus"

	"github.com/Seelly/scholar_mcp_server/common"
)

// ScopusSource Scopus数据源实现
type ScopusSource struct {
	client *scopus.ScopusClient
	config SourceConfig
}

// NewScopusSource 创建新的Scopus数据源
func NewScopusSource(config SourceConfig) *ScopusSource {
	if config.APIKey == "" {
		return nil // Scopus需要API密钥
	}

	return &ScopusSource{
		client: scopus.NewScopusClient(config.APIKey),
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (s *ScopusSource) GetSourceName() string {
	return "scopus"
}

// SearchPapers 搜索论文
func (s *ScopusSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	if s.client == nil {
		return nil, 0, fmt.Errorf("Scopus client not initialized - API key required")
	}

	var (
		result *scopus.SearchResult
		err    error
	)

	if hasScopusStructuredFilters(params) {
		result, err = s.client.SearchPapersWithParams(ctx, buildScopusSearchParam(params))
	} else {
		result, err = s.client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	}
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authorNames []string
		for _, author := range paper.Author {
			authorNames = append(authorNames, author.AuthorName)
		}

		var categories []string
		for _, subject := range paper.SubjectAreas {
			categories = append(categories, subject.Name)
		}

		unified := common.UnifiedPaper{
			ID:            paper.EID,
			DOI:           paper.DOI,
			Title:         paper.Title,
			Abstract:      paper.Abstract,
			Authors:       authorNames,
			Journal:       paper.PublicationName,
			Publisher:     paper.Publisher,
			PublishedDate: paper.CoverDate,
			Volume:        paper.Volume,
			Issue:         paper.IssueIdentifier,
			Pages:         paper.PageRange,
			Categories:    categories,
			CitationCount: paper.CitedByCount,
			Language:      paper.Language,
			URL:           paper.URL,
			Type:          paper.SubtypeDescription,
			Source:        "scopus",
			ExternalIDs: map[string]string{
				"scopus": paper.EID,
				"doi":    paper.DOI,
			},
			Metadata: map[string]interface{}{
				"aggregation_type": paper.AggregationType,
				"article_number":   paper.ArticleNumber,
				"affiliation":      paper.Affiliation,
				"fund":             paper.Fund,
				"open_access":      paper.OpenAccess,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.TotalCount, nil
}

// GetPaper 获取特定论文
func (s *ScopusSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	if s.client == nil {
		return nil, fmt.Errorf("Scopus client not initialized - API key required")
	}

	paper, err := s.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	var authorNames []string
	for _, author := range paper.Author {
		authorNames = append(authorNames, author.AuthorName)
	}

	var categories []string
	for _, subject := range paper.SubjectAreas {
		categories = append(categories, subject.Name)
	}

	unified := &common.UnifiedPaper{
		ID:            paper.EID,
		DOI:           paper.DOI,
		Title:         paper.Title,
		Abstract:      paper.Abstract,
		Authors:       authorNames,
		Journal:       paper.PublicationName,
		Publisher:     paper.Publisher,
		PublishedDate: paper.CoverDate,
		Volume:        paper.Volume,
		Issue:         paper.IssueIdentifier,
		Pages:         paper.PageRange,
		Categories:    categories,
		CitationCount: paper.CitedByCount,
		Language:      paper.Language,
		URL:           paper.URL,
		Type:          paper.SubtypeDescription,
		Source:        "scopus",
		ExternalIDs: map[string]string{
			"scopus": paper.EID,
			"doi":    paper.DOI,
		},
		Metadata: map[string]interface{}{
			"aggregation_type": paper.AggregationType,
			"article_number":   paper.ArticleNumber,
			"affiliation":      paper.Affiliation,
			"fund":             paper.Fund,
			"open_access":      paper.OpenAccess,
		},
	}

	return unified, nil
}

// IsAvailable 检查数据源是否可用
func (s *ScopusSource) IsAvailable(ctx context.Context) bool {
	if s.client == nil {
		return false
	}
	_, err := s.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

// GetCapabilities 获取数据源能力
func (s *ScopusSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          false,
		RequiresAPIKey:       true,
		SupportedFields:      []string{"title", "author", "journal", "subject", "affiliation", "year"},
		MaxResultsPerPage:    200,
	}
}
