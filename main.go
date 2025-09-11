package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"log"
	"mcp/arxiv"
	"mcp/semanticscholar"
	"mcp/crossref"
	"mcp/scopus"
	"mcp/adsabs"
	"mcp/scihub"
	"net/http"
	"os"
	"strings"
)

// arXiv搜索功能
func searchArxivPapers(ctx context.Context, req *mcp.CallToolRequest, params *arxiv.ArxivSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索arXiv论文 ==========")
	log.Printf("[DEBUG] 接收到的原始参数: %+v", params)

	// 设置默认值
	if params.Start < 0 {
		params.Start = 0
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 10
	}
	if params.MaxResults > 100 {
		params.MaxResults = 100
	}

	log.Printf("[DEBUG] 参数验证后的最终值: Query='%s', Start=%d, MaxResults=%d", params.Query, params.Start, params.MaxResults)

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		log.Printf("[ERROR] 搜索关键词为空，终止请求")
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索arXiv: 关键词='%s', Start=%d, MaxResults=%d", params.Query, params.Start, params.MaxResults)

	// 创建arXiv客户端并搜索
	client := arxiv.NewArxivClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Start, params.MaxResults)
	if err != nil {
		log.Printf("[ERROR] arXiv搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	log.Printf("[DEBUG] 开始格式化搜索结果文本输出...")
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 arXiv论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Start+1, params.Start+len(result.Papers), result.TotalCount))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		resultBuilder.WriteString(fmt.Sprintf("   ID: %s\n", paper.ID))
		resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(paper.Authors, ", ")))
		if paper.Published != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发布日期: %s\n", paper.Published))
		}
		if len(paper.Categories) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   分类: %s\n", strings.Join(paper.Categories, ", ")))
		}
		if paper.Abstract != "" {
			abstract := paper.Abstract
			if len(abstract) > 300 {
				abstract = abstract[:300] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", abstract))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		if paper.PDFURL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   PDF: %s\n", paper.PDFURL))
		}
		resultBuilder.WriteString("\n")
	}
	log.Printf("[DEBUG] 文本格式化完成，长度: %d 字符", resultBuilder.Len())

	// 将结果转换为JSON用于结构化数据
	log.Printf("[DEBUG] 开始JSON序列化搜索结果...")
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	} else {
		log.Printf("[DEBUG] JSON序列化成功，长度: %d 字节", len(jsonResult))
	}

	log.Printf("[INFO] ========== arXiv搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// 获取arXiv论文详情功能
func getArxivPaper(ctx context.Context, req *mcp.CallToolRequest, params *arxiv.ArxivGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取arXiv论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

	// 验证ID参数
	if strings.TrimSpace(params.ID) == "" {
		log.Printf("[ERROR] arXiv ID为空，终止请求")
		return nil, nil, fmt.Errorf("arXiv ID不能为空")
	}

	log.Printf("[INFO] 开始获取arXiv论文详情: ID='%s'", params.ID)

	// 创建arXiv客户端并获取论文
	client := arxiv.NewArxivClient()
	paper, err := client.GetPaper(ctx, params.ID)
	if err != nil {
		log.Printf("[ERROR] 获取arXiv论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	log.Printf("[DEBUG] 开始格式化论文详情输出...")
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 arXiv论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("🆔 ID: %s\n\n", paper.ID))
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))

	if len(paper.Authors) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(paper.Authors, ", ")))
	}

	if paper.Published != "" {
		resultBuilder.WriteString(fmt.Sprintf("📅 发布日期: %s\n", paper.Published))
	}

	if paper.Updated != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔄 更新日期: %s\n", paper.Updated))
	}

	if len(paper.Categories) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("🏷️ 分类: %s\n\n", strings.Join(paper.Categories, ", ")))
	}

	if paper.Abstract != "" {
		resultBuilder.WriteString(fmt.Sprintf("📋 摘要:\n%s\n\n", paper.Abstract))
	}

	if paper.JournalRef != "" {
		resultBuilder.WriteString(fmt.Sprintf("📖 期刊引用: %s\n", paper.JournalRef))
	}

	if paper.DOI != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔗 DOI: %s\n", paper.DOI))
	}

	if paper.Comment != "" {
		resultBuilder.WriteString(fmt.Sprintf("💬 注释: %s\n", paper.Comment))
	}

	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("🌐 arXiv链接: %s\n", paper.URL))
	}

	if paper.PDFURL != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 PDF下载: %s\n", paper.PDFURL))
	}

	log.Printf("[DEBUG] 论文详情文本格式化完成，长度: %d 字符", resultBuilder.Len())

	// 将结果转换为JSON
	log.Printf("[DEBUG] 开始JSON序列化论文详情...")
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	} else {
		log.Printf("[DEBUG] JSON序列化成功，长度: %d 字节", len(jsonResult))
	}

	log.Printf("[INFO] ========== arXiv论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// Semantic Scholar搜索功能
func searchSemanticScholarPapers(ctx context.Context, req *mcp.CallToolRequest, params *semanticscholar.SemanticScholarSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索Semantic Scholar论文 ==========")
	log.Printf("[DEBUG] 接收到的原始参数: %+v", params)

	// 设置默认值
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		log.Printf("[ERROR] 搜索关键词为空，终止请求")
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索Semantic Scholar: 关键词='%s', Offset=%d, Limit=%d", params.Query, params.Offset, params.Limit)

	// 创建Semantic Scholar客户端并搜索
	client := semanticscholar.NewSemanticScholarClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	if err != nil {
		log.Printf("[ERROR] Semantic Scholar搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 Semantic Scholar论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(result.Papers), result.Total))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		resultBuilder.WriteString(fmt.Sprintf("   Paper ID: %s\n", paper.PaperID))
		
		if len(paper.Authors) > 0 {
			var authorNames []string
			for _, author := range paper.Authors {
				authorNames = append(authorNames, author.Name)
			}
			resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(authorNames, ", ")))
		}
		
		if paper.Year > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   年份: %d\n", paper.Year))
		}
		if paper.Venue != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发表场所: %s\n", paper.Venue))
		}
		if paper.CitationCount > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   引用数: %d\n", paper.CitationCount))
		}
		if paper.Abstract != "" {
			abstract := paper.Abstract
			if len(abstract) > 300 {
				abstract = abstract[:300] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", abstract))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		resultBuilder.WriteString("\n")
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Semantic Scholar搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// 获取Semantic Scholar论文详情功能
func getSemanticScholarPaper(ctx context.Context, req *mcp.CallToolRequest, params *semanticscholar.SemanticScholarGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取Semantic Scholar论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

	// 验证ID参数
	if strings.TrimSpace(params.PaperID) == "" {
		log.Printf("[ERROR] Paper ID为空，终止请求")
		return nil, nil, fmt.Errorf("Paper ID不能为空")
	}

	log.Printf("[INFO] 开始获取Semantic Scholar论文详情: ID='%s'", params.PaperID)

	// 创建Semantic Scholar客户端并获取论文
	client := semanticscholar.NewSemanticScholarClient()
	paper, err := client.GetPaper(ctx, params.PaperID)
	if err != nil {
		log.Printf("[ERROR] 获取Semantic Scholar论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 Semantic Scholar论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("🆔 Paper ID: %s\n\n", paper.PaperID))
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))

	if len(paper.Authors) > 0 {
		var authorNames []string
		for _, author := range paper.Authors {
			authorNames = append(authorNames, author.Name)
		}
		resultBuilder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(authorNames, ", ")))
	}

	if paper.Year > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📅 年份: %d\n", paper.Year))
	}
	if paper.Venue != "" {
		resultBuilder.WriteString(fmt.Sprintf("📍 发表场所: %s\n", paper.Venue))
	}
	if paper.CitationCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📊 引用数: %d\n", paper.CitationCount))
	}
	if paper.ReferenceCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📚 参考文献数: %d\n", paper.ReferenceCount))
	}
	if paper.InfluentialCitationCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("⭐ 有影响力引用数: %d\n", paper.InfluentialCitationCount))
	}

	if len(paper.FieldsOfStudy) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("🏷️ 研究领域: %s\n\n", strings.Join(paper.FieldsOfStudy, ", ")))
	}

	if paper.Abstract != "" {
		resultBuilder.WriteString(fmt.Sprintf("📋 摘要:\n%s\n\n", paper.Abstract))
	}

	if paper.ExternalIDs.DOI != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔗 DOI: %s\n", paper.ExternalIDs.DOI))
	}
	if paper.ExternalIDs.ArXiv != "" {
		resultBuilder.WriteString(fmt.Sprintf("📖 arXiv: %s\n", paper.ExternalIDs.ArXiv))
	}
	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("🌐 Semantic Scholar链接: %s\n", paper.URL))
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Semantic Scholar论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// Crossref搜索功能
func searchCrossrefPapers(ctx context.Context, req *mcp.CallToolRequest, params *crossref.CrossrefSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索Crossref论文 ==========")

	// 设置默认值
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Rows <= 0 {
		params.Rows = 20
	}
	if params.Rows > 1000 {
		params.Rows = 1000
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索Crossref: 关键词='%s', Offset=%d, Rows=%d", params.Query, params.Offset, params.Rows)

	// 创建Crossref客户端并搜索
	client := crossref.NewCrossrefClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Rows)
	if err != nil {
		log.Printf("[ERROR] Crossref搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 Crossref论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(result.Papers), result.TotalResults))

	for i, paper := range result.Papers {
		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, title))
		resultBuilder.WriteString(fmt.Sprintf("   DOI: %s\n", paper.DOI))
		
		if len(paper.Author) > 0 {
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
			if len(authorNames) > 0 {
				resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(authorNames, ", ")))
			}
		}
		
		if len(paper.ContainerTitle) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   期刊: %s\n", paper.ContainerTitle[0]))
		}
		if paper.Publisher != "" {
			resultBuilder.WriteString(fmt.Sprintf("   出版商: %s\n", paper.Publisher))
		}
		if paper.IsReferencedByCount > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   被引用数: %d\n", paper.IsReferencedByCount))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		resultBuilder.WriteString("\n")
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Crossref搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// 获取Crossref论文详情功能
func getCrossrefPaper(ctx context.Context, req *mcp.CallToolRequest, params *crossref.CrossrefGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取Crossref论文详情 ==========")

	// 验证DOI参数
	if strings.TrimSpace(params.DOI) == "" {
		return nil, nil, fmt.Errorf("DOI不能为空")
	}

	log.Printf("[INFO] 开始获取Crossref论文详情: DOI='%s'", params.DOI)

	// 创建Crossref客户端并获取论文
	client := crossref.NewCrossrefClient()
	paper, err := client.GetPaper(ctx, params.DOI)
	if err != nil {
		log.Printf("[ERROR] 获取Crossref论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 Crossref论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("🆔 DOI: %s\n\n", paper.DOI))
	
	if len(paper.Title) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title[0]))
	}

	if len(paper.Author) > 0 {
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
		if len(authorNames) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(authorNames, ", ")))
		}
	}

	if len(paper.ContainerTitle) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📍 期刊: %s\n", paper.ContainerTitle[0]))
	}
	if paper.Publisher != "" {
		resultBuilder.WriteString(fmt.Sprintf("🏢 出版商: %s\n", paper.Publisher))
	}
	if paper.Volume != "" {
		resultBuilder.WriteString(fmt.Sprintf("📖 卷号: %s\n", paper.Volume))
	}
	if paper.Issue != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 期号: %s\n", paper.Issue))
	}
	if paper.Page != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 页码: %s\n", paper.Page))
	}
	if paper.IsReferencedByCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📊 被引用数: %d\n", paper.IsReferencedByCount))
	}
	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("🌐 Crossref链接: %s\n", paper.URL))
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Crossref论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// Scopus搜索功能 (需要API密钥)
func searchScopusPapers(ctx context.Context, req *mcp.CallToolRequest, params *scopus.ScopusSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索Scopus论文 ==========")

	// 从环境变量获取API密钥
	apiKey := os.Getenv("SCOPUS_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("Scopus API密钥未设置，请设置SCOPUS_API_KEY环境变量")
	}

	// 设置默认值
	if params.Start < 0 {
		params.Start = 0
	}
	if params.Count <= 0 {
		params.Count = 25
	}
	if params.Count > 200 {
		params.Count = 200
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索Scopus: 关键词='%s', Start=%d, Count=%d", params.Query, params.Start, params.Count)

	// 创建Scopus客户端并搜索
	client := scopus.NewScopusClient(apiKey)
	result, err := client.SearchPapers(ctx, params.Query, params.Start, params.Count)
	if err != nil {
		log.Printf("[ERROR] Scopus搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 Scopus论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Start+1, params.Start+len(result.Papers), result.TotalCount))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		resultBuilder.WriteString(fmt.Sprintf("   EID: %s\n", paper.EID))
		
		if len(paper.Author) > 0 {
			var authorNames []string
			for _, author := range paper.Author {
				authorNames = append(authorNames, author.AuthorName)
			}
			resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(authorNames, ", ")))
		}
		
		if paper.PublicationName != "" {
			resultBuilder.WriteString(fmt.Sprintf("   期刊: %s\n", paper.PublicationName))
		}
		if paper.CoverDate != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发表日期: %s\n", paper.CoverDate))
		}
		if paper.CitedByCount > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   引用数: %d\n", paper.CitedByCount))
		}
		if paper.DOI != "" {
			resultBuilder.WriteString(fmt.Sprintf("   DOI: %s\n", paper.DOI))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		resultBuilder.WriteString("\n")
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Scopus搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// ADSABS搜索功能 (需要API密钥)
func searchAdsabsPapers(ctx context.Context, req *mcp.CallToolRequest, params *adsabs.AdsabsSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索ADSABS论文 ==========")

	// 从环境变量获取API密钥
	apiKey := os.Getenv("ADSABS_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("ADSABS API密钥未设置，请设置ADSABS_API_KEY环境变量")
	}

	// 设置默认值
	if params.Start < 0 {
		params.Start = 0
	}
	if params.Rows <= 0 {
		params.Rows = 10
	}
	if params.Rows > 2000 {
		params.Rows = 2000
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索ADSABS: 关键词='%s', Start=%d, Rows=%d", params.Query, params.Start, params.Rows)

	// 创建ADSABS客户端并搜索
	client := adsabs.NewAdsabsClient(apiKey)
	result, err := client.SearchPapers(ctx, params.Query, params.Start, params.Rows)
	if err != nil {
		log.Printf("[ERROR] ADSABS搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 ADSABS论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Start+1, params.Start+len(result.Papers), result.NumFound))

	for i, paper := range result.Papers {
		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, title))
		resultBuilder.WriteString(fmt.Sprintf("   Bibcode: %s\n", paper.Bibcode))
		
		if len(paper.Author) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(paper.Author, ", ")))
		}
		
		if paper.Pub != "" {
			resultBuilder.WriteString(fmt.Sprintf("   期刊: %s\n", paper.Pub))
		}
		if paper.PubDate != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发表日期: %s\n", paper.PubDate))
		}
		if paper.CitationCount > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   引用数: %d\n", paper.CitationCount))
		}
		if len(paper.DOI) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   DOI: %s\n", paper.DOI[0]))
		}
		if paper.Abstract != "" {
			abstract := paper.Abstract
			if len(abstract) > 300 {
				abstract = abstract[:300] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", abstract))
		}
		resultBuilder.WriteString("\n")
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== ADSABS搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// Sci-Hub搜索功能
func searchSciHubPapers(ctx context.Context, req *mcp.CallToolRequest, params *scihub.SciHubSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索Sci-Hub论文 ==========")

	// 设置默认值
	if params.MaxResults <= 0 {
		params.MaxResults = 10
	}
	if params.MaxResults > 50 {
		params.MaxResults = 50
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索Sci-Hub: 关键词='%s', MaxResults=%d", params.Query, params.MaxResults)

	// 创建Sci-Hub客户端并搜索
	client := scihub.NewSciHubClient()
	result, err := client.SearchPapers(ctx, params.Query, params.MaxResults)
	if err != nil {
		log.Printf("[ERROR] Sci-Hub搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 Sci-Hub论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("找到%d篇论文\n\n", len(result.Papers)))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		if paper.DOI != "" {
			resultBuilder.WriteString(fmt.Sprintf("   DOI: %s\n", paper.DOI))
		}
		if paper.Authors != "" {
			resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", paper.Authors))
		}
		if paper.Journal != "" {
			resultBuilder.WriteString(fmt.Sprintf("   期刊: %s\n", paper.Journal))
		}
		if paper.Year != "" {
			resultBuilder.WriteString(fmt.Sprintf("   年份: %s\n", paper.Year))
		}
		resultBuilder.WriteString(fmt.Sprintf("   PDF可用: %t\n", paper.Available))
		if paper.PDFURL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   PDF链接: %s\n", paper.PDFURL))
		}
		if paper.SciHubURL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   Sci-Hub链接: %s\n", paper.SciHubURL))
		}
		resultBuilder.WriteString("\n")
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== Sci-Hub搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

func main() {
	log.Printf("[INFO] ========== 学术论文检索MCP服务器启动 ==========")
	log.Printf("[INFO] 正在初始化多数据源论文检索MCP服务器...")

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "academic-papers-mcp-server",
		Version: "2.0.0",
	}, nil)
	log.Printf("[DEBUG] MCP服务器创建完成，名称: %s, 版本: %s", "academic-papers-mcp-server", "2.0.0")

	// 添加arXiv论文搜索工具
	log.Printf("[DEBUG] 正在注册arXiv论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchArxivPapers",
		Description: "Search academic papers from arXiv. Supports keywords, categories, authors, and advanced search queries. Returns paper titles, authors, abstracts, categories, and download links.",
	}, searchArxivPapers)
	log.Printf("[INFO] arXiv论文搜索工具注册完成")

	// 添加获取arXiv论文详情工具
	log.Printf("[DEBUG] 正在注册arXiv论文详情获取工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getArxivPaper",
		Description: "Get detailed information of a specific paper by arXiv ID. Returns complete paper metadata including title, authors, abstract, categories, publication dates, and download links.",
	}, getArxivPaper)
	log.Printf("[INFO] arXiv论文详情获取工具注册完成")

	// 添加Semantic Scholar论文搜索工具
	log.Printf("[DEBUG] 正在注册Semantic Scholar论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchSemanticScholarPapers",
		Description: "Search academic papers from Semantic Scholar. Supports keywords, author names, venue names with advanced filtering options. Returns paper titles, authors, abstracts, citation counts, and venue information.",
	}, searchSemanticScholarPapers)
	log.Printf("[INFO] Semantic Scholar论文搜索工具注册完成")

	// 添加获取Semantic Scholar论文详情工具
	log.Printf("[DEBUG] 正在注册Semantic Scholar论文详情获取工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getSemanticScholarPaper",
		Description: "Get detailed information of a specific paper by Semantic Scholar Paper ID, DOI, or arXiv ID. Returns complete paper metadata including citation metrics, influential citations, and external IDs.",
	}, getSemanticScholarPaper)
	log.Printf("[INFO] Semantic Scholar论文详情获取工具注册完成")

	// 添加Crossref论文搜索工具
	log.Printf("[DEBUG] 正在注册Crossref论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchCrossrefPapers",
		Description: "Search academic papers from Crossref database. Supports keywords, author names, journal titles, publisher names with DOI-based identification. Returns comprehensive publication metadata.",
	}, searchCrossrefPapers)
	log.Printf("[INFO] Crossref论文搜索工具注册完成")

	// 添加获取Crossref论文详情工具
	log.Printf("[DEBUG] 正在注册Crossref论文详情获取工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getCrossrefPaper",
		Description: "Get detailed information of a specific paper by DOI from Crossref. Returns complete publication metadata including author affiliations, funding information, and license details.",
	}, getCrossrefPaper)
	log.Printf("[INFO] Crossref论文详情获取工具注册完成")

	// 添加Scopus论文搜索工具 (需要API密钥)
	log.Printf("[DEBUG] 正在注册Scopus论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchScopusPapers",
		Description: "Search academic papers from Scopus database. Requires SCOPUS_API_KEY environment variable. Supports advanced queries with author, journal, subject area filtering. Returns citation metrics and affiliation data.",
	}, searchScopusPapers)
	log.Printf("[INFO] Scopus论文搜索工具注册完成")

	// 添加ADSABS论文搜索工具 (需要API密钥)
	log.Printf("[DEBUG] 正在注册ADSABS论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchAdsabsPapers",
		Description: "Search astronomy and astrophysics papers from ADSABS. Requires ADSABS_API_KEY environment variable. Supports bibcode, author, object, and advanced astronomical queries. Returns citation counts and astronomical object associations.",
	}, searchAdsabsPapers)
	log.Printf("[INFO] ADSABS论文搜索工具注册完成")

	// 添加Sci-Hub论文搜索工具
	log.Printf("[DEBUG] 正在注册Sci-Hub论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchSciHubPapers",
		Description: "Search and check PDF availability from Sci-Hub. Supports DOI, PMID, and title searches. Returns PDF download links when available. Note: Respects applicable laws and terms of service.",
	}, searchSciHubPapers)
	log.Printf("[INFO] Sci-Hub论文搜索工具注册完成")

	// Create the streamable HTTP handler.
	log.Printf("[DEBUG] 正在创建HTTP处理程序...")
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		log.Printf("[DEBUG] HTTP请求处理: %s %s", req.Method, req.URL.Path)
		return server
	}, nil)
	log.Printf("[INFO] HTTP处理程序创建完成")

	handlerWithLogging := loggingHandler(handler)
	log.Printf("[DEBUG] 日志中间件已附加")

	url := "http://127.0.0.1:8080/"
	log.Printf("[INFO] 学术论文检索MCP服务器正在监听: %s", url)
	log.Printf("[INFO] 可用的工具:")
	log.Printf("[INFO]   📚 arXiv:")
	log.Printf("[INFO]     - searchArxivPapers: 搜索arXiv论文")
	log.Printf("[INFO]     - getArxivPaper: 获取特定arXiv论文详情")
	log.Printf("[INFO]   🧠 Semantic Scholar:")
	log.Printf("[INFO]     - searchSemanticScholarPapers: 搜索Semantic Scholar论文")
	log.Printf("[INFO]     - getSemanticScholarPaper: 获取特定Semantic Scholar论文详情")
	log.Printf("[INFO]   🔗 Crossref:")
	log.Printf("[INFO]     - searchCrossrefPapers: 搜索Crossref论文")
	log.Printf("[INFO]     - getCrossrefPaper: 获取特定Crossref论文详情")
	log.Printf("[INFO]   📊 Scopus (需要API密钥):")
	log.Printf("[INFO]     - searchScopusPapers: 搜索Scopus论文")
	log.Printf("[INFO]   🌌 ADSABS (需要API密钥):")
	log.Printf("[INFO]     - searchAdsabsPapers: 搜索天体物理学论文")
	log.Printf("[INFO]   📄 Sci-Hub:")
	log.Printf("[INFO]     - searchSciHubPapers: 搜索和获取PDF")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 环境变量配置:")
	log.Printf("[INFO]   - SCOPUS_API_KEY: Scopus API密钥 (可选)")
	log.Printf("[INFO]   - ADSABS_API_KEY: ADSABS API密钥 (可选)")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 使用示例:")
	log.Printf("[INFO]   arXiv搜索: {\"query\": \"machine learning\", \"max_results\": 5}")
	log.Printf("[INFO]   DOI查询: {\"query\": \"10.1038/nature12373\"}")
	log.Printf("[INFO]   多源搜索: 使用不同工具获取更全面的结果")

	// Start the HTTP server with logging handler.
	log.Printf("[INFO] ========== 服务器启动完成，等待连接 ==========")
	if err := http.ListenAndServe(":8080", handlerWithLogging); err != nil {
		log.Printf("[FATAL] 服务器启动失败: %v", err)
		log.Fatalf("Server failed: %v", err)
	}
}