package adsabs

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

const (
	AdsabsAPIBaseURL = "https://api.adsabs.harvard.edu/v1"
	DefaultRows      = 10
	MaxRows          = 2000
)

// AdsabsClient ADSABS API客户端
type AdsabsClient struct {
	client  *http.Client
	baseURL string
	apiKey  string // 必需的API密钥
}

// NewAdsabsClient 创建新的ADSABS客户端
// 注意：ADSABS API需要API密钥才能使用
func NewAdsabsClient(apiKey string) *AdsabsClient {
	return &AdsabsClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: AdsabsAPIBaseURL,
		apiKey:  apiKey,
	}
}

// SearchPapers 搜索ADSABS论文
func (c *AdsabsClient) SearchPapers(ctx context.Context, query string, start, rows int) (*SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ADSABS API密钥是必需的")
	}

	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if rows <= 0 {
		rows = DefaultRows
	}
	if rows > MaxRows {
		rows = MaxRows
	}
	if start < 0 {
		start = 0
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("q", query)
	params.Set("start", strconv.Itoa(start))
	params.Set("rows", strconv.Itoa(rows))
	params.Set("sort", "date desc")
	// 设置返回的字段
	fields := []string{
		"bibcode", "title", "author", "first_author", "pub", "pubdate",
		"volume", "issue", "page", "doi", "arxiv_id", "abstract",
		"citation_count", "read_count", "database", "doctype",
		"property", "keyword", "bibstem", "aff", "identifier",
	}
	params.Set("fl", strings.Join(fields, ","))

	reqURL := fmt.Sprintf("%s/search/query?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ADSABS-MCP-Client/1.0")

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

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:          query,
		Start:          start,
		Rows:           rows,
		NumFound:       apiResponse.Response.NumFound,
		Papers:         apiResponse.Response.Docs,
		ResponseHeader: apiResponse.ResponseHeader,
	}

	return result, nil
}

// GetPaper 根据Bibcode获取特定论文
func (c *AdsabsClient) GetPaper(ctx context.Context, bibcode string) (*Paper, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ADSABS API密钥是必需的")
	}

	if bibcode == "" {
		return nil, fmt.Errorf("Bibcode不能为空")
	}

	// 构建查询URL
	params := url.Values{}
	params.Set("q", fmt.Sprintf("bibcode:%s", bibcode))
	params.Set("rows", "1")
	// 设置返回的字段
	fields := []string{
		"bibcode", "title", "author", "first_author", "pub", "pubdate",
		"volume", "issue", "page", "doi", "arxiv_id", "abstract",
		"citation_count", "read_count", "database", "doctype",
		"property", "keyword", "bibstem", "aff", "identifier",
		"facility", "instrument", "object", "orcid", "grant",
	}
	params.Set("fl", strings.Join(fields, ","))

	reqURL := fmt.Sprintf("%s/search/query?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ADSABS-MCP-Client/1.0")

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

	if len(apiResponse.Response.Docs) == 0 {
		return nil, fmt.Errorf("未找到Bibcode为 %s 的论文", bibcode)
	}

	return &apiResponse.Response.Docs[0], nil
}

// SearchPapersWithParams 使用详细参数搜索论文
func (c *AdsabsClient) SearchPapersWithParams(ctx context.Context, params *AdsabsSearchParam) (*SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ADSABS API密钥是必需的")
	}

	if params.Query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if params.Rows <= 0 {
		params.Rows = DefaultRows
	}
	if params.Rows > MaxRows {
		params.Rows = MaxRows
	}
	if params.Start < 0 {
		params.Start = 0
	}

	// 构建复杂查询字符串
	query := params.Query

	// 添加可选过滤条件
	if params.Author != "" {
		query += fmt.Sprintf(" author:\"%s\"", params.Author)
	}
	if params.Title != "" {
		query += fmt.Sprintf(" title:\"%s\"", params.Title)
	}
	if params.Abstract != "" {
		query += fmt.Sprintf(" abstract:\"%s\"", params.Abstract)
	}
	if params.Year != "" {
		query += fmt.Sprintf(" year:%s", params.Year)
	}
	if params.Pub != "" {
		query += fmt.Sprintf(" pub:\"%s\"", params.Pub)
	}
	if params.Object != "" {
		query += fmt.Sprintf(" object:\"%s\"", params.Object)
	}
	if params.Database != "" {
		query += fmt.Sprintf(" database:%s", params.Database)
	}
	if params.Property != "" {
		query += fmt.Sprintf(" property:%s", params.Property)
	}
	if params.DocType != "" {
		query += fmt.Sprintf(" doctype:%s", params.DocType)
	}
	if params.Bibstem != "" {
		query += fmt.Sprintf(" bibstem:%s", params.Bibstem)
	}

	// 构建查询URL
	urlParams := url.Values{}
	urlParams.Set("q", query)
	urlParams.Set("start", strconv.Itoa(params.Start))
	urlParams.Set("rows", strconv.Itoa(params.Rows))
	
	if params.Sort != "" {
		urlParams.Set("sort", params.Sort)
	} else {
		urlParams.Set("sort", "date desc")
	}

	// 设置返回的字段
	fields := []string{
		"bibcode", "title", "author", "first_author", "pub", "pubdate",
		"volume", "issue", "page", "doi", "arxiv_id", "abstract",
		"citation_count", "read_count", "database", "doctype",
		"property", "keyword", "bibstem", "aff", "identifier",
	}
	if len(params.Fields) > 0 {
		fields = params.Fields
	}
	urlParams.Set("fl", strings.Join(fields, ","))

	reqURL := fmt.Sprintf("%s/search/query?%s", c.baseURL, urlParams.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ADSABS-MCP-Client/1.0")

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

	// 转换为我们的搜索结果结构
	result := &SearchResult{
		Query:          params.Query,
		Start:          params.Start,
		Rows:           params.Rows,
		NumFound:       apiResponse.Response.NumFound,
		Papers:         apiResponse.Response.Docs,
		ResponseHeader: apiResponse.ResponseHeader,
	}

	return result, nil
}

// ExportCitation 导出引用格式
func (c *AdsabsClient) ExportCitation(ctx context.Context, bibcodes []string, format ExportFormat) (*ExportResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ADSABS API密钥是必需的")
	}

	if len(bibcodes) == 0 {
		return nil, fmt.Errorf("Bibcode列表不能为空")
	}

	// 构建请求体
	exportReq := ExportRequest{
		Bibcode: bibcodes,
		Format:  format,
		Sort:    "date desc",
	}

	reqBody, err := json.Marshal(exportReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建请求
	reqURL := fmt.Sprintf("%s/export/%s", c.baseURL, string(format))
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ADSABS-MCP-Client/1.0")

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
	var exportResponse ExportResponse
	if err := json.Unmarshal(body, &exportResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &exportResponse, nil
}

// GetMetrics 获取论文指标
func (c *AdsabsClient) GetMetrics(ctx context.Context, bibcodes []string) (*MetricsResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ADSABS API密钥是必需的")
	}

	if len(bibcodes) == 0 {
		return nil, fmt.Errorf("Bibcode列表不能为空")
	}

	// 构建请求体
	reqData := map[string]interface{}{
		"bibcodes": bibcodes,
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建请求
	reqURL := fmt.Sprintf("%s/metrics", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ADSABS-MCP-Client/1.0")

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
	var metricsResponse MetricsResponse
	if err := json.Unmarshal(body, &metricsResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &metricsResponse, nil
}
