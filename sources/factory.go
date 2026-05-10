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

	// 注册arXiv数据源
	if configs["arxiv"].Enabled {
		arxivSource := NewArxivSource(configs["arxiv"])
		sf.manager.RegisterSource("arxiv", arxivSource, configs["arxiv"])
	}

	// 注册Semantic Scholar数据源
	if configs["semantic_scholar"].Enabled {
		ssSource := NewSemanticScholarSource(configs["semantic_scholar"])
		sf.manager.RegisterSource("semantic_scholar", ssSource, configs["semantic_scholar"])
	}

	// 注册Crossref数据源
	if configs["crossref"].Enabled {
		crossrefSource := NewCrossrefSource(configs["crossref"])
		sf.manager.RegisterSource("crossref", crossrefSource, configs["crossref"])
	}

	// 注册Scopus数据源
	if configs["scopus"].Enabled && configs["scopus"].APIKey != "" {
		scopusSource := NewScopusSource(configs["scopus"])
		if scopusSource != nil {
			sf.manager.RegisterSource("scopus", scopusSource, configs["scopus"])
		}
	}

	// 注册ADSABS数据源
	if configs["adsabs"].Enabled && configs["adsabs"].APIKey != "" {
		adsabsSource := NewAdsabsSource(configs["adsabs"])
		if adsabsSource != nil {
			sf.manager.RegisterSource("adsabs", adsabsSource, configs["adsabs"])
		}
	}

	// 注册Google Scholar数据源 (SerpAPI兼容接口)
	if configs["google_scholar"].Enabled && configs["google_scholar"].APIKey != "" {
		googleScholarSource := NewGoogleScholarSource(configs["google_scholar"])
		sf.manager.RegisterSource("google_scholar", googleScholarSource, configs["google_scholar"])
	}

	// 注册Sci-Hub数据源
	if configs["scihub"].Enabled {
		scihubSource := NewSciHubSource(configs["scihub"])
		sf.manager.RegisterSource("scihub", scihubSource, configs["scihub"])
	}

	return sf.manager
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
