package arxiv

import "encoding/xml"

// Paper 论文信息结构体
type Paper struct {
	ID         string   `json:"id"`          // arXiv ID
	Title      string   `json:"title"`       // 标题
	Authors    []string `json:"authors"`     // 作者列表
	Abstract   string   `json:"abstract"`    // 摘要
	Categories []string `json:"categories"`  // 分类
	Published  string   `json:"published"`   // 发布日期
	Updated    string   `json:"updated"`     // 更新日期
	URL        string   `json:"url"`         // arXiv URL
	PDFURL     string   `json:"pdf_url"`     // PDF下载链接
	DOI        string   `json:"doi"`         // DOI
	JournalRef string   `json:"journal_ref"` // 期刊引用
	Comment    string   `json:"comment"`     // 注释
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query      string  `json:"query"`       // 查询关键词
	Start      int     `json:"start"`       // 开始位置
	MaxResults int     `json:"max_results"` // 最大结果数
	TotalCount int     `json:"total_count"` // 总数量
	Papers     []Paper `json:"papers"`      // 论文列表
}

// ArxivSearchParam arXiv搜索参数
type ArxivSearchParam struct {
	Query      string `json:"query"`       // 搜索关键词
	Start      int    `json:"start"`       // 开始位置，默认0
	MaxResults int    `json:"max_results"` // 最大结果数，默认10
	SortBy     string `json:"sort_by"`     // 排序方式: relevance, lastUpdatedDate, submittedDate
	SortOrder  string `json:"sort_order"`  // 排序顺序: ascending, descending
}

// ArxivGetParam 获取特定论文参数
type ArxivGetParam struct {
	ID string `json:"id"` // arXiv ID (例如: 2301.07041)
}

// arXiv API XML响应结构体
type AtomFeed struct {
	XMLName      xml.Name `xml:"feed"`
	Title        string   `xml:"title"`
	ID           string   `xml:"id"`
	Updated      string   `xml:"updated"`
	TotalResults int      `xml:"totalResults"`
	StartIndex   int      `xml:"startIndex"`
	ItemsPerPage int      `xml:"itemsPerPage"`
	Entries      []Entry  `xml:"entry"`
}

type Entry struct {
	ID         string     `xml:"id"`
	Title      string     `xml:"title"`
	Summary    string     `xml:"summary"`
	Authors    []Author   `xml:"author"`
	Categories []Category `xml:"category"`
	Links      []Link     `xml:"link"`
	Published  string     `xml:"published"`
	Updated    string     `xml:"updated"`
	DOI        string     `xml:"doi"`
	JournalRef string     `xml:"journal_ref"`
	Comment    string     `xml:"comment"`
}

type Author struct {
	Name string `xml:"name"`
}

type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

type Link struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
}
