package scopus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	ScopusAPIBaseURL = "https://api.elsevier.com/content"
	DefaultCount     = 25
	MaxCount         = 200
)

// ScopusClient Scopus API客户端
type ScopusClient struct {
	client  *http.Client
	baseURL string
	apiKey  string // 必需的API密钥
}

// NewScopusClient 创建新的Scopus客户端
// 注意：Scopus API需要API密钥才能使用
func NewScopusClient(apiKey string) *ScopusClient {
	return &ScopusClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: ScopusAPIBaseURL,
		apiKey:  apiKey,
	}
}

// SearchPapers 搜索Scopus论文
func (c *ScopusClient) SearchPapers(ctx context.Context, query string, start, count int) (*SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Scopus API密钥是必需的")
	}

	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if count <= 0 {
		count = DefaultCount
	}
	if count > MaxCount {
		count = MaxCount
	}
	if start < 0 {
		start = 0
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.Itoa(start))
	params.Set("count", strconv.Itoa(count))
	params.Set("view", "STANDARD")
	params.Set("sort", "relevancy")
	params.Set("httpAccept", "application/json")

	reqURL := fmt.Sprintf("%s/search/scopus?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-ELS-APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Scopus-MCP-Client/1.0")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("API密钥无效或权限不足")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 转换总数量
	totalCount := 0
	if totalStr := apiResponse.SearchResults.TotalResults; totalStr != "" {
		if count, err := strconv.Atoi(totalStr); err == nil {
			totalCount = count
		}
	}

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:      query,
		Start:      start,
		Count:      count,
		TotalCount: totalCount,
		Papers:     apiResponse.SearchResults.Entry,
	}

	return result, nil
}

// GetPaper 根据EID或DOI获取特定论文
func (c *ScopusClient) GetPaper(ctx context.Context, identifier string) (*Paper, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Scopus API密钥是必需的")
	}

	if identifier == "" {
		return nil, fmt.Errorf("论文标识符不能为空")
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("view", "FULL")
	params.Set("httpAccept", "application/json")

	// 判断是EID还是DOI
	var reqURL string
	if isEID(identifier) {
		reqURL = fmt.Sprintf("%s/abstract/eid/%s?%s", c.baseURL, url.PathEscape(identifier), params.Encode())
	} else {
		reqURL = fmt.Sprintf("%s/abstract/doi/%s?%s", c.baseURL, url.PathEscape(identifier), params.Encode())
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-ELS-APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Scopus-MCP-Client/1.0")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("未找到标识符为 %s 的论文", identifier)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("API密钥无效或权限不足")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var singleResponse SinglePaperResponse
	if err := json.Unmarshal(body, &singleResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 转换为Paper结构
	paper := convertAbstractRetrievalToPaper(singleResponse.AbstractRetrievalResponse)
	return &paper, nil
}

// SearchPapersWithParams 使用详细参数搜索论文
func (c *ScopusClient) SearchPapersWithParams(ctx context.Context, params *ScopusSearchParam) (*SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Scopus API密钥是必需的")
	}

	if params.Query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if params.Count <= 0 {
		params.Count = DefaultCount
	}
	if params.Count > MaxCount {
		params.Count = MaxCount
	}
	if params.Start < 0 {
		params.Start = 0
	}

	// 构建查询字符串
	query := params.Query

	// 添加可选过滤条件
	if params.Author != "" {
		query += fmt.Sprintf(" AND AUTH(%s)", params.Author)
	}
	if params.Title != "" {
		query += fmt.Sprintf(" AND TITLE(%s)", params.Title)
	}
	if params.Journal != "" {
		query += fmt.Sprintf(" AND SRCTITLE(%s)", params.Journal)
	}
	if params.Year != "" {
		query += fmt.Sprintf(" AND PUBYEAR IS %s", params.Year)
	}
	if params.SubjectArea != "" {
		query += fmt.Sprintf(" AND SUBJAREA(%s)", params.SubjectArea)
	}
	if params.Affiliation != "" {
		query += fmt.Sprintf(" AND AFFIL(%s)", params.Affiliation)
	}
	if params.Language != "" {
		query += fmt.Sprintf(" AND LANGUAGE(%s)", params.Language)
	}
	if params.DocType != "" {
		query += fmt.Sprintf(" AND DOCTYPE(%s)", params.DocType)
	}

	// 构建查询URL
	urlParams := url.Values{}
	urlParams.Set("query", query)
	urlParams.Set("start", strconv.Itoa(params.Start))
	urlParams.Set("count", strconv.Itoa(params.Count))
	urlParams.Set("view", "STANDARD")
	if params.View != "" {
		urlParams.Set("view", params.View)
	}
	if params.Sort != "" {
		urlParams.Set("sort", params.Sort)
	} else {
		urlParams.Set("sort", "relevancy")
	}
	urlParams.Set("httpAccept", "application/json")

	reqURL := fmt.Sprintf("%s/search/scopus?%s", c.baseURL, urlParams.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-ELS-APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Scopus-MCP-Client/1.0")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("API密钥无效或权限不足")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 转换总数量
	totalCount := 0
	if totalStr := apiResponse.SearchResults.TotalResults; totalStr != "" {
		if count, err := strconv.Atoi(totalStr); err == nil {
			totalCount = count
		}
	}

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:      params.Query,
		Start:      params.Start,
		Count:      params.Count,
		TotalCount: totalCount,
		Papers:     apiResponse.SearchResults.Entry,
	}

	return result, nil
}

// isEID 判断是否为Scopus EID
func isEID(identifier string) bool {
	// Scopus EID通常以"2-s2.0-"开头
	return len(identifier) > 7 && identifier[:7] == "2-s2.0-"
}

// convertAbstractRetrievalToPaper 将Abstract Retrieval响应转换为Paper结构
func convertAbstractRetrievalToPaper(data AbstractRetrievalData) Paper {
	// 提取被引用次数
	citedByCount := 0
	if data.CoreData.CitedByCount != "" {
		if count, err := strconv.Atoi(data.CoreData.CitedByCount); err == nil {
			citedByCount = count
		}
	}

	// 提取摘要
	abstract := ""
	if data.CoreData.Description != "" {
		abstract = data.CoreData.Description
	}

	// 构建Scopus URL
	scopusURL := ""
	for _, link := range data.CoreData.Link {
		if link.Ref == "scopus" {
			scopusURL = link.Href
			break
		}
	}

	return Paper{
		EID:              data.CoreData.EID,
		DOI:              data.CoreData.DOI,
		URL:              scopusURL,
		Title:            data.CoreData.Title,
		Abstract:         abstract,
		PublicationName:  data.CoreData.PublicationName,
		CoverDate:        data.CoreData.CoverDate,
		Publisher:        data.CoreData.Publisher,
		AggregationType:  data.CoreData.AggregationType,
		SubtypeDescription: data.CoreData.SubtypeDescription,
		CitedByCount:     citedByCount,
		Author:           data.Authors.Author,
		Affiliation:      data.Affiliation,
		SubjectAreas:     data.SubjectAreas.SubjectArea,
	}
}
