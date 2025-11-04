package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"llm-mcp-rag-simple/types"
	"llm-mcp-rag-simple/utils"
	"strings"
	"sync"
	"time"
)

type Agent struct {
	name            string
	chatClient      types.ChatClient
	embeddingClient types.EmbeddingRetriever
	vectorStore     types.VectorStore
	mcpClients      map[string]types.MCPClient
	systemPrompt    string
	context         string
	maxRetries      int //最大重试数
	timeout         time.Duration
	mu              sync.RWMutex
}

type AgentConfig struct {
	Name         string
	SystemPrompt string
	Context      string
	MaxRetries   int
	Timeout      time.Duration
}

func NewAgent(config AgentConfig, chatClient types.ChatClient, embeddingClient types.EmbeddingRetriever, vectorStore types.VectorStore) *Agent {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Name == "" {
		config.Name = "LLM-MCP-RAG Agent"
	}

	agent := &Agent{
		name:            config.Name,
		chatClient:      chatClient,
		embeddingClient: embeddingClient,
		vectorStore:     vectorStore,
		mcpClients:      make(map[string]types.MCPClient),
		systemPrompt:    config.SystemPrompt,
		context:         config.Context,
		maxRetries:      config.MaxRetries,
		timeout:         config.Timeout,
	}

	//设置系统提示词和上下文
	if config.SystemPrompt != "" {
		chatClient.SetSystemPrompt(config.SystemPrompt)
	}
	if config.Context != "" {
		chatClient.SetContext(config.Context)
	}
	return agent
}

// 添加mcp客户端
func (a *Agent) AddMCPClient(name string, client types.MCPClient) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	//避免重复添加
	if _, exists := a.mcpClients[name]; exists {
		return fmt.Errorf("该mcp Client：【%s】 已存在\n", name)
	}
	//初始化
	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		return fmt.Errorf("初始化mcp client 【%s】失败：%w \n", name, err)
	}

	a.mcpClients[name] = client
	utils.LogInfo(fmt.Sprintf("已添加MCP client【%s】\n", name))
	return nil
}

// 移除一个mcp client
func (a *Agent) RemoveMCPClient(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	client, exists := a.mcpClients[name]
	if !exists {
		return fmt.Errorf("mcp client【%s】不存在\n", name)
	}
	if err := client.Close(); err != nil {
		utils.LogWarn(fmt.Sprintf("关闭mcp client【%s】失败：%v", name, err))
	}
	delete(a.mcpClients, name)
	utils.LogInfo(fmt.Sprintf("删除MCP Client 【%s】成功 \n", name))
	return nil
}

// 返回所有可用MCP列表
func (a *Agent) GetMCPClient() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, len(a.mcpClients))
	for name := range a.mcpClients {
		names = append(names, name)
	}
	return names
}

// 向知识库添加文档
func (a *Agent) AddKnowledge(ctx context.Context, documents []string) error {
	if len(documents) == 0 {
		return fmt.Errorf("没有要添加的文档")
	}

	utils.LogInfo(fmt.Sprintf("正在将 %d 个文档添加到知识库\n", len(documents)))

	type result struct {
		index int
		err   error
	}

	resultChan := make(chan result, len(documents))
	//控制并发数
	semaphore := make(chan struct{}, 5)

	for i, doc := range documents {
		go func(index int, document string) {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()
			_, err := a.embeddingClient.EmbedDocument(ctx, document)
			resultChan <- result{
				index: index,
				err:   err,
			}
		}(i, doc)

	}
	var errors []string
	for i := 0; i < len(documents); i++ {
		res := <-resultChan
		if res.err != nil {
			errors = append(errors, fmt.Sprintf("文档%d嵌入失败：%v", res.index, res.err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("添加文档失败：%s", strings.Join(errors, ";"))
	}

	utils.LogInfo(fmt.Sprintf("成功将 %d 个文档添加到知识库\n", len(documents)))
	return nil
}

// 用户查询处理
func (a *Agent) Query(ctx context.Context, query string) (*types.ChatResponse, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("查询内容不能为空")
	}

	utils.LogInfo(fmt.Sprintf("正在查询:%s\n", query))
	queryCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	//检索相关文档（RAG）
	relevantDocs, err := a.retrieveRelevantDocuments(queryCtx, query)
	if err != nil {
		utils.LogWarn(fmt.Sprintf("检索相关文档失败：%v", err))
		//检索失败继续处理，只是没有RAG增强
	}
	//增强查询，相关文档作为上下文
	enhancedQuery := a.buildEnhancedQuery(query, relevantDocs)
	//获取所有可用工具
	tools := a.getAllTools()

	//重试查询
	var response *types.ChatResponse
	var lastErr error

	for attempt := 1; attempt <= a.maxRetries; attempt++ {
		response, lastErr = a.processQueryWithTools(queryCtx, enhancedQuery, tools)
		if lastErr == nil {
			break
		}
		utils.LogWarn(fmt.Sprintf("查询尝试%d失败:%v \n", attempt, lastErr))
		if attempt < a.maxRetries {
			//时间间隔attempt秒，避免频繁重试
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("经过%d次尝试后查询失败:%w", a.maxRetries, lastErr)
	}

	utils.LogInfo(fmt.Sprintf("处理请求成功"))
	return response, nil
}

// 返回聊天消失历史
func (a *Agent) GetMessageHistory() []types.ChatMessage {
	return a.chatClient.GetMessageHistory()
}

// 清除对话历史
func (a *Agent) ClearHistory() {
	a.chatClient.ClearHistory()
	utils.LogInfo(fmt.Sprintf("对话历史已清除"))
}

// 更新系统提示词
func (a *Agent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = prompt
	a.chatClient.SetSystemPrompt(prompt)

	utils.LogInfo(fmt.Sprintf("系统提示词已更新"))
}

// 更新上下文信息
func (a *Agent) SetContext(context string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.context = context
	a.chatClient.SetContext(context)

	utils.LogInfo(fmt.Sprintf("上下文信息已更新"))
}

// 关闭所有mcp客户端并清理资源
func (a *Agent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var errors []string
	for name, client := range a.mcpClients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("关闭mcp client【%s】失败:%v", name, err))
		}
	}
	//清空客户端映射
	a.mcpClients = make(map[string]types.MCPClient)
	if len(errors) > 0 {
		return fmt.Errorf("关闭mcp client失败：%s", strings.Join(errors, ";"))
	}
	utils.LogInfo(fmt.Sprintf("所有mcp client已关闭"))
	return nil
}

// 查询检索相关文档
func (a *Agent) retrieveRelevantDocuments(ctx context.Context, query string) ([]string, error) {
	if a.vectorStore.Size() == 0 {
		return nil, fmt.Errorf("知识库中没有文档")
	}
	docs, err := a.embeddingClient.Retrieve(ctx, query, 5)
	if err != nil {
		return nil, fmt.Errorf("未能检索到相似的文档：%w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("没有找到相关文档")
	}

	utils.LogDebug(fmt.Sprintf("检索到%d份相关文档\n", len(docs)))
	return docs, nil
}

// 构建包含RAG上下文的增强查询(将检索相关文档上下文添加到查询)
func (a *Agent) buildEnhancedQuery(query string, relevantDocs []string) string {
	if len(relevantDocs) == 0 {
		return query
	}
	var builder strings.Builder
	builder.WriteString("根据以下相关信息:\n\n")

	for i, doc := range relevantDocs {
		builder.WriteString(fmt.Sprintf("文档%d: %s\n\n", i+1, doc))
	}
	builder.WriteString(fmt.Sprintf("请回答以下问题：%s", query))
	return builder.String()
}

// 获取所有可用mcp工具
func (a *Agent) getAllTools() []types.Tool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var allTools []types.Tool
	for clientName, client := range a.mcpClients {
		tools := client.GetTools()
		for _, tool := range tools {
			tool.Name = fmt.Sprintf("%S.%S", clientName, tool.Name)
			allTools = append(allTools, tool)
		}
	}
	return allTools
}

// 处理查询并执行工具调用
func (a *Agent) processQueryWithTools(ctx context.Context, query string, tools []types.Tool) (*types.ChatResponse, error) {
	response, err := a.chatClient.Chat(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("获取对话响应失败：%w", err)
	}
	//检查是否请求调用工具
	if len(response.ToolCalls) > 0 {
		utils.LogDebug(fmt.Sprintf("调用%d个工具\n", len(response.ToolCalls)))
		for _, toolCall := range response.ToolCalls {
			result, err := a.executeToolCall(ctx, toolCall)
			if err != nil {
				utils.LogError(fmt.Sprintf("调用工具失败：%v\n", err))
				//错误信息，让模型根据错误调整
				result = &types.MCPToolResult{
					Content: []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						{
							Type: "text",
							Text: fmt.Sprintf("调用工具失败：%v", err),
						},
					},
				}
			}

			//将执行结果添加到对话历史中
			var resultText string
			if result != nil && len(result.Content) > 0 {
				resultText = result.Content[0].Text
			} else {
				resultText = "没有结果"
			}
			a.chatClient.AppendToolResult(toolCall.ID, resultText)
		}
		//最终回复
		finalResponse, err := a.chatClient.Chat(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("获取工具调用最终回复失败：%w", err)
		}
		return finalResponse, nil
	}
	return response, nil
}

// 执行单个工具调用
func (a *Agent) executeToolCall(ctx context.Context, toolCall types.ToolCall) (*types.MCPToolResult, error) {
	parts := strings.SplitN(toolCall.Function.Name, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("工具名称格式错误：%s", toolCall.Function.Name)
	}
	clientName := parts[0]
	toolName := parts[1]

	a.mu.RLock()
	client, exists := a.mcpClients[clientName]
	a.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("mcp client不存在：%s", clientName)
	}

	utils.LogDebug(fmt.Sprintf("执行工具：%s.%s", clientName, toolName))
	var arguments map[string]interface{}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
		return nil, fmt.Errorf("解析工具参数失败：%w", err)
	}

	//执行工具
	result, err := client.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("执行工具%s失败：%w", toolCall.Function.Name, err)
	}
	utils.LogDebug(fmt.Sprintf("工具%s执行成功\n", toolCall.Function.Name))
	return result, nil
}
