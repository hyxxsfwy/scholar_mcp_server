package scihub

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

const (
	// 注意：这些是示例镜像地址，实际使用时需要获取最新的可用镜像
	DefaultSciHubMirror = "https://sci-hub.jp"
	DefaultMaxResults   = 10
	MaxMaxResults       = 50
	headerUserAgent     = "User-Agent"
)

// SciHubClient Sci-Hub客户端
type SciHubClient struct {
	client        *http.Client
	mirrors       []string
	currentMirror string
	userAgent     string
}

// NewSciHubClient 创建新的Sci-Hub客户端
func NewSciHubClient() *SciHubClient {
	return &SciHubClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		mirrors: []string{
			"https://sci-hub.jp",
			"https://sci-hub.st",
			"https://sci-hub.ru",
		},
		currentMirror: DefaultSciHubMirror,
		userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
}

// NewSciHubClientWithMirrors 创建带自定义镜像的客户端
func NewSciHubClientWithMirrors(mirrors []string) *SciHubClient {
	client := NewSciHubClient()
	if len(mirrors) > 0 {
		client.mirrors = mirrors
		client.currentMirror = mirrors[0]
	}
	return client
}

// SearchPapers 搜索Sci-Hub论文
// 注意：Sci-Hub主要通过DOI等标识符进行查找，不支持传统的关键词搜索
func (c *SciHubClient) SearchPapers(ctx context.Context, query string, maxResults int) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	if maxResults <= 0 {
		maxResults = DefaultMaxResults
	}
	if maxResults > MaxMaxResults {
		maxResults = MaxMaxResults
	}

	// Sci-Hub只支持通过DOI查找，不支持关键词搜索
	if !isDOI(query) {
		return nil, fmt.Errorf("Sci-Hub仅支持DOI查找，不支持关键词搜索")
	}

	papers := []Paper{}
	if paper, err := c.GetPaper(ctx, query); err == nil {
		papers = append(papers, *paper)
	}

	return &SearchResult{
		Query:  query,
		Papers: papers,
		Total:  len(papers),
		Source: "Sci-Hub",
	}, nil
}

// GetPaper 根据DOI等标识符获取论文PDF
func (c *SciHubClient) GetPaper(ctx context.Context, identifier string) (*Paper, error) {
	if identifier == "" {
		return nil, fmt.Errorf("论文标识符不能为空")
	}

	// 尝试从当前镜像获取论文
	for _, mirror := range c.mirrors {
		paper, err := c.getPaperFromMirror(ctx, mirror, identifier)
		if err == nil && paper != nil {
			return paper, nil
		}
	}

	return nil, fmt.Errorf("未能从任何镜像获取标识符为 %s 的论文", identifier)
}

// getPaperFromMirror 从指定镜像获取论文
func (c *SciHubClient) getPaperFromMirror(ctx context.Context, mirror, identifier string) (*Paper, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s/%s", mirror, url.PathEscape(identifier))

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析HTML页面以提取PDF链接和论文信息
	paper, err := c.parseHTMLResponse(string(body), mirror, identifier)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return paper, nil
}

// parseHTMLResponse 解析HTML响应以提取论文信息
func (c *SciHubClient) parseHTMLResponse(html, mirror, identifier string) (*Paper, error) {
	// 提取PDF链接
	pdfURL := extractPDFURL(html, mirror)
	if pdfURL == "" {
		return nil, fmt.Errorf("未找到PDF链接")
	}

	// 提取论文标题
	title := extractTitle(html)

	// 提取作者信息
	authors := extractAuthors(html)

	// 提取年份
	year := extractYear(html)

	// 提取期刊信息
	journal := extractJournal(html)

	// 检查PDF是否可用
	available := pdfURL != ""

	paper := &Paper{
		DOI:       identifier,
		Title:     title,
		Authors:   authors,
		Year:      year,
		Journal:   journal,
		PDFURL:    pdfURL,
		Available: available,
		SciHubURL: fmt.Sprintf("%s/%s", mirror, identifier),
	}

	return paper, nil
}

// GetPDFURL 获取PDF下载链接
func (c *SciHubClient) GetPDFURL(ctx context.Context, identifier string) (*PDFResponse, error) {
	paper, err := c.GetPaper(ctx, identifier)
	if err != nil {
		return &PDFResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	return &PDFResponse{
		Success:   paper.Available,
		PDFURL:    paper.PDFURL,
		SciHubURL: paper.SciHubURL,
		Message:   "PDF获取成功",
	}, nil
}

// CheckAvailability 检查论文是否可用
func (c *SciHubClient) CheckAvailability(ctx context.Context, doi string) (*AvailabilityCheck, error) {
	paper, err := c.GetPaper(ctx, doi)

	result := &AvailabilityCheck{
		DOI:       doi,
		Available: false,
		CheckTime: time.Now().Format(time.RFC3339),
	}

	if err == nil && paper != nil && paper.Available {
		result.Available = true
		result.URL = paper.PDFURL
	}

	return result, nil
}

// BulkCheckAvailability 批量检查论文可用性
func (c *SciHubClient) BulkCheckAvailability(ctx context.Context, dois []string) (*BulkCheckResponse, error) {
	if len(dois) == 0 {
		return nil, fmt.Errorf("DOI列表不能为空")
	}

	var results []AvailabilityCheck
	successCount := 0

	for _, doi := range dois {
		check, _ := c.CheckAvailability(ctx, doi)
		if check != nil {
			results = append(results, *check)
			if check.Available {
				successCount++
			}
		}
	}

	return &BulkCheckResponse{
		Results: results,
		Total:   len(results),
		Success: successCount,
	}, nil
}

// GetMirrorStatus 获取镜像状态
func (c *SciHubClient) GetMirrorStatus(ctx context.Context) (*MirrorStatus, error) {
	var mirrors []Mirror
	activeCount := 0

	for _, mirrorURL := range c.mirrors {
		status := "unknown"

		// 简单的连通性检查
		if err := c.checkMirrorConnectivity(ctx, mirrorURL); err == nil {
			status = "active"
			activeCount++
		} else {
			status = "inactive"
		}

		mirrors = append(mirrors, Mirror{
			URL:       mirrorURL,
			Status:    status,
			Location:  "unknown", // 在实际实现中可以添加地理位置检测
			LastCheck: time.Now().Format(time.RFC3339),
		})
	}

	return &MirrorStatus{
		Mirrors:     mirrors,
		ActiveCount: activeCount,
		LastUpdate:  time.Now().Format(time.RFC3339),
	}, nil
}

// checkMirrorConnectivity 检查镜像连通性
func (c *SciHubClient) checkMirrorConnectivity(ctx context.Context, mirrorURL string) error {
	req, err := http.NewRequestWithContext(ctx, "HEAD", mirrorURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set(headerUserAgent, c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("镜像不可用，状态码: %d", resp.StatusCode)
	}

	return nil
}

// DownloadAndConvertPDF 下载 PDF 并转换为 Markdown 文本
func (c *SciHubClient) DownloadAndConvertPDF(ctx context.Context, doi string) (string, error) {
	paper, err := c.GetPaper(ctx, doi)
	if err != nil {
		return "", fmt.Errorf("获取论文失败: %w", err)
	}
	if paper.PDFURL == "" {
		return "", fmt.Errorf("未找到 PDF 链接")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", paper.PDFURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载 PDF 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 PDF 失败，状态码: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 PDF 数据失败: %w", err)
	}

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("解析 PDF 失败: %w", err)
	}

	var sb strings.Builder
	if paper.Title != "" {
		sb.WriteString("# ")
		sb.WriteString(paper.Title)
		sb.WriteString("\n\n")
	}

	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
	}

	return sb.String(), nil
}

// 辅助函数：检查是否为DOI
func isDOI(s string) bool {
	// DOI格式：10.xxxx/xxxx
	doiPattern := regexp.MustCompile(`^10\.\d{4,}/[^\s]+$`)
	return doiPattern.MatchString(s)
}

// 辅助函数：从HTML中提取PDF URL
func extractPDFURL(html, mirror string) string {
	// 查找PDF链接的正则表达式
	patterns := []string{
		`(?i)href="([^"]*\.pdf[^"]*)"`,
		`(?i)src="([^"]*\.pdf[^"]*)"`,
		`(?i)location\.href\s*=\s*["']([^"']*\.pdf[^"']*)["']`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			url := matches[1]
			// 如果是相对路径，转换为绝对路径
			if strings.HasPrefix(url, "/") {
				url = mirror + url
			} else if !strings.HasPrefix(url, "http") {
				url = mirror + "/" + url
			}
			return url
		}
	}

	return ""
}

// 辅助函数：从HTML中提取标题
func extractTitle(html string) string {
	patterns := []string{
		`<title>([^<]+)</title>`,
		`<h1[^>]*>([^<]+)</h1>`,
		`(?i)citation_title[^>]*content="([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			title := strings.TrimSpace(matches[1])
			if title != "" && !strings.Contains(title, "Sci-Hub") {
				return title
			}
		}
	}

	return ""
}

// 辅助函数：从HTML中提取作者
func extractAuthors(html string) string {
	patterns := []string{
		`(?i)citation_author[^>]*content="([^"]+)"`,
		`(?i)authors?[^>]*>([^<]+)</`,
	}

	var authors []string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				author := strings.TrimSpace(match[1])
				if author != "" {
					authors = append(authors, author)
				}
			}
		}
	}

	return strings.Join(authors, ", ")
}

// 辅助函数：从HTML中提取年份
func extractYear(html string) string {
	patterns := []string{
		`(?i)citation_date[^>]*content="([^"]+)"`,
		`(?i)citation_year[^>]*content="([^"]+)"`,
		`\b(19|20)\d{2}\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			year := strings.TrimSpace(matches[1])
			if yearInt, err := strconv.Atoi(year); err == nil && yearInt >= 1900 && yearInt <= time.Now().Year() {
				return year
			}
		}
	}

	return ""
}

// 辅助函数：从HTML中提取期刊信息
func extractJournal(html string) string {
	patterns := []string{
		`(?i)citation_journal_title[^>]*content="([^"]+)"`,
		`(?i)journal[^>]*>([^<]+)</`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			journal := strings.TrimSpace(matches[1])
			if journal != "" {
				return journal
			}
		}
	}

	return ""
}
