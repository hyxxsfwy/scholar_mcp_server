package crossref

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
	CrossrefAPIBaseURL = "https://api.crossref.org/works"
	DefaultRows        = 20
	MaxRows            = 1000
)

// CrossrefClient Crossref API客户端
type CrossrefClient struct {
	client  *http.Client
	baseURL string
	email   string // 礼貌性邮箱，用于API请求识别
}

// NewCrossrefClient 创建新的Crossref客户端
func NewCrossrefClient() *CrossrefClient {
	return &CrossrefClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: CrossrefAPIBaseURL,
	}
}

// NewCrossrefClientWithEmail 创建带邮箱的客户端
func NewCrossrefClientWithEmail(email string) *CrossrefClient {
	client := NewCrossrefClient()
	client.email = email
	return client
}

// SearchPapers 搜索Crossref论文
func (c *CrossrefClient) SearchPapers(ctx context.Context, query string, offset, rows int) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if rows <= 0 {
		rows = DefaultRows
	}
	if rows > MaxRows {
		rows = MaxRows
	}
	if offset < 0 {
		offset = 0
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("query", query)
	params.Set("offset", strconv.Itoa(offset))
	params.Set("rows", strconv.Itoa(rows))
	params.Set("sort", "relevance")

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Crossref-MCP-Client/1.0")
	if c.email != "" {
		req.Header.Set("User-Agent", fmt.Sprintf("Crossref-MCP-Client/1.0 (mailto:%s)", c.email))
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

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

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:        query,
		Offset:       offset,
		Rows:         rows,
		TotalResults: apiResponse.Message.TotalResults,
		ItemsPerPage: apiResponse.Message.ItemsPerPage,
		Papers:       apiResponse.Message.Items,
	}

	return result, nil
}

// GetPaper 根据DOI获取特定论文
func (c *CrossrefClient) GetPaper(ctx context.Context, doi string) (*Paper, error) {
	if doi == "" {
		return nil, fmt.Errorf("DOI不能为空")
	}

	// 构建查询URL
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, url.PathEscape(doi))

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Crossref-MCP-Client/1.0")
	if c.email != "" {
		req.Header.Set("User-Agent", fmt.Sprintf("Crossref-MCP-Client/1.0 (mailto:%s)", c.email))
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("未找到DOI为 %s 的论文", doi)
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

	// Crossref单个论文API返回的结构稍有不同
	type SinglePaperResponse struct {
		Status         string `json:"status"`
		MessageType    string `json:"message-type"`
		MessageVersion string `json:"message-version"`
		Message        Paper  `json:"message"`
	}

	var singleResponse SinglePaperResponse
	if err := json.Unmarshal(body, &singleResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &singleResponse.Message, nil
}

// SearchPapersWithParams 使用详细参数搜索论文
func (c *CrossrefClient) SearchPapersWithParams(ctx context.Context, params *CrossrefSearchParam) (*SearchResult, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if params.Rows <= 0 {
		params.Rows = DefaultRows
	}
	if params.Rows > MaxRows {
		params.Rows = MaxRows
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// 构建查询URL
	urlParams := url.Values{}
	urlParams.Set("query", params.Query)
	urlParams.Set("offset", strconv.Itoa(params.Offset))
	urlParams.Set("rows", strconv.Itoa(params.Rows))

	// 添加可选过滤参数
	if params.Author != "" {
		urlParams.Set("query.author", params.Author)
	}
	if params.Title != "" {
		urlParams.Set("query.title", params.Title)
	}
	if params.Container != "" {
		urlParams.Set("query.container-title", params.Container)
	}
	if params.Publisher != "" {
		urlParams.Set("query.publisher-name", params.Publisher)
	}
	if params.Type != "" {
		urlParams.Set("filter", fmt.Sprintf("type:%s", params.Type))
	}
	if params.FromDate != "" {
		urlParams.Set("filter", fmt.Sprintf("from-pub-date:%s", params.FromDate))
	}
	if params.UntilDate != "" {
		urlParams.Set("filter", fmt.Sprintf("until-pub-date:%s", params.UntilDate))
	}
	if params.HasOrcid {
		urlParams.Set("filter", "has-orcid:true")
	}
	if params.HasFunder {
		urlParams.Set("filter", "has-funder:true")
	}
	if params.HasLicense {
		urlParams.Set("filter", "has-license:true")
	}
	if params.Sort != "" {
		sortParam := params.Sort
		if params.Order != "" {
			sortParam = fmt.Sprintf("%s:%s", params.Sort, params.Order)
		}
		urlParams.Set("sort", sortParam)
	}

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, urlParams.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Crossref-MCP-Client/1.0")
	if c.email != "" {
		req.Header.Set("User-Agent", fmt.Sprintf("Crossref-MCP-Client/1.0 (mailto:%s)", c.email))
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

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

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:        params.Query,
		Offset:       params.Offset,
		Rows:         params.Rows,
		TotalResults: apiResponse.Message.TotalResults,
		ItemsPerPage: apiResponse.Message.ItemsPerPage,
		Papers:       apiResponse.Message.Items,
	}

	return result, nil
}
