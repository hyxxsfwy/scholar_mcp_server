package sources

import (
	"os"
	"strconv"
)

// SourceFactory 数据源工厂
type SourceFactory struct {
	manager *SourceManager
}

// NewSourceFactory 创建新的数据源工厂
func NewSourceFactory() *SourceFactory {
	return &SourceFactory{
		manager: NewSourceManager(),
	}
}

// InitializeAllSources 初始化所有数据源
func (sf *SourceFactory) InitializeAllSources() *SourceManager {
	// 从环境变量读取配置
	configs := sf.loadConfigsFromEnv()

	sf.registerEnabledSource("arxiv", configs["arxiv"], func(config SourceConfig) PaperSource { return NewArxivSource(config) })
	sf.registerEnabledSource("semantic_scholar", configs["semantic_scholar"], func(config SourceConfig) PaperSource { return NewSemanticScholarSource(config) })
	sf.registerEnabledSource("crossref", configs["crossref"], func(config SourceConfig) PaperSource { return NewCrossrefSource(config) })
	sf.registerEnabledSource("openalex", configs["openalex"], func(config SourceConfig) PaperSource { return NewOpenAlexSource(config) })
	sf.registerEnabledSource("scopus", configs["scopus"], newScopusPaperSource)
	sf.registerEnabledSource("adsabs", configs["adsabs"], newAdsabsPaperSource)
	sf.registerEnabledSource("google_scholar", configs["google_scholar"], newGoogleScholarPaperSource)
	sf.registerEnabledSource("broker_research", configs["broker_research"], newBrokerResearchPaperSource)
	sf.registerEnabledSource("scihub", configs["scihub"], func(config SourceConfig) PaperSource { return NewSciHubSource(config) })

	return sf.manager
}

func (sf *SourceFactory) registerEnabledSource(name string, config SourceConfig, createSource func(SourceConfig) PaperSource) {
	if !config.Enabled {
		return
	}

	source := createSource(config)
	if source != nil {
		sf.manager.RegisterSource(name, source, config)
	}
}

func newScopusPaperSource(config SourceConfig) PaperSource {
	if config.APIKey == "" {
		return nil
	}
	return NewScopusSource(config)
}

func newAdsabsPaperSource(config SourceConfig) PaperSource {
	if config.APIKey == "" {
		return nil
	}
	return NewAdsabsSource(config)
}

func newGoogleScholarPaperSource(config SourceConfig) PaperSource {
	if config.APIKey == "" {
		return nil
	}
	return NewGoogleScholarSource(config)
}

func newBrokerResearchPaperSource(config SourceConfig) PaperSource {
	if config.BaseURL == "" {
		return nil
	}
	return NewBrokerResearchSource(config)
}

// loadConfigsFromEnv 从环境变量加载配置
func (sf *SourceFactory) loadConfigsFromEnv() map[string]SourceConfig {
	configs := make(map[string]SourceConfig)

	// 默认配置
	defaultTimeout := 30
	defaultRetries := 3
	defaultRateLimit := 10

	// 从环境变量读取通用配置
	if timeoutStr := os.Getenv("REQUEST_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			defaultTimeout = timeout
		}
	}

	// arXiv配置
	configs["arxiv"] = SourceConfig{
		Enabled:        getBoolEnv("ENABLE_ARXIV", true),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      defaultRateLimit,
	}

	// Semantic Scholar配置
	configs["semantic_scholar"] = SourceConfig{
		APIKey:         os.Getenv("SEMANTIC_SCHOLAR_API_KEY"),
		Enabled:        getBoolEnv("ENABLE_SEMANTIC_SCHOLAR", true),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      defaultRateLimit,
	}

	// Crossref配置
	configs["crossref"] = SourceConfig{
		APIKey:         os.Getenv("CROSSREF_EMAIL"), // Crossref使用邮箱作为礼貌标识
		Enabled:        getBoolEnv("ENABLE_CROSSREF", true),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      defaultRateLimit,
	}

	// OpenAlex配置
	configs["openalex"] = SourceConfig{
		APIKey:         firstEnvValue("OPENALEX_EMAIL", "CROSSREF_EMAIL"), // OpenAlex使用邮箱进入polite pool
		BaseURL:        os.Getenv("OPENALEX_BASE_URL"),
		Enabled:        getBoolEnv("ENABLE_OPENALEX", true),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      defaultRateLimit,
	}

	// Scopus配置
	configs["scopus"] = SourceConfig{
		APIKey:         os.Getenv("SCOPUS_API_KEY"),
		Enabled:        getBoolEnv("ENABLE_SCOPUS", false), // 默认关闭，需要API密钥
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      5, // Scopus限制更严格
	}

	// ADSABS配置
	configs["adsabs"] = SourceConfig{
		APIKey:         os.Getenv("ADSABS_API_KEY"),
		Enabled:        getBoolEnv("ENABLE_ADSABS", false), // 默认关闭，需要API密钥
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      defaultRateLimit,
	}

	// Google Scholar配置 (通过SerpAPI兼容接口)
	configs["google_scholar"] = SourceConfig{
		APIKey:         firstEnvValue("SERPAPI_API_KEY", "GOOGLE_SCHOLAR_API_KEY"),
		BaseURL:        firstEnvValue("SERPAPI_BASE_URL", "GOOGLE_SCHOLAR_BASE_URL"),
		Enabled:        getBoolEnv("ENABLE_GOOGLE_SCHOLAR", false), // 默认关闭，需要第三方SERP API密钥
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      2,
	}

	// 券商金工研报配置。仅接入用户有权访问的JSON/RSS/Atom源。
	configs["broker_research"] = SourceConfig{
		APIKey:         os.Getenv("BROKER_RESEARCH_API_KEY"),
		BaseURL:        os.Getenv("BROKER_RESEARCH_FEEDS"),
		Enabled:        getBoolEnv("ENABLE_BROKER_RESEARCH", false),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      2,
	}

	// Sci-Hub配置
	configs["scihub"] = SourceConfig{
		Enabled:        getBoolEnv("ENABLE_SCIHUB", true),
		RequestTimeout: defaultTimeout,
		MaxRetries:     defaultRetries,
		RateLimit:      3, // Sci-Hub限制更严格
	}

	return configs
}

// getBoolEnv 从环境变量获取布尔值，如果不存在则返回默认值
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.ParseBool(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func firstEnvValue(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return ""
}

// GetDefaultSourceManager 获取默认配置的数据源管理器
func GetDefaultSourceManager() *SourceManager {
	factory := NewSourceFactory()
	return factory.InitializeAllSources()
}
