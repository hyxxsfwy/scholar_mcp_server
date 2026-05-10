package sources

import (
	"context"
	"strconv"

	"github.com/Seelly/scholar_mcp_server/sources/semanticscholar"

	"github.com/Seelly/scholar_mcp_server/common"
)

// SemanticScholarSource Semantic Scholar数据源实现
type SemanticScholarSource struct {
	client *semanticscholar.SemanticScholarClient
	config SourceConfig
}

// NewSemanticScholarSource 创建新的Semantic Scholar数据源
func NewSemanticScholarSource(config SourceConfig) *SemanticScholarSource {
	var client *semanticscholar.SemanticScholarClient
	if config.APIKey != "" {
		client = semanticscholar.NewSemanticScholarClientWithAPIKey(config.APIKey)
	} else {
		client = semanticscholar.NewSemanticScholarClient()
	}

	return &SemanticScholarSource{
		client: client,
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (s *SemanticScholarSource) GetSourceName() string {
	return "semantic_scholar"
}

// SearchPapers 搜索论文
func (s *SemanticScholarSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	var (
		result *semanticscholar.SearchResult
		err    error
	)

	if hasSemanticScholarStructuredFilters(params) {
		result, err = s.client.SearchPapersWithParams(ctx, buildSemanticScholarSearchParam(params))
	} else {
		result, err = s.client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	}
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authorNames []string
		for _, author := range paper.Authors {
			authorNames = append(authorNames, author.Name)
		}

		unified := common.UnifiedPaper{
			ID:            paper.PaperID,
			DOI:           paper.ExternalIDs.DOI,
			Title:         paper.Title,
			Abstract:      paper.Abstract,
			Authors:       authorNames,
			Journal:       paper.Venue,
			PublishedDate: strconv.Itoa(paper.Year),
			Categories:    paper.FieldsOfStudy,
			CitationCount: paper.CitationCount,
			ReadCount:     0, // Semantic Scholar不提供阅读数
			URL:           paper.URL,
			IsOpenAccess:  paper.IsOpenAccess,
			Type:          "article",
			Source:        "semantic_scholar",
			ExternalIDs: map[string]string{
				"semantic_scholar": paper.PaperID,
				"doi":              paper.ExternalIDs.DOI,
				"arxiv":            paper.ExternalIDs.ArXiv,
				"pubmed":           paper.ExternalIDs.PubMed,
				"dblp":             paper.ExternalIDs.DBLP,
			},
			Metadata: map[string]interface{}{
				"corpus_id":                  paper.CorpusID,
				"reference_count":            paper.ReferenceCount,
				"influential_citation_count": paper.InfluentialCitationCount,
				"publication_types":          paper.PublicationTypes,
				"journal":                    paper.Journal,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.Total, nil
}

// GetPaper 获取特定论文
func (s *SemanticScholarSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := s.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	var authorNames []string
	for _, author := range paper.Authors {
		authorNames = append(authorNames, author.Name)
	}

	unified := &common.UnifiedPaper{
		ID:            paper.PaperID,
		DOI:           paper.ExternalIDs.DOI,
		Title:         paper.Title,
		Abstract:      paper.Abstract,
		Authors:       authorNames,
		Journal:       paper.Venue,
		PublishedDate: strconv.Itoa(paper.Year),
		Categories:    paper.FieldsOfStudy,
		CitationCount: paper.CitationCount,
		URL:           paper.URL,
		IsOpenAccess:  paper.IsOpenAccess,
		Type:          "article",
		Source:        "semantic_scholar",
		ExternalIDs: map[string]string{
			"semantic_scholar": paper.PaperID,
			"doi":              paper.ExternalIDs.DOI,
			"arxiv":            paper.ExternalIDs.ArXiv,
			"pubmed":           paper.ExternalIDs.PubMed,
		},
		Metadata: map[string]interface{}{
			"corpus_id":                  paper.CorpusID,
			"reference_count":            paper.ReferenceCount,
			"influential_citation_count": paper.InfluentialCitationCount,
		},
	}

	return unified, nil
}

// IsAvailable 检查数据源是否可用
func (s *SemanticScholarSource) IsAvailable(ctx context.Context) bool {
	_, err := s.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

// GetCapabilities 获取数据源能力
func (s *SemanticScholarSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          false,
		RequiresAPIKey:       false, // 可选
		SupportedFields:      []string{"title", "author", "venue", "year", "fieldsOfStudy"},
		MaxResultsPerPage:    100,
	}
}
