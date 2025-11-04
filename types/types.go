package types

import "context"

// 向量存储
type VectorStoreItem struct {
	Embedding []float64              `json:"embedding"`          //文档向量表示
	Document  string                 `json:"document"`           //原始文档内容
	Metadata  map[string]interface{} `json:"metadata,omitempty"` //文档元数据
}

type ToolCall struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"` // 输入参数模式
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"toolCalls"`
}

// 嵌入
type EmbeddingRequest struct {
	Model          string `json:"model"` //嵌入模型
	Input          string `json:"input"`
	EncodingFormat string `json:"encodingFormat"` //编码格式
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"` //嵌入后向量结果
	} `json:"data"`
}

// mcpTool
type MCPToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

type VectorStore interface {
	AddEmbedding(ctx context.Context, embedding []float64, document string, metadata map[string]interface{}) error
	Search(ctx context.Context, queryEmbedding []float64, limit int) ([]string, error)
	Size() int
}

type EmbeddingRetriever interface {
	EmbedDocument(ctx context.Context, document string) ([]float64, error)
	EmbedQuery(ctx context.Context, query string) ([]float64, error)
	Retrieve(ctx context.Context, query string, limit int) ([]string, error)
}

type MCPClient interface {
	Init(ctx context.Context) error
	Close() error
	GetTools() []Tool
	CallTool(ctx context.Context, name string, params map[string]interface{}) (*MCPToolResult, error)
}

type ChatClient interface {
	Chat(ctx context.Context, prompt string) (*ChatResponse, error)
	AppendToolResult(toolCallID, toolOutput string)
	SetSystemPrompt(context string)
	SetContext(context string)
	GetMessageHistory() []ChatMessage
	ClearHistory()
}

type Agent interface {
	Init(ctx context.Context) error
	Close() error
	Invoke(ctx context.Context, prompt string) (string, error)
}
