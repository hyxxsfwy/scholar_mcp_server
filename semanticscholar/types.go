package semanticscholar

// Paper Semantic Scholar论文信息结构体
type Paper struct {
	PaperID      string   `json:"paperId"`      // Semantic Scholar ID
	CorpusID     int64    `json:"corpusId"`     // Corpus ID
	URL          string   `json:"url"`          // Semantic Scholar URL
	Title        string   `json:"title"`        // 标题
	Abstract     string   `json:"abstract"`     // 摘要
	Venue        string   `json:"venue"`        // 发表场所
	Year         int      `json:"year"`         // 发表年份
	ReferenceCount int    `json:"referenceCount"` // 引用数量
	CitationCount  int    `json:"citationCount"`  // 被引用数量
	InfluentialCitationCount int `json:"influentialCitationCount"` // 有影响力的引用数量
	IsOpenAccess bool     `json:"isOpenAccess"` // 是否开放获取
	FieldsOfStudy []string `json:"fieldsOfStudy"` // 研究领域
	Authors      []Author `json:"authors"`      // 作者列表
	ExternalIDs  ExternalIDs `json:"externalIds"` // 外部ID
	PublicationTypes []string `json:"publicationTypes"` // 发表类型
	PublicationDate  string   `json:"publicationDate"`  // 发表日期
	Journal          Journal  `json:"journal"`          // 期刊信息
}

// Author 作者信息
type Author struct {
	AuthorID string `json:"authorId"` // 作者ID
	Name     string `json:"name"`     // 作者姓名
}

// ExternalIDs 外部ID信息
type ExternalIDs struct {
	ArXiv      string `json:"ArXiv"`      // arXiv ID
	DBLP       string `json:"DBLP"`       // DBLP ID
	DOI        string `json:"DOI"`        // DOI
	PubMed     string `json:"PubMed"`     // PubMed ID
	PubMedCentral string `json:"PubMedCentral"` // PMC ID
}

// Journal 期刊信息
type Journal struct {
	Name   string `json:"name"`   // 期刊名称
	Pages  string `json:"pages"`  // 页码
	Volume string `json:"volume"` // 卷号
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query      string  `json:"query"`       // 查询关键词
	Offset     int     `json:"offset"`      // 偏移量
	Limit      int     `json:"limit"`       // 限制数量
	Total      int     `json:"total"`       // 总数量
	Papers     []Paper `json:"data"`        // 论文列表
}

// SemanticScholarSearchParam 搜索参数
type SemanticScholarSearchParam struct {
	Query           string   `json:"query"`           // 搜索关键词
	Year            string   `json:"year"`            // 年份过滤 (例如: "2020-2023")
	Venue           []string `json:"venue"`           // 发表场所过滤
	FieldsOfStudy   []string `json:"fieldsOfStudy"`   // 研究领域过滤
	PublicationTypes []string `json:"publicationTypes"` // 发表类型过滤
	MinCitationCount int     `json:"minCitationCount"` // 最小引用数
	Offset          int      `json:"offset"`          // 偏移量，默认0
	Limit           int      `json:"limit"`           // 限制数量，默认10
	Sort            string   `json:"sort"`            // 排序方式: relevance, citationCount, publicationDate
}

// SemanticScholarGetParam 获取特定论文参数
type SemanticScholarGetParam struct {
	PaperID string `json:"paperId"` // Semantic Scholar Paper ID 或 DOI 或 arXiv ID
}

// API响应结构体
type APIResponse struct {
	Total  int     `json:"total"`
	Offset int     `json:"offset"`
	Next   int     `json:"next"`
	Data   []Paper `json:"data"`
}
