package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidateSearchParams 验证搜索参数
func ValidateSearchParams(params SearchParams) *ValidationResult {
	var errors []ValidationError

	// 验证查询关键词
	if strings.TrimSpace(params.Query) == "" {
		errors = append(errors, ValidationError{
			Field:   "query",
			Message: "查询关键词不能为空",
			Value:   params.Query,
		})
	}

	// 验证偏移量
	if params.Offset < 0 {
		errors = append(errors, ValidationError{
			Field:   "offset",
			Message: "偏移量不能为负数",
			Value:   strconv.Itoa(params.Offset),
		})
	}

	// 验证限制数量
	if params.Limit < 0 {
		errors = append(errors, ValidationError{
			Field:   "limit",
			Message: "限制数量不能为负数",
			Value:   strconv.Itoa(params.Limit),
		})
	}

	if params.Limit > 1000 {
		errors = append(errors, ValidationError{
			Field:   "limit",
			Message: "限制数量不能超过1000",
			Value:   strconv.Itoa(params.Limit),
		})
	}

	// 验证年份格式
	if params.Year != "" && !isValidYear(params.Year) {
		errors = append(errors, ValidationError{
			Field:   "year",
			Message: "年份格式无效，应为YYYY或YYYY-YYYY",
			Value:   params.Year,
		})
	}

	// 验证排序参数
	if params.SortBy != "" && !isValidSortField(params.SortBy) {
		errors = append(errors, ValidationError{
			Field:   "sort_by",
			Message: "无效的排序字段",
			Value:   params.SortBy,
		})
	}

	if params.SortOrder != "" && !isValidSortOrder(params.SortOrder) {
		errors = append(errors, ValidationError{
			Field:   "sort_order",
			Message: "排序顺序必须为asc或desc",
			Value:   params.SortOrder,
		})
	}

	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// NormalizeSearchParams 标准化搜索参数
func NormalizeSearchParams(params *SearchParams) {
	// 清理查询关键词
	params.Query = strings.TrimSpace(params.Query)

	// 设置默认值
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// 标准化排序参数
	if params.SortBy == "" {
		params.SortBy = "relevance"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}

	// 清理字符串字段
	params.Author = strings.TrimSpace(params.Author)
	params.Title = strings.TrimSpace(params.Title)
	params.Abstract = strings.TrimSpace(params.Abstract)
	params.Journal = strings.TrimSpace(params.Journal)
	params.Publisher = strings.TrimSpace(params.Publisher)
	params.Language = strings.TrimSpace(params.Language)
	params.Type = strings.TrimSpace(params.Type)
}

// ExtractDOI 从文本中提取DOI
func ExtractDOI(text string) string {
	// DOI正则表达式
	doiPattern := regexp.MustCompile(`(?i)(?:doi:?\s*)?(?:https?://(?:dx\.)?doi\.org/)?(?:https?://doi\.org/)?(10\.\d{4,}/[^\s<>"{}|\\^` + "`" + `\[\]]+)`)
	matches := doiPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// IsDOI 检查字符串是否为有效的DOI
func IsDOI(s string) bool {
	doiPattern := regexp.MustCompile(`^10\.\d{4,}/[^\s]+$`)
	return doiPattern.MatchString(s)
}

// IsArxivID 检查字符串是否为有效的arXiv ID
func IsArxivID(s string) bool {
	// 新格式: YYMM.NNNN[vN]
	newPattern := regexp.MustCompile(`^\d{4}\.\d{4,5}(v\d+)?$`)
	// 旧格式: subject-class/YYMMnnn
	oldPattern := regexp.MustCompile(`^[a-z-]+/\d{7}$`)
	
	return newPattern.MatchString(s) || oldPattern.MatchString(s)
}

// IsPubMedID 检查字符串是否为有效的PubMed ID
func IsPubMedID(s string) bool {
	// PubMed ID是纯数字
	pmidPattern := regexp.MustCompile(`^\d+$`)
	return pmidPattern.MatchString(s) && len(s) >= 1 && len(s) <= 10
}

// NormalizeDateString 标准化日期字符串
func NormalizeDateString(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// 尝试解析常见的日期格式
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2 January 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02")
		}
	}

	// 如果无法解析，返回原字符串
	return dateStr
}

// SanitizeQuery 清理查询字符串
func SanitizeQuery(query string) string {
	// 移除特殊字符，保留字母、数字、空格和一些常用符号
	reg := regexp.MustCompile(`[^\w\s\-\+\(\)\[\]"':.,?!]`)
	cleaned := reg.ReplaceAllString(query, " ")
	
	// 压缩多个空格为一个
	spaceReg := regexp.MustCompile(`\s+`)
	cleaned = spaceReg.ReplaceAllString(cleaned, " ")
	
	return strings.TrimSpace(cleaned)
}

// SplitAuthors 分割作者字符串
func SplitAuthors(authorsStr string) []string {
	if authorsStr == "" {
		return []string{}
	}

	// 尝试不同的分隔符
	separators := []string{";", ",", " and ", " & "}
	
	for _, sep := range separators {
		if strings.Contains(authorsStr, sep) {
			parts := strings.Split(authorsStr, sep)
			var authors []string
			for _, part := range parts {
				author := strings.TrimSpace(part)
				if author != "" {
					authors = append(authors, author)
				}
			}
			if len(authors) > 1 {
				return authors
			}
		}
	}

	// 如果没有找到分隔符，返回整个字符串作为单个作者
	return []string{strings.TrimSpace(authorsStr)}
}

// FormatAuthors 格式化作者列表
func FormatAuthors(authors []string) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) == 1 {
		return authors[0]
	}
	if len(authors) == 2 {
		return authors[0] + " and " + authors[1]
	}
	
	// 超过2个作者时
	result := strings.Join(authors[:len(authors)-1], ", ")
	result += ", and " + authors[len(authors)-1]
	return result
}

// TruncateText 截断文本到指定长度
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	if maxLength <= 3 {
		return text[:maxLength]
	}
	
	return text[:maxLength-3] + "..."
}

// ExtractKeywords 从文本中提取关键词
func ExtractKeywords(text string, maxKeywords int) []string {
	// 简单的关键词提取，基于词频
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)
	
	// 停用词列表
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true, "this": true,
		"that": true, "these": true, "those": true, "we": true, "they": true,
		"it": true, "its": true, "our": true, "their": true, "from": true,
		"into": true, "during": true, "before": true, "after": true,
	}
	
	// 计算词频
	for _, word := range words {
		// 清理标点符号
		word = regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if len(word) > 2 && !stopWords[word] {
			wordCount[word]++
		}
	}
	
	// 按频率排序
	type wordFreq struct {
		word  string
		count int
	}
	
	var sorted []wordFreq
	for word, count := range wordCount {
		sorted = append(sorted, wordFreq{word, count})
	}
	
	// 简单排序（冒泡排序）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].count < sorted[j+1].count {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	// 提取前N个关键词
	var keywords []string
	limit := maxKeywords
	if len(sorted) < limit {
		limit = len(sorted)
	}
	
	for i := 0; i < limit; i++ {
		keywords = append(keywords, sorted[i].word)
	}
	
	return keywords
}

// GenerateSearchSuggestions 生成搜索建议
func GenerateSearchSuggestions(query string) []SearchHint {
	var suggestions []SearchHint
	
	query = strings.TrimSpace(strings.ToLower(query))
	
	// 如果查询看起来像DOI
	if IsDOI(query) {
		suggestions = append(suggestions, SearchHint{
			Query:       query,
			Type:        "doi",
			Confidence:  0.9,
			Description: "这看起来是一个DOI，建议使用精确搜索",
		})
	}
	
	// 如果查询看起来像arXiv ID
	if IsArxivID(query) {
		suggestions = append(suggestions, SearchHint{
			Query:       query,
			Type:        "arxiv_id",
			Confidence:  0.9,
			Description: "这看起来是一个arXiv ID",
		})
	}
	
	// 如果查询看起来像PubMed ID
	if IsPubMedID(query) {
		suggestions = append(suggestions, SearchHint{
			Query:       "PMID:" + query,
			Type:        "pubmed_id",
			Confidence:  0.8,
			Description: "这看起来是一个PubMed ID",
		})
	}
	
	// 添加一些通用的搜索建议
	if len(query) > 0 {
		// 标题搜索建议
		suggestions = append(suggestions, SearchHint{
			Query:       fmt.Sprintf("title:\"%s\"", query),
			Type:        "title_search",
			Confidence:  0.7,
			Description: "在标题中搜索",
		})
		
		// 作者搜索建议
		suggestions = append(suggestions, SearchHint{
			Query:       fmt.Sprintf("author:\"%s\"", query),
			Type:        "author_search",
			Confidence:  0.6,
			Description: "在作者中搜索",
		})
		
		// 摘要搜索建议
		suggestions = append(suggestions, SearchHint{
			Query:       fmt.Sprintf("abstract:\"%s\"", query),
			Type:        "abstract_search",
			Confidence:  0.5,
			Description: "在摘要中搜索",
		})
	}
	
	return suggestions
}

// 辅助函数

func isValidYear(year string) bool {
	// 支持单年份或年份范围
	if matched, _ := regexp.MatchString(`^\d{4}$`, year); matched {
		return true
	}
	if matched, _ := regexp.MatchString(`^\d{4}-\d{4}$`, year); matched {
		return true
	}
	return false
}

func isValidSortField(field string) bool {
	validFields := []string{
		"relevance", "date", "published_date", "citation_count",
		"read_count", "title", "author", "journal",
	}
	for _, valid := range validFields {
		if field == valid {
			return true
		}
	}
	return false
}

func isValidSortOrder(order string) bool {
	return order == "asc" || order == "desc"
}
