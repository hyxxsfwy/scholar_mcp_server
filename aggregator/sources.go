package aggregator

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Seelly/scholar_mcp_server/arxiv"
	"github.com/Seelly/scholar_mcp_server/semanticscholar"
	"github.com/Seelly/scholar_mcp_server/crossref"
	"github.com/Seelly/scholar_mcp_server/scopus"
	"github.com/Seelly/scholar_mcp_server/adsabs"
	"github.com/Seelly/scholar_mcp_server/scihub"
	"github.com/Seelly/scholar_mcp_server/common"
)

// searchArxiv 搜索arXiv
func (a *ScholarAggregator) searchArxiv(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	client := arxiv.NewArxivClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		unified := common.UnifiedPaper{
			ID:            paper.ID,
			DOI:           paper.DOI,
			Title:         paper.Title,
			Abstract:      paper.Abstract,
			Authors:       paper.Authors,
			Journal:       "", // arXiv没有期刊概念
			PublishedDate: paper.Published,
			Categories:    paper.Categories,
			URL:           paper.URL,
			PDFURL:        paper.PDFURL,
			IsOpenAccess:  true, // arXiv都是开放获取
			Type:          "preprint",
			Source:        "arxiv",
			ExternalIDs: map[string]string{
				"arxiv": paper.ID,
				"doi":   paper.DOI,
			},
			Metadata: map[string]interface{}{
				"journal_ref": paper.JournalRef,
				"comment":     paper.Comment,
				"updated":     paper.Updated,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.TotalCount, nil
}

// searchSemanticScholar 搜索Semantic Scholar
func (a *ScholarAggregator) searchSemanticScholar(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	client := semanticscholar.NewSemanticScholarClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
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

// searchCrossref 搜索Crossref
func (a *ScholarAggregator) searchCrossref(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	client := crossref.NewCrossrefClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authorNames []string
		for _, author := range paper.Author {
			name := ""
			if author.Given != "" && author.Family != "" {
				name = author.Given + " " + author.Family
			} else if author.Family != "" {
				name = author.Family
			}
			if name != "" {
				authorNames = append(authorNames, name)
			}
		}

		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}

		journal := ""
		if len(paper.ContainerTitle) > 0 {
			journal = paper.ContainerTitle[0]
		}

		unified := common.UnifiedPaper{
			ID:            paper.DOI,
			DOI:           paper.DOI,
			Title:         title,
			Authors:       authorNames,
			Journal:       journal,
			Publisher:     paper.Publisher,
			Volume:        paper.Volume,
			Issue:         paper.Issue,
			Pages:         paper.Page,
			CitationCount: paper.IsReferencedByCount,
			URL:           paper.URL,
			Type:          paper.Type,
			Source:        "crossref",
			ExternalIDs: map[string]string{
				"doi": paper.DOI,
			},
			Metadata: map[string]interface{}{
				"references_count": paper.ReferencesCount,
				"license":          paper.License,
				"funder":           paper.Funder,
				"language":         paper.Language,
				"subject":          paper.Subject,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.TotalResults, nil
}

// searchScopus 搜索Scopus
func (a *ScholarAggregator) searchScopus(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	apiKey := os.Getenv("SCOPUS_API_KEY")
	if apiKey == "" {
		apiKey = a.config.ScopusAPIKey
	}
	if apiKey == "" {
		return nil, 0, fmt.Errorf("Scopus API密钥未设置")
	}

	client := scopus.NewScopusClient(apiKey)
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
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

// searchAdsabs 搜索ADSABS
func (a *ScholarAggregator) searchAdsabs(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	apiKey := os.Getenv("ADSABS_API_KEY")
	if apiKey == "" {
		apiKey = a.config.AdsabsAPIKey
	}
	if apiKey == "" {
		return nil, 0, fmt.Errorf("ADSABS API密钥未设置")
	}

	client := adsabs.NewAdsabsClient(apiKey)
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}

		doi := ""
		if len(paper.DOI) > 0 {
			doi = paper.DOI[0]
		}

		arxivID := ""
		if len(paper.ArxivID) > 0 {
			arxivID = paper.ArxivID[0]
		}

		unified := common.UnifiedPaper{
			ID:            paper.Bibcode,
			DOI:           doi,
			Title:         title,
			Abstract:      paper.Abstract,
			Authors:       paper.Author,
			Journal:       paper.Pub,
			PublishedDate: paper.PubDate,
			Volume:        paper.Volume,
			Issue:         paper.Issue,
			Categories:    paper.Keyword,
			CitationCount: paper.CitationCount,
			ReadCount:     paper.ReadCount,
			Type:          paper.DocType,
			Source:        "adsabs",
			ExternalIDs: map[string]string{
				"adsabs": paper.Bibcode,
				"doi":    doi,
				"arxiv":  arxivID,
			},
			Metadata: map[string]interface{}{
				"bibstem":    paper.Bibstem,
				"property":   paper.Property,
				"database":   paper.Database,
				"facility":   paper.Facility,
				"instrument": paper.Instrument,
				"object":     paper.Object,
				"orcid":      paper.ORCID,
				"grant":      paper.Grant,
				"aff":        paper.Aff,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.NumFound, nil
}

// searchSciHub 搜索Sci-Hub
func (a *ScholarAggregator) searchSciHub(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	client := scihub.NewSciHubClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authors []string
		if paper.Authors != "" {
			authors = common.SplitAuthors(paper.Authors)
		}

		unified := common.UnifiedPaper{
			ID:           paper.DOI,
			DOI:          paper.DOI,
			Title:        paper.Title,
			Authors:      authors,
			Journal:      paper.Journal,
			PublishedDate: paper.Year,
			PDFURL:       paper.PDFURL,
			IsOpenAccess: paper.Available,
			Type:         "article",
			Source:       "scihub",
			ExternalIDs: map[string]string{
				"doi": paper.DOI,
			},
			Metadata: map[string]interface{}{
				"scihub_url": paper.SciHubURL,
				"file_size":  paper.FileSize,
				"citation":   paper.Citation,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.Total, nil
}

// getPaperFromArxiv 从arXiv获取论文详情
func (a *ScholarAggregator) getPaperFromArxiv(ctx context.Context, id string) (*common.UnifiedPaper, error) {
	client := arxiv.NewArxivClient()
	paper, err := client.GetPaper(ctx, id)
	if err != nil {
		return nil, err
	}

	unified := &common.UnifiedPaper{
		ID:            paper.ID,
		DOI:           paper.DOI,
		Title:         paper.Title,
		Abstract:      paper.Abstract,
		Authors:       paper.Authors,
		PublishedDate: paper.Published,
		Categories:    paper.Categories,
		URL:           paper.URL,
		PDFURL:        paper.PDFURL,
		IsOpenAccess:  true,
		Type:          "preprint",
		Source:        "arxiv",
		ExternalIDs: map[string]string{
			"arxiv": paper.ID,
			"doi":   paper.DOI,
		},
		Metadata: map[string]interface{}{
			"journal_ref": paper.JournalRef,
			"comment":     paper.Comment,
			"updated":     paper.Updated,
		},
	}

	return unified, nil
}

// getPaperFromSemanticScholar 从Semantic Scholar获取论文详情
func (a *ScholarAggregator) getPaperFromSemanticScholar(ctx context.Context, id string) (*common.UnifiedPaper, error) {
	client := semanticscholar.NewSemanticScholarClient()
	paper, err := client.GetPaper(ctx, id)
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

// getPaperFromCrossref 从Crossref获取论文详情
func (a *ScholarAggregator) getPaperFromCrossref(ctx context.Context, doi string) (*common.UnifiedPaper, error) {
	client := crossref.NewCrossrefClient()
	paper, err := client.GetPaper(ctx, doi)
	if err != nil {
		return nil, err
	}

	var authorNames []string
	for _, author := range paper.Author {
		name := ""
		if author.Given != "" && author.Family != "" {
			name = author.Given + " " + author.Family
		} else if author.Family != "" {
			name = author.Family
		}
		if name != "" {
			authorNames = append(authorNames, name)
		}
	}

	title := ""
	if len(paper.Title) > 0 {
		title = paper.Title[0]
	}

	journal := ""
	if len(paper.ContainerTitle) > 0 {
		journal = paper.ContainerTitle[0]
	}

	unified := &common.UnifiedPaper{
		ID:            paper.DOI,
		DOI:           paper.DOI,
		Title:         title,
		Authors:       authorNames,
		Journal:       journal,
		Publisher:     paper.Publisher,
		Volume:        paper.Volume,
		Issue:         paper.Issue,
		Pages:         paper.Page,
		CitationCount: paper.IsReferencedByCount,
		URL:           paper.URL,
		Type:          paper.Type,
		Source:        "crossref",
		ExternalIDs: map[string]string{
			"doi": paper.DOI,
		},
		Metadata: map[string]interface{}{
			"references_count": paper.ReferencesCount,
			"license":          paper.License,
			"funder":           paper.Funder,
		},
	}

	return unified, nil
}

// searchAllSourcesForPaper 在所有数据源中搜索论文
func (a *ScholarAggregator) searchAllSourcesForPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	// 尝试使用标识符作为查询词搜索
	params := common.SearchParams{
		Query:  identifier,
		Limit:  5, // 只要少量结果
		Offset: 0,
	}

	result, err := a.SearchPapers(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(result.Papers) > 0 {
		return &result.Papers[0], nil
	}

	return nil, fmt.Errorf("未在任何数据源中找到论文")
}
