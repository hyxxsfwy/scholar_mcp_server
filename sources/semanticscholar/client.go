package semanticscholar

import (
	"context"
	"encoding/json"
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
	SemanticScholarAPIBaseURL = "https://api.semanticscholar.org/graph/v1"
	DefaultLimit              = 10
	MaxLimit                  = 100
)

// SemanticScholarClient Semantic Scholar API客户端
type SemanticScholarClient struct {
	client  *http.Client
	baseURL string
	apiKey  string // 可选的API密钥，用于更高的请求限制
}

// NewSemanticScholarClient 创建新的Semantic Scholar客户端
func NewSemanticScholarClient() *SemanticScholarClient {
	return &SemanticScholarClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: SemanticScholarAPIBaseURL,
	}
}

// NewSemanticScholarClientWithAPIKey 创建带API密钥的客户端
func NewSemanticScholarClientWithAPIKey(apiKey string) *SemanticScholarClient {
	client := NewSemanticScholarClient()
	client.apiKey = apiKey
	return client
}

// SearchPapers 搜索Semantic Scholar论文
func (c *SemanticScholarClient) SearchPapers(ctx context.Context, query string, offset, limit int) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("query", query)
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("fields", "paperId,corpusId,url,title,abstract,venue,year,referenceCount,citationCount,influentialCitationCount,isOpenAccess,fieldsOfStudy,authors,externalIds,publicationTypes,publicationDate,journal")

	reqURL := fmt.Sprintf("%s/paper/search?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "SemanticScholar-MCP-Client/1.0")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := doWithRetry(ctx, c.client, req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &SearchResult{
		Query:  query,
		Offset: offset,
		Limit:  limit,
		Total:  apiResponse.Total,
		Papers: apiResponse.Data,
	}, nil
}

// GetPaper 根据Paper ID获取特定论文
func (c *SemanticScholarClient) GetPaper(ctx context.Context, paperID string) (*Paper, error) {
	if paperID == "" {
		return nil, fmt.Errorf("Paper ID不能为空")
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("fields", "paperId,corpusId,url,title,abstract,venue,year,referenceCount,citationCount,influentialCitationCount,isOpenAccess,fieldsOfStudy,authors,externalIds,publicationTypes,publicationDate,journal")

	reqURL := fmt.Sprintf("%s/paper/%s?%s", c.baseURL, url.PathEscape(paperID), params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "SemanticScholar-MCP-Client/1.0")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := doWithRetry(ctx, c.client, req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("未找到ID为 %s 的论文", paperID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var paper Paper
	if err := json.Unmarshal(body, &paper); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &paper, nil
}

// SearchPapersWithParams 使用详细参数搜索论文
func (c *SemanticScholarClient) SearchPapersWithParams(ctx context.Context, params *SemanticScholarSearchParam) (*SearchResult, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if params.Limit <= 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// 构建查询URL
	urlParams := url.Values{}
	urlParams.Set("query", params.Query)
	urlParams.Set("offset", strconv.Itoa(params.Offset))
	urlParams.Set("limit", strconv.Itoa(params.Limit))
	urlParams.Set("fields", "paperId,corpusId,url,title,abstract,venue,year,referenceCount,citationCount,influentialCitationCount,isOpenAccess,fieldsOfStudy,authors,externalIds,publicationTypes,publicationDate,journal")

	// 添加可选过滤参数
	if params.Year != "" {
		urlParams.Set("year", params.Year)
	}
	if len(params.Venue) > 0 {
		urlParams.Set("venue", strings.Join(params.Venue, ","))
	}
	if len(params.FieldsOfStudy) > 0 {
		urlParams.Set("fieldsOfStudy", strings.Join(params.FieldsOfStudy, ","))
	}
	if len(params.PublicationTypes) > 0 {
		urlParams.Set("publicationTypes", strings.Join(params.PublicationTypes, ","))
	}
	if params.MinCitationCount > 0 {
		urlParams.Set("minCitationCount", strconv.Itoa(params.MinCitationCount))
	}
	if params.Sort != "" {
		urlParams.Set("sort", params.Sort)
	}

	reqURL := fmt.Sprintf("%s/paper/search?%s", c.baseURL, urlParams.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "SemanticScholar-MCP-Client/1.0")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := doWithRetry(ctx, c.client, req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &SearchResult{
		Query:  params.Query,
		Offset: params.Offset,
		Limit:  params.Limit,
		Total:  apiResponse.Total,
		Papers: apiResponse.Data,
	}, nil
}
