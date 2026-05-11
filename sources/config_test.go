package sources

import (
	"strings"
	"testing"
)

func TestParseProjectConfigJSONC(t *testing.T) {
	config, err := ParseProjectConfigJSONC([]byte(`{
  // line comment
  "server": {
    "http": true,
    "port": "9900",
  },
  "sources": {
    "google_scholar": {
      "enabled": true,
      "api_key": "from-config",
      "base_url": "https://serpapi.example.com/search.json", // keep URL scheme
    },
    "broker_research": {
      "enabled": true,
      "feeds": [
        {
          "name": "公开金工Feed",
          "url": "https://research.example.com/rss.xml",
        },
      ],
    },
  },
}`))
	if err != nil {
		t.Fatalf("ParseProjectConfigJSONC failed: %v", err)
	}
	if !config.Server.HTTPDefault(false) || config.Server.PortDefault("8899") != "9900" {
		t.Fatalf("expected server config to be parsed, got %+v", config.Server)
	}

	google := config.Sources["google_scholar"]
	if google.APIKey == nil || *google.APIKey != "from-config" {
		t.Fatalf("expected api key from config, got %+v", google)
	}
	if google.BaseURL == nil || !strings.HasPrefix(*google.BaseURL, "https://") {
		t.Fatalf("expected URL string to survive comment stripping, got %+v", google.BaseURL)
	}

	brokerConfig, err := config.Sources["broker_research"].applyWithError(SourceConfig{})
	if err != nil {
		t.Fatalf("apply broker config failed: %v", err)
	}
	if !strings.Contains(brokerConfig.BaseURL, "research.example.com") {
		t.Fatalf("expected feeds array to be compacted into BaseURL, got %q", brokerConfig.BaseURL)
	}
}

func TestProjectConfigOverridesEnvConfig(t *testing.T) {
	t.Setenv("ENABLE_GOOGLE_SCHOLAR", "false")
	t.Setenv("SERPAPI_API_KEY", "from-env")
	t.Setenv("SERPAPI_BASE_URL", "https://env.example.com/search.json")
	t.Setenv("REQUEST_TIMEOUT", "15")

	factory := NewSourceFactory()
	configs := factory.loadConfigsFromEnv()

	config, err := ParseProjectConfigJSONC([]byte(`{
  "request": {"timeout": 45},
  "sources": {
    "google_scholar": {
      "enabled": true,
      "api_key": "from-config",
      "base_url": "https://config.example.com/search.json",
      "rate_limit": 7
    }
  }
}`))
	if err != nil {
		t.Fatalf("ParseProjectConfigJSONC failed: %v", err)
	}
	if err := config.ApplyToSourceConfigs(configs); err != nil {
		t.Fatalf("ApplyToSourceConfigs failed: %v", err)
	}

	google := configs["google_scholar"]
	if !google.Enabled {
		t.Fatalf("expected config to override env enabled=false")
	}
	if google.APIKey != "from-config" || google.BaseURL != "https://config.example.com/search.json" {
		t.Fatalf("expected config credentials to override env, got %+v", google)
	}
	if google.RequestTimeout != 45 || google.RateLimit != 7 {
		t.Fatalf("expected request defaults and source rate limit from config, got %+v", google)
	}
}

func TestProjectConfigCanLeaveSecretsOnEnv(t *testing.T) {
	t.Setenv("ENABLE_SCOPUS", "true")
	t.Setenv("SCOPUS_API_KEY", "from-env")

	factory := NewSourceFactory()
	configs := factory.loadConfigsFromEnv()

	config, err := ParseProjectConfigJSONC([]byte(`{
  "sources": {
    "scopus": {
      "enabled": true,
      "rate_limit": 4
    }
  }
}`))
	if err != nil {
		t.Fatalf("ParseProjectConfigJSONC failed: %v", err)
	}
	if err := config.ApplyToSourceConfigs(configs); err != nil {
		t.Fatalf("ApplyToSourceConfigs failed: %v", err)
	}

	scopus := configs["scopus"]
	if scopus.APIKey != "from-env" {
		t.Fatalf("expected absent api_key in config to keep env value, got %+v", scopus)
	}
	if scopus.RateLimit != 4 {
		t.Fatalf("expected source config to override rate limit, got %+v", scopus)
	}
}
