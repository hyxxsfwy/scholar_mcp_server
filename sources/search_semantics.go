package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Seelly/scholar_mcp_server/common"
)

const (
	semanticRerankWindow = 10

	managedFuturesTerm            = "managed futures"
	commodityTradingAdvisorTerm   = "commodity trading advisor"
	trendFollowingTerm            = "trend following"
	timeSeriesMomentumTerm        = "time series momentum"
	quantitativeFinanceTerm       = "quantitative finance"
	quantitativeTradingTerm       = "quantitative trading"
	computedTomographyTerm        = "computed tomography"
	machineLearningTerm           = "machine learning"
	deepLearningTerm              = "deep learning"
	neuralNetworkTerm             = "neural network"
	computerVisionTerm            = "computer vision"
	imageRecognitionTerm          = "image recognition"
	objectDetectionTerm           = "object detection"
	naturalLanguageProcessingTerm = "natural language processing"
	largeLanguageModelTerm        = "large language model"
)

type semanticPaperText struct {
	title    string
	abstract string
	journal  string
	metadata string
}

type termWeights struct {
	title    float64
	abstract float64
	journal  float64
	metadata float64
}

type semanticSearchPlanContextKey struct{}

type semanticPlanAnalyzerFunc func(context.Context, common.SearchParams, string) (*SemanticSearchPlan, error)
type semanticRerankerFunc func(context.Context, *SemanticSearchPlan, []common.UnifiedPaper) ([]string, error)

var semanticPlanAnalyzer semanticPlanAnalyzerFunc = analyzeSemanticSearchWithLLM
var semanticReranker semanticRerankerFunc = rerankSemanticPapersWithLLM

type SemanticSearchPlan struct {
	Applied         bool     `json:"applied"`
	OriginalQuery   string   `json:"original_query,omitempty"`
	SearchQuery     string   `json:"search_query,omitempty"`
	Context         string   `json:"context,omitempty"`
	ContextLabel    string   `json:"context_label,omitempty"`
	Confidence      float64  `json:"confidence,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
	RequiredTerms   []string `json:"required_terms,omitempty"`
	OptionalTerms   []string `json:"optional_terms,omitempty"`
	NegativeTerms   []string `json:"negative_terms,omitempty"`
	ArxivCategories []string `json:"arxiv_categories,omitempty"`
	Notes           []string `json:"notes,omitempty"`
	RerankWindow    int      `json:"rerank_window,omitempty"`
	Analyzer        string   `json:"analyzer,omitempty"`
	FallbackReason  string   `json:"fallback_reason,omitempty"`
	RerankAnalyzer  string   `json:"rerank_analyzer,omitempty"`
	RerankFallback  string   `json:"rerank_fallback_reason,omitempty"`
}

type semanticRerankCandidate struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Abstract      string   `json:"abstract,omitempty"`
	Authors       []string `json:"authors,omitempty"`
	Journal       string   `json:"journal,omitempty"`
	PublishedDate string   `json:"published_date,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	CitationCount int      `json:"citation_count,omitempty"`
	Source        string   `json:"source,omitempty"`
	DOI           string   `json:"doi,omitempty"`
}

type semanticRerankResponse struct {
	OrderedIDs []string `json:"ordered_ids"`
	Notes      string   `json:"notes,omitempty"`
}

func buildSemanticSearchPlan(ctx context.Context, params common.SearchParams) *SemanticSearchPlan {
	input := semanticPlanInput(params)
	if strings.TrimSpace(input) == "" {
		return nil
	}
	if semanticPlanAnalyzer != nil {
		plan, err := semanticPlanAnalyzer(ctx, params, input)
		if err == nil && plan != nil {
			return plan
		}
		if err != nil {
			log.Printf("[WARN] LLM搜索语义分析失败，使用规则兜底: %v", err)
			plan := buildRuleBasedSemanticSearchPlan(params, input)
			if plan != nil {
				plan.FallbackReason = err.Error()
			}
			return plan
		}
	}

	return buildRuleBasedSemanticSearchPlan(params, input)
}

func buildRuleBasedSemanticSearchPlan(params common.SearchParams, input string) *SemanticSearchPlan {
	plan := &SemanticSearchPlan{
		OriginalQuery: params.Query,
		SearchQuery:   params.Query,
		Context:       "general",
		ContextLabel:  "通用学术检索",
		Confidence:    0.35,
		Keywords:      semanticKeywords(input),
		RerankWindow:  semanticRerankWindow,
		Analyzer:      "rules",
	}

	if applyQuantFinancePlan(plan, input) {
		return plan
	}
	if applyMachineLearningPlan(plan, input) {
		return plan
	}
	if applyComputerVisionPlan(plan, input) {
		return plan
	}
	if applyNLPPlan(plan, input) {
		return plan
	}

	plan.RequiredTerms = append([]string(nil), plan.Keywords...)
	return plan
}

func semanticPlanInput(params common.SearchParams) string {
	parts := []string{
		params.Query,
		params.Title,
		params.Abstract,
		params.Journal,
		params.Publisher,
		strings.Join(params.Categories, " "),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func analyzeSemanticSearchWithLLM(ctx context.Context, params common.SearchParams, input string) (*SemanticSearchPlan, error) {
	config := LoadResultEnrichmentConfig()
	if strings.TrimSpace(config.LLM.APIKey) == "" {
		return nil, fmt.Errorf("未配置LLM API Key")
	}
	config.LLM.MaxTokens = semanticLLMMaxTokens(config.LLM.MaxTokens)
	config.LLM.Temperature = 0.1
	timeout := semanticLLMTimeout(config.TimeoutSeconds)
	analysisCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := newOpenAICompatibleLLMClient(config.LLM, &http.Client{Timeout: timeout})
	content, err := client.GenerateWithSystem(analysisCtx, semanticAnalysisSystemPrompt(), buildSemanticAnalysisPrompt(params, input))
	if err != nil {
		return nil, err
	}

	return parseLLMSemanticSearchPlan(content, params)
}

func semanticLLMMaxTokens(configured int) int {
	if configured <= 0 || configured > 800 {
		return 800
	}
	return configured
}

func semanticLLMTimeout(configuredSeconds int) time.Duration {
	if configuredSeconds <= 0 || configuredSeconds > 20 {
		configuredSeconds = 20
	}
	return time.Duration(configuredSeconds) * time.Second
}

func semanticAnalysisSystemPrompt() string {
	return "你是学术搜索查询规划器。你的任务是识别用户检索意图、领域语义、歧义，并把搜索请求改写为适合学术数据库的英文关键词和结构化过滤条件。只输出JSON，不要输出Markdown或解释。"
}

func buildSemanticAnalysisPrompt(params common.SearchParams, input string) string {
	payload := map[string]interface{}{
		"raw_input": input,
		"params": map[string]interface{}{
			"query":         params.Query,
			"title":         params.Title,
			"abstract":      params.Abstract,
			"journal":       params.Journal,
			"publisher":     params.Publisher,
			"categories":    params.Categories,
			"year":          params.Year,
			"year_range":    params.YearRange,
			"type":          params.Type,
			"sort_by":       params.SortBy,
			"sort_order":    params.SortOrder,
			"min_citations": params.MinCitations,
		},
	}
	data, _ := json.Marshal(payload)

	var builder strings.Builder
	builder.WriteString("请分析以下学术搜索请求，返回严格JSON对象。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. search_query 使用英文关键词，适合 OpenAlex/Crossref/Semantic Scholar/arXiv 检索。\n")
	builder.WriteString("2. keywords 应包含领域术语、同义词和必要的英文翻译。\n")
	builder.WriteString("3. required_terms 用于相关性重排，negative_terms 用于排除明显歧义。\n")
	builder.WriteString("4. arxiv_categories 只在高度确定时填写合法 arXiv 分类，例如 q-fin.TR, q-fin.PM, q-fin.ST, cs.LG, stat.ML, cs.CV, cs.CL。\n")
	builder.WriteString("5. 不要编造用户没有表达的作者、期刊或年份过滤。\n")
	builder.WriteString("JSON Schema：{\"applied\":true,\"context\":\"short_machine_name\",\"context_label\":\"中文领域标签\",\"confidence\":0.0,\"search_query\":\"...\",\"keywords\":[\"...\"],\"required_terms\":[\"...\"],\"optional_terms\":[\"...\"],\"negative_terms\":[\"...\"],\"arxiv_categories\":[\"...\"],\"notes\":[\"...\"]}\n")
	builder.WriteString("搜索请求：\n")
	builder.Write(data)
	return builder.String()
}

func parseLLMSemanticSearchPlan(content string, params common.SearchParams) (*SemanticSearchPlan, error) {
	jsonContent := extractJSONObject(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("LLM语义分析未返回JSON对象")
	}

	var plan SemanticSearchPlan
	if err := json.Unmarshal([]byte(jsonContent), &plan); err != nil {
		return nil, fmt.Errorf("解析LLM语义分析JSON失败: %w", err)
	}
	plan = normalizeLLMSemanticSearchPlan(plan, params)
	return &plan, nil
}

func normalizeLLMSemanticSearchPlan(plan SemanticSearchPlan, params common.SearchParams) SemanticSearchPlan {
	plan.OriginalQuery = params.Query
	plan.SearchQuery = strings.Join(strings.Fields(plan.SearchQuery), " ")
	if plan.SearchQuery == "" {
		plan.SearchQuery = params.Query
	}
	plan.Context = sanitizeSemanticIdentifier(plan.Context)
	if plan.Context == "" {
		plan.Context = "general"
	}
	plan.ContextLabel = strings.Join(strings.Fields(plan.ContextLabel), " ")
	if plan.ContextLabel == "" {
		plan.ContextLabel = "通用学术检索"
	}
	if plan.Confidence < 0 {
		plan.Confidence = 0
	}
	if plan.Confidence > 1 {
		plan.Confidence = 1
	}
	plan.Keywords = sanitizeSemanticTerms(plan.Keywords, 16)
	plan.RequiredTerms = sanitizeSemanticTerms(plan.RequiredTerms, 12)
	plan.OptionalTerms = sanitizeSemanticTerms(plan.OptionalTerms, 12)
	plan.NegativeTerms = sanitizeSemanticTerms(plan.NegativeTerms, 12)
	plan.ArxivCategories = sanitizeArxivCategories(plan.ArxivCategories)
	plan.Notes = sanitizeSemanticTerms(plan.Notes, 4)
	if len(plan.Keywords) == 0 {
		plan.Keywords = semanticKeywords(plan.SearchQuery)
	}
	if len(plan.RequiredTerms) == 0 {
		plan.RequiredTerms = append([]string(nil), plan.Keywords...)
	}
	plan.RerankWindow = semanticRerankWindow
	plan.Analyzer = "llm"
	plan.Applied = true
	return plan
}

func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return ""
	}
	return content[start : end+1]
}

func sanitizeSemanticIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func sanitizeSemanticTerms(values []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if value == "" || len([]rune(value)) > 120 {
			continue
		}
		cleaned = append(cleaned, value)
		if len(cleaned) >= limit {
			break
		}
	}
	return uniqueNonEmpty(cleaned)
}

func sanitizeArxivCategories(categories []string) []string {
	pattern := regexp.MustCompile(`^[a-z-]+(\.[A-Z]{2})?$`)
	cleaned := make([]string, 0, len(categories))
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category == "" || !pattern.MatchString(category) {
			continue
		}
		cleaned = append(cleaned, category)
	}
	return uniqueNonEmpty(cleaned)
}

func contextWithSemanticSearchPlan(ctx context.Context, plan *SemanticSearchPlan) context.Context {
	if plan == nil {
		return ctx
	}
	return context.WithValue(ctx, semanticSearchPlanContextKey{}, plan)
}

func semanticSearchPlanFromContext(ctx context.Context) *SemanticSearchPlan {
	if ctx == nil {
		return nil
	}
	plan, _ := ctx.Value(semanticSearchPlanContextKey{}).(*SemanticSearchPlan)
	return plan
}

func applyQuantFinancePlan(plan *SemanticSearchPlan, input string) bool {
	lowerInput := strings.ToLower(input)
	financeHits := countContainsAny(lowerInput, []string{
		"量化", "金融", "期货", "商品交易顾问", "管理期货", "趋势跟踪", "趋势追踪", "动量", "时序动量", "资产配置", "金融工程",
		managedFuturesTerm, commodityTradingAdvisorTerm, trendFollowingTerm, timeSeriesMomentumTerm, quantitativeFinanceTerm, quantitativeTradingTerm, "futures", "portfolio", "asset pricing",
	})
	ctaFinancial := strings.Contains(lowerInput, "cta") && financeHits > 0 && !containsAny(lowerInput, []string{computedTomographyTerm, "angiography", "coronary", "artery"})
	if financeHits < 2 && !ctaFinancial {
		return false
	}

	keywords := []string{managedFuturesTerm, trendFollowingTerm, timeSeriesMomentumTerm, commodityTradingAdvisorTerm, "quantitative trading strategy", "futures", "momentum"}
	if strings.Contains(lowerInput, "cta") {
		keywords = append(keywords, "CTA")
	}

	plan.Applied = true
	plan.Context = "quant_finance_trading"
	plan.ContextLabel = "量化金融/CTA策略"
	plan.Confidence = 0.9
	plan.SearchQuery = strings.Join(uniqueNonEmpty(keywords), " ")
	plan.Keywords = uniqueNonEmpty(append(keywords, plan.Keywords...))
	plan.RequiredTerms = []string{managedFuturesTerm, trendFollowingTerm, timeSeriesMomentumTerm, commodityTradingAdvisorTerm, "futures", "momentum", "CTA"}
	plan.OptionalTerms = []string{quantitativeFinanceTerm, quantitativeTradingTerm, "trading strategy", "portfolio", "asset allocation", "returns", "commodity futures"}
	plan.NegativeTerms = []string{
		computedTomographyTerm, "angiography", "coronary", "artery", "stenosis", "disease", "injury", "health", "clinical", "patient", "cancer",
		"climate", "warming", "emission", "sustainability", "environmental", "metaverse", "language learning",
	}
	plan.ArxivCategories = []string{"q-fin.TR", "q-fin.PM", "q-fin.ST"}
	plan.Notes = []string{"识别为CTA/管理期货/趋势跟踪语义，使用英文金融文献常用表达检索。"}
	plan.Analyzer = "rules"
	return true
}

func applyMachineLearningPlan(plan *SemanticSearchPlan, input string) bool {
	lowerInput := strings.ToLower(input)
	if !containsAny(lowerInput, []string{machineLearningTerm, deepLearningTerm, neuralNetworkTerm, "transformer", "机器学习", "深度学习", "神经网络"}) {
		return false
	}

	plan.Applied = true
	plan.Context = "machine_learning"
	plan.ContextLabel = "机器学习"
	plan.Confidence = 0.78
	plan.Keywords = uniqueNonEmpty(append([]string{machineLearningTerm, deepLearningTerm, neuralNetworkTerm}, plan.Keywords...))
	plan.RequiredTerms = []string{machineLearningTerm, deepLearningTerm, neuralNetworkTerm}
	plan.ArxivCategories = []string{"cs.LG", "stat.ML"}
	plan.Notes = []string{"识别为机器学习语义，arXiv检索会优先限定 cs.LG/stat.ML。"}
	plan.Analyzer = "rules"
	return true
}

func applyComputerVisionPlan(plan *SemanticSearchPlan, input string) bool {
	lowerInput := strings.ToLower(input)
	if !containsAny(lowerInput, []string{computerVisionTerm, imageRecognitionTerm, objectDetectionTerm, "图像识别", "目标检测", "计算机视觉"}) {
		return false
	}

	plan.Applied = true
	plan.Context = "computer_vision"
	plan.ContextLabel = "计算机视觉"
	plan.Confidence = 0.78
	plan.Keywords = uniqueNonEmpty(append([]string{computerVisionTerm, imageRecognitionTerm, objectDetectionTerm}, plan.Keywords...))
	plan.RequiredTerms = []string{computerVisionTerm, imageRecognitionTerm, objectDetectionTerm}
	plan.ArxivCategories = []string{"cs.CV"}
	plan.Notes = []string{"识别为计算机视觉语义，arXiv检索会限定 cs.CV。"}
	plan.Analyzer = "rules"
	return true
}

func applyNLPPlan(plan *SemanticSearchPlan, input string) bool {
	lowerInput := strings.ToLower(input)
	if !containsAny(lowerInput, []string{naturalLanguageProcessingTerm, largeLanguageModelTerm, "llm", "nlp", "自然语言处理", "大语言模型"}) {
		return false
	}

	plan.Applied = true
	plan.Context = "natural_language_processing"
	plan.ContextLabel = "自然语言处理"
	plan.Confidence = 0.78
	plan.Keywords = uniqueNonEmpty(append([]string{naturalLanguageProcessingTerm, largeLanguageModelTerm, "language model"}, plan.Keywords...))
	plan.RequiredTerms = []string{naturalLanguageProcessingTerm, largeLanguageModelTerm, "language model"}
	plan.ArxivCategories = []string{"cs.CL"}
	plan.Notes = []string{"识别为自然语言处理语义，arXiv检索会限定 cs.CL。"}
	plan.Analyzer = "rules"
	return true
}

func applySemanticSearchPlan(params common.SearchParams, plan *SemanticSearchPlan) common.SearchParams {
	if plan == nil || strings.TrimSpace(plan.SearchQuery) == "" {
		return params
	}
	if plan.Applied && strings.TrimSpace(plan.SearchQuery) != strings.TrimSpace(params.Query) {
		params.Query = plan.SearchQuery
	}
	return params
}

func applySmartArxivCategories(ctx context.Context, params common.SearchParams) common.SearchParams {
	if len(params.Categories) > 0 {
		return params
	}
	plan := semanticSearchPlanFromContext(ctx)
	if plan == nil {
		plan = buildSemanticSearchPlan(ctx, params)
	}
	if plan == nil || len(plan.ArxivCategories) == 0 {
		return params
	}
	params.Categories = append([]string(nil), plan.ArxivCategories...)
	return params
}

func semanticCandidateLimit(offset, limit int) int {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	candidateLimit := offset + limit
	if candidateLimit < semanticRerankWindow {
		candidateLimit = semanticRerankWindow
	}
	return candidateLimit
}

func rerankTopSemanticWindow(ctx context.Context, papers []common.UnifiedPaper, plan *SemanticSearchPlan, window int) {
	if len(papers) < 2 || plan == nil || !plan.Applied {
		return
	}
	if window <= 0 || window > len(papers) {
		window = len(papers)
	}
	top := papers[:window]
	if semanticReranker != nil {
		orderedIDs, err := semanticReranker(ctx, plan, top)
		if err == nil && applySemanticRerankOrder(top, orderedIDs) {
			plan.RerankAnalyzer = "llm"
			plan.RerankFallback = ""
			return
		}
		if err != nil {
			log.Printf("[WARN] LLM搜索结果重排失败，使用本地重排兜底: %v", err)
			plan.RerankFallback = err.Error()
		}
	}
	plan.RerankAnalyzer = "rules"
	rerankTopSemanticWindowByScore(top, plan)
}

func rerankTopSemanticWindowByScore(top []common.UnifiedPaper, plan *SemanticSearchPlan) {
	sort.SliceStable(top, func(i, j int) bool {
		scoreI := semanticPaperScore(top[i], plan)
		scoreJ := semanticPaperScore(top[j], plan)
		if scoreI == scoreJ {
			return top[i].CitationCount > top[j].CitationCount
		}
		return scoreI > scoreJ
	})
}

func rerankSemanticPapersWithLLM(ctx context.Context, plan *SemanticSearchPlan, papers []common.UnifiedPaper) ([]string, error) {
	if len(papers) < 2 {
		return nil, fmt.Errorf("候选论文不足，无需LLM重排")
	}
	config := LoadResultEnrichmentConfig()
	if strings.TrimSpace(config.LLM.APIKey) == "" {
		return nil, fmt.Errorf("未配置LLM API Key")
	}
	config.LLM.MaxTokens = semanticRerankLLMMaxTokens(config.LLM.MaxTokens)
	config.LLM.Temperature = 0
	timeout := semanticLLMTimeout(config.TimeoutSeconds)
	rerankCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := newOpenAICompatibleLLMClient(config.LLM, &http.Client{Timeout: timeout})
	content, err := client.GenerateWithSystem(rerankCtx, semanticRerankSystemPrompt(), buildSemanticRerankPrompt(plan, papers))
	if err != nil {
		return nil, err
	}
	return parseLLMSemanticRerankOrder(content, len(papers))
}

func semanticRerankLLMMaxTokens(configured int) int {
	if configured <= 0 || configured > 1000 {
		return 1000
	}
	return configured
}

func semanticRerankSystemPrompt() string {
	return "你是学术论文搜索结果重排器。你只能根据给定查询语义和候选论文元数据判断相关性，返回严格JSON，不要输出Markdown或解释。"
}

func buildSemanticRerankPrompt(plan *SemanticSearchPlan, papers []common.UnifiedPaper) string {
	candidates := make([]semanticRerankCandidate, 0, len(papers))
	for candidateIndex, paper := range papers {
		candidates = append(candidates, semanticRerankCandidate{
			ID:            semanticRerankCandidateID(candidateIndex),
			Title:         paper.Title,
			Abstract:      common.TruncateText(paper.Abstract, 700),
			Authors:       truncateStringSlice(paper.Authors, 6),
			Journal:       paper.Journal,
			PublishedDate: paper.PublishedDate,
			Categories:    truncateStringSlice(paper.Categories, 8),
			Keywords:      truncateStringSlice(paper.Keywords, 8),
			CitationCount: paper.CitationCount,
			Source:        paper.Source,
			DOI:           paper.DOI,
		})
	}
	payload := map[string]interface{}{
		"query_semantics": map[string]interface{}{
			"original_query":   plan.OriginalQuery,
			"search_query":     plan.SearchQuery,
			"context":          plan.Context,
			"context_label":    plan.ContextLabel,
			"keywords":         plan.Keywords,
			"required_terms":   plan.RequiredTerms,
			"optional_terms":   plan.OptionalTerms,
			"negative_terms":   plan.NegativeTerms,
			"arxiv_categories": plan.ArxivCategories,
		},
		"candidates": candidates,
	}
	data, _ := json.Marshal(payload)

	var builder strings.Builder
	builder.WriteString("请按与搜索语义的学术相关性对候选论文重新排序。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. 只使用候选论文元数据，不要引入外部事实。\n")
	builder.WriteString("2. 优先考虑标题、摘要、分类与 query_semantics 的一致性，引用数只能作为同等相关时的次要信号。\n")
	builder.WriteString("3. 如果候选明显命中 negative_terms 或属于歧义领域，应排到后面。\n")
	builder.WriteString("4. 必须返回所有候选 ID，每个 ID 只出现一次。\n")
	builder.WriteString("JSON Schema：{\"ordered_ids\":[\"p1\",\"p2\"],\"notes\":\"简短说明\"}\n")
	builder.WriteString("输入：\n")
	builder.Write(data)
	return builder.String()
}

func parseLLMSemanticRerankOrder(content string, candidateCount int) ([]string, error) {
	jsonContent := extractJSONObject(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("LLM重排未返回JSON对象")
	}

	var response semanticRerankResponse
	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return nil, fmt.Errorf("解析LLM重排JSON失败: %w", err)
	}
	orderedIDs := sanitizeSemanticRerankIDs(response.OrderedIDs, candidateCount)
	if len(orderedIDs) == 0 {
		return nil, fmt.Errorf("LLM重排未返回有效候选ID")
	}
	return orderedIDs, nil
}

func sanitizeSemanticRerankIDs(values []string, candidateCount int) []string {
	seen := make(map[string]struct{}, len(values))
	orderedIDs := make([]string, 0, len(values))
	validIDs := make(map[string]struct{}, candidateCount)
	for candidateIndex := 0; candidateIndex < candidateCount; candidateIndex++ {
		validIDs[semanticRerankCandidateID(candidateIndex)] = struct{}{}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, valid := validIDs[value]; !valid {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		orderedIDs = append(orderedIDs, value)
	}
	return orderedIDs
}

func applySemanticRerankOrder(papers []common.UnifiedPaper, orderedIDs []string) bool {
	if len(papers) < 2 || len(orderedIDs) == 0 {
		return false
	}
	original := append([]common.UnifiedPaper(nil), papers...)
	used := make([]bool, len(original))
	reordered := make([]common.UnifiedPaper, 0, len(original))
	for _, orderedID := range orderedIDs {
		candidateIndex, ok := semanticRerankCandidateIndex(orderedID, len(original))
		if !ok || used[candidateIndex] {
			continue
		}
		used[candidateIndex] = true
		reordered = append(reordered, original[candidateIndex])
	}
	if len(reordered) == 0 {
		return false
	}
	for candidateIndex, paper := range original {
		if !used[candidateIndex] {
			reordered = append(reordered, paper)
		}
	}
	copy(papers, reordered)
	return true
}

func semanticRerankCandidateID(candidateIndex int) string {
	return fmt.Sprintf("p%d", candidateIndex+1)
}

func semanticRerankCandidateIndex(candidateID string, candidateCount int) (int, bool) {
	if !strings.HasPrefix(candidateID, "p") {
		return 0, false
	}
	var oneBasedIndex int
	if _, err := fmt.Sscanf(candidateID, "p%d", &oneBasedIndex); err != nil {
		return 0, false
	}
	candidateIndex := oneBasedIndex - 1
	if candidateIndex < 0 || candidateIndex >= candidateCount {
		return 0, false
	}
	return candidateIndex, true
}

func truncateStringSlice(values []string, limit int) []string {
	if limit <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) <= limit {
		return append([]string(nil), values...)
	}
	return append([]string(nil), values[:limit]...)
}

func pageSemanticPapers(papers []common.UnifiedPaper, offset, limit int) []common.UnifiedPaper {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if offset >= len(papers) {
		return []common.UnifiedPaper{}
	}
	end := offset + limit
	if end > len(papers) {
		end = len(papers)
	}
	return papers[offset:end]
}

func semanticPaperScore(paper common.UnifiedPaper, plan *SemanticSearchPlan) float64 {
	if plan == nil {
		return basePaperScore(paper)
	}

	fields := semanticPaperText{
		title:    strings.ToLower(paper.Title),
		abstract: strings.ToLower(paper.Abstract),
		journal:  strings.ToLower(paper.Journal + " " + paper.Publisher),
		metadata: strings.ToLower(strings.Join(append(append([]string{}, paper.Categories...), paper.Keywords...), " ")),
	}
	fullText := strings.Join([]string{fields.title, fields.abstract, fields.journal, fields.metadata, strings.ToLower(paper.Source), strings.ToLower(paper.Type)}, " ")

	score := basePaperScore(paper)
	for _, term := range plan.RequiredTerms {
		score += weightedTermScore(term, fields, termWeights{title: 22, abstract: 10, journal: 4, metadata: 8})
	}
	for _, term := range plan.OptionalTerms {
		score += weightedTermScore(term, fields, termWeights{title: 10, abstract: 5, journal: 2, metadata: 4})
	}
	for _, term := range plan.Keywords {
		score += weightedTermScore(term, fields, termWeights{title: 12, abstract: 5, journal: 2, metadata: 4})
	}
	for _, term := range plan.NegativeTerms {
		if containsTerm(fullText, term) {
			score -= 45
		}
	}

	if plan.Context == "quant_finance_trading" {
		if containsAny(fullText, []string{"finance", "financial", "futures", "trading", "portfolio", "asset pricing", "commodity", "ssrn", "q-fin"}) {
			score += 18
		}
		if containsAny(fields.metadata, []string{"finance", "economics", "q-fin.tr", "q-fin.pm", "q-fin.st"}) {
			score += 18
		}
		if containsAny(fullText, []string{computedTomographyTerm, "angiography", "coronary", "clinical", "patient"}) {
			score -= 70
		}
	}

	return score
}

func basePaperScore(paper common.UnifiedPaper) float64 {
	score := math.Log1p(float64(maxInt(paper.CitationCount, 0))) * 1.5
	if paper.IsOpenAccess {
		score += 1
	}
	if strings.TrimSpace(paper.Abstract) != "" {
		score += 1.5
	}
	return score
}

func weightedTermScore(term string, fields semanticPaperText, weights termWeights) float64 {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return 0
	}
	score := 0.0
	if containsTerm(fields.title, term) {
		score += weights.title
	}
	if containsTerm(fields.abstract, term) {
		score += weights.abstract
	}
	if containsTerm(fields.journal, term) {
		score += weights.journal
	}
	if containsTerm(fields.metadata, term) {
		score += weights.metadata
	}
	return score
}

func semanticSearchHints(plan *SemanticSearchPlan) []common.SearchHint {
	if plan == nil || !plan.Applied {
		return nil
	}

	hints := []common.SearchHint{
		{
			Query:       plan.SearchQuery,
			Type:        "semantic_query",
			Confidence:  plan.Confidence,
			Description: "基于搜索语义改写后的关键词",
		},
	}
	if len(plan.ArxivCategories) > 0 {
		hints = append(hints, common.SearchHint{
			Query:       strings.Join(plan.ArxivCategories, ", "),
			Type:        "arxiv_category_filter",
			Confidence:  plan.Confidence,
			Description: "自动启用的arXiv分类过滤",
		})
	}
	return hints
}

func semanticKeywords(input string) []string {
	fields := strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '.')
	})
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true, "that": true, "this": true, "into": true,
		"研究": true, "论文": true, "文献": true, "搜索": true, "策略": true,
	}
	var keywords []string
	for _, field := range fields {
		field = strings.Trim(field, " .,:;!?()[]{}\"'")
		if len([]rune(field)) < 2 || stopWords[field] {
			continue
		}
		keywords = append(keywords, field)
	}
	return uniqueNonEmpty(keywords)
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if containsTerm(text, term) {
			return true
		}
	}
	return false
}

func countContainsAny(text string, terms []string) int {
	count := 0
	for _, term := range terms {
		if containsTerm(text, term) {
			count++
		}
	}
	return count
}

func containsTerm(text, term string) bool {
	text = strings.ToLower(text)
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return false
	}
	if strings.ContainsAny(term, " -.") || containsNonASCII(term) {
		return strings.Contains(text, term)
	}
	pattern := regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(term) + `([^a-z0-9]|$)`)
	return pattern.MatchString(text)
}

func containsNonASCII(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, value := range values {
		value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
