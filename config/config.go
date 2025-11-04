package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

type Config struct {
	OpenAI    OpenAIConfig    `json:"openai"`
	Embedding EmbeddingConfig `json:"embedding"`
	App       AppConfig       `json:"app"`
	Agent     AgentConfig     `json:"agent"`
}

type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

type EmbeddingConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}
type AppConfig struct {
	LogLevel   string        `json:"log_level"`
	MaxRetries int           `json:"max_retries"`
	Timeout    time.Duration `json:"timeout"`
	Debug      bool          `json:"debug"`
}

type AgentConfig struct {
	Name         string `json:"name"`
	SystemPrompt string `json:"systemPrompt"`
	Context      string `json:"context"`
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("加载环境变量失败：%v\n", err)
		return nil, err
	}
	config := &Config{
		OpenAI: OpenAIConfig{
			APIKey:  getEnvString("OPENAI_API_KEY"),
			BaseURL: getEnvString("OPENAI_BASE_URL"),
			Model:   getEnvString("OPENAI_MODEL"),
		},
		Embedding: EmbeddingConfig{
			BaseURL: getEnvString("EMBEDDING_BASE_URL"),
			APIKey:  getEnvString("EMBEDDING_KEY"),
			Model:   getEnvString("EMBEDDING_MODEL"),
		},
		App: AppConfig{
			LogLevel:   getEnvString("LOG_LEVEL"),
			MaxRetries: getEnvInt("MAX_RETRIES", 3),
			Timeout:    time.Duration(getEnvInt("TIMEOUT_SECONDS", 30)) * time.Second,
			Debug:      getEnvBool("DEBUG", false),
		},
		Agent: AgentConfig{
			Name:         getEnvString("AGENT_NAME"),
			SystemPrompt: defaultSystemPrompt("AGENT_SYSTEM_PROMPT"),
			Context:      getEnvString("AGENT_CONTEXT"),
		},
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败：%w", err)
	}
	return config, nil
}

func (c *Config) Validate() error {
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY 不能为空")
	}
	if c.Embedding.BaseURL == "" {
		return fmt.Errorf("EMBEDDING_BASE_URL 不能为空 ")
	}

	if c.Embedding.APIKey == "" {
		return fmt.Errorf("EMBEDDING_KEY 不能为空")
	}

	if c.App.MaxRetries <= 0 {
		return fmt.Errorf("MAX_RETRIES 必需大于0")
	}

	if c.App.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT_SECONDS 必需大于0")
	}

	validLogLevels := []string{"debug", "info", "warning", "error"}
	if !contains(validLogLevels, c.App.LogLevel) {
		return fmt.Errorf("无效的日志级别：%s", validLogLevels)
	}
	return nil
}

func defaultSystemPrompt(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return getDefaultSystemPrompt()
}

func getEnvString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Errorf("环境变量%s不存在", key)
		return ""
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getDefaultSystemPrompt() string {
	return `You are an intelligent assistant with access to various tools and a knowledge base. 
You can help users by:
1. Answering questions using your knowledge base through RAG (Retrieval-Augmented Generation)
2. Using available tools to perform specific tasks
3. Providing accurate and helpful responses

When using tools:
- Always explain what you're doing and why
- Handle errors gracefully and inform the user
- Use the most appropriate tool for each task

When using the knowledge base:
- Cite relevant information from the retrieved documents
- Be clear about the source of your information
- If no relevant information is found, say so clearly

Always be helpful, accurate, and transparent in your responses.`
}
