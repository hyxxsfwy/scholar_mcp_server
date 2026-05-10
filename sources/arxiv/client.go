package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func doWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = client.Do(req.Clone(ctx))
		if err != nil || resp.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(1<<uint(i)) * time.Second):
		}
	}
	return resp, err
}

const (
	ArxivAPIBaseURL = "http://export.arxiv.org/api/query"
	DefaultPageSize = 10
	MaxPageSize     = 2000
)

// ArxivClient arXiv API客户端
type ArxivClient struct {
	client  *http.Client
	baseURL string
}

// NewArxivClient 创建新的arXiv客户端
func NewArxivClient() *ArxivClient {
	return &ArxivClient{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: ArxivAPIBaseURL,
	}
}

// SearchPapers 搜索arXiv论文
func (c *ArxivClient) SearchPapers(ctx context.Context, query string, start, maxResults int) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if maxResults <= 0 {
		maxResults = DefaultPageSize
	}
	if maxResults > MaxPageSize {
		maxResults = MaxPageSize
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("search_query", query)
	params.Set("start", strconv.Itoa(start))
	params.Set("max_results", strconv.Itoa(maxResults))
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "arXiv-MCP-Client/1.0")

	resp, err := doWithRetry(ctx, c.client, req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析XML响应失败: %w", err)
	}

	result := &SearchResult{
		Query:      query,
		Start:      start,
		MaxResults: maxResults,
		TotalCount: feed.TotalResults,
		Papers:     make([]Paper, len(feed.Entries)),
	}

	for i, entry := range feed.Entries {
		result.Papers[i] = convertEntryToPaper(entry)
	}

	return result, nil
}

// GetPaper 根据arXiv ID获取特定论文
func (c *ArxivClient) GetPaper(ctx context.Context, arxivID string) (*Paper, error) {
	if arxivID == "" {
		return nil, fmt.Errorf("arXiv ID不能为空")
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("id_list", arxivID)

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "arXiv-MCP-Client/1.0")

	resp, err := doWithRetry(ctx, c.client, req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析XML响应失败: %w", err)
	}

	if len(feed.Entries) == 0 {
		return nil, fmt.Errorf("未找到ID为 %s 的论文", arxivID)
	}

	paper := convertEntryToPaper(feed.Entries[0])
	return &paper, nil
}

// convertEntryToPaper 将arXiv API的entry转换为Paper结构
func convertEntryToPaper(entry Entry) Paper {
	// 提取作者信息
	var authors []string
	for _, author := range entry.Authors {
		authors = append(authors, author.Name)
	}

	// 提取分类信息
	var categories []string
	for _, category := range entry.Categories {
		categories = append(categories, category.Term)
	}

	// 解析发布时间
	publishedDate := ""
	if entry.Published != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", entry.Published); err == nil {
			publishedDate = t.Format("2006-01-02")
		}
	}

	// 解析更新时间
	updatedDate := ""
	if entry.Updated != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", entry.Updated); err == nil {
			updatedDate = t.Format("2006-01-02")
		}
	}

	// 提取PDF链接
	pdfURL := ""
	for _, link := range entry.Links {
		if link.Type == "application/pdf" {
			pdfURL = link.Href
			break
		}
	}

	// 提取arXiv ID
	arxivID := ""
	if entry.ID != "" {
		parts := strings.Split(entry.ID, "/")
		if len(parts) > 0 {
			arxivID = parts[len(parts)-1]
		}
	}

	return Paper{
		ID:         arxivID,
		Title:      strings.TrimSpace(entry.Title),
		Authors:    authors,
		Abstract:   strings.TrimSpace(entry.Summary),
		Categories: categories,
		Published:  publishedDate,
		Updated:    updatedDate,
		URL:        entry.ID,
		PDFURL:     pdfURL,
		DOI:        entry.DOI,
		JournalRef: entry.JournalRef,
		Comment:    entry.Comment,
	}
}
