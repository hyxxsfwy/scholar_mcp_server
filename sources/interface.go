package sources

import (
	"context"
	"github.com/Seelly/scholar_mcp_server/common"
)

// PaperSource 论文数据源统一接口
type PaperSource interface {
	// GetSourceName 获取数据源名称
	GetSourceName() string

	// SearchPapers 搜索论文
	SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error)

	// GetPaper 获取特定论文
	GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error)

	// IsAvailable 检查数据源是否可用
	IsAvailable(ctx context.Context) bool

	// GetCapabilities 获取数据源能力
	GetCapabilities() SourceCapabilities
}

// SourceCapabilities 数据源能力
type SourceCapabilities struct {
	SupportsFullText     bool     // 支持全文搜索
	SupportsAuthorSearch bool     // 支持作者搜索
	SupportsDateFilter   bool     // 支持日期过滤
	SupportsCitation     bool     // 支持引用信息
	SupportsPDF          bool     // 支持PDF获取
	RequiresAPIKey       bool     // 需要API密钥
	SupportedFields      []string // 支持的搜索字段
	MaxResultsPerPage    int      // 每页最大结果数
}

// SourceConfig 数据源配置
type SourceConfig struct {
	APIKey         string
	BaseURL        string
	RequestTimeout int
	MaxRetries     int
	RateLimit      int
	Enabled        bool
}

// SourceManager 数据源管理器
type SourceManager struct {
	sources map[string]PaperSource
	configs map[string]SourceConfig
}

// NewSourceManager 创建新的数据源管理器
func NewSourceManager() *SourceManager {
	return &SourceManager{
		sources: make(map[string]PaperSource),
		configs: make(map[string]SourceConfig),
	}
}

// RegisterSource 注册数据源
func (sm *SourceManager) RegisterSource(name string, source PaperSource, config SourceConfig) {
	sm.sources[name] = source
	sm.configs[name] = config
}

// GetSource 获取数据源
func (sm *SourceManager) GetSource(name string) (PaperSource, bool) {
	source, exists := sm.sources[name]
	return source, exists
}

// GetAllSources 获取所有已注册的数据源
func (sm *SourceManager) GetAllSources() map[string]PaperSource {
	return sm.sources
}

// GetEnabledSources 获取所有启用的数据源
func (sm *SourceManager) GetEnabledSources() map[string]PaperSource {
	enabled := make(map[string]PaperSource)
	for name, source := range sm.sources {
		if config, exists := sm.configs[name]; exists && config.Enabled {
			enabled[name] = source
		}
	}
	return enabled
}
