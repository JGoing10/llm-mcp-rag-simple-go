package mcp

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"llm-mcp-rag-simple/types"
	"llm-mcp-rag-simple/utils"
	"os/exec"
	"sync"
)

type Client struct {
	name    string   //客户端名称
	version string   //版本号
	command string   //mcp服务可执行文件路径
	args    []string //mcp服务的命令行参数

	client    *mcp.Client
	session   *mcp.ClientSession    //客户端会话
	transport *mcp.CommandTransport //命令传输层

	//状态管理
	tools  []types.Tool
	mu     sync.RWMutex
	closed bool //客户端是否关闭
}

func NewClient(name, command string, args []string, version string) *Client {
	if version == "" {
		version = "1.0.0"
	}
	return &Client{
		name:    name,
		version: version,
		command: command,
		args:    args,
		tools:   make([]types.Tool, 0),
	}
}

// 初始化并连接到服务
func (c *Client) Init(ctx context.Context) error {
	utils.LogInfo(fmt.Sprintf("初始化mcp client %s", c.name))
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return fmt.Errorf("client 已经初始化了")
	}
	//创建mcp客户端实例
	impl := &mcp.Implementation{
		Name:    c.name,
		Version: c.version,
	}
	c.client = mcp.NewClient(impl, nil)
	//命令传输层
	// CommandTransport负责启动外部MCP服务器进程并管理通信
	cmd := exec.CommandContext(ctx, c.command, c.args...)
	c.transport = &mcp.CommandTransport{
		Command: cmd,
	}
	//建立连接并创建会话
	session, err := c.client.Connect(ctx, c.transport, nil)
	if err != nil {
		return fmt.Errorf("连接mcp服务错误,%w", err)
	}
	c.session = session
	//获取工具列表
	if err := c.ListTools(ctx); err != nil {
		return fmt.Errorf("获取工具列表失败:%w", err)
	}

	utils.LogInfo(fmt.Sprintf("mcp client 发现%d个工具，工具列表:%v", len(c.tools), c.getToolNames()))
	return nil
}

// 关闭mcp客户端，并清理资源
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	if c.session != nil {
		c.session.Close()
		c.session = nil
	}
	//清理客户端引用
	c.client = nil
	c.transport = nil
	utils.LogInfo(fmt.Sprintf("mcp client [%s],关闭", c.name))

	return nil
}

// 返回当前可用工具列表
func (c *Client) GetTools() []types.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tools := make([]types.Tool, len(c.tools))
	copy(tools, c.tools)
	return tools
}

// 调用指定mcp工具并返回结果
func (c *Client) CallTool(ctx context.Context, name string, params map[string]interface{}) (*types.MCPToolResult, error) {
	c.mu.RLock()
	// 检查客户端是否已关闭或未初始化
	if c.closed || c.session == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("mcp client 不可用")
	}
	session := c.session
	c.mu.RUnlock()
	callParams := &mcp.CallToolParams{
		Name:      name,
		Arguments: params,
	}

	//通过会话调用工具
	result, err := session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("调用工具%s错误：%w", name, err)
	}

	toolResult := c.convertCallToolResult(result)
	return toolResult, nil

}

// 从mcp服务获取工具列表
func (c *Client) ListTools(ctx context.Context) error {

	result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("获取list tool 错误: %w", err)
	}
	tools := make([]types.Tool, len(result.Tools))
	for i, tool := range result.Tools {
		tools[i] = types.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: c.convertInputSchema(tool.InputSchema),
		}
	}
	//更新工具列表
	c.tools = tools

	return nil
}

// 转换CallToolResult格式
func (c *Client) convertCallToolResult(result *mcp.CallToolResult) *types.MCPToolResult {
	toolResult := &types.MCPToolResult{
		Content: make([]struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}, 0),
		IsError: result.IsError,
	}
	//转换内容数组
	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			toolResult.Content = append(toolResult.Content, struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{Type: "text", Text: c.Text})
		case *mcp.ImageContent:
			toolResult.Content = append(toolResult.Content, struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{Type: "text", Text: fmt.Sprintf("[Image:%s]", c.Data)})
		case *mcp.AudioContent:
			toolResult.Content = append(toolResult.Content, struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{Type: "text", Text: fmt.Sprintf("[Audio:%s]", c.Data)})
		default:
			//未知类型
			toolResult.Content = append(toolResult.Content, struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{Type: "text", Text: fmt.Sprintf("%v", content)})
		}
	}
	//没有内容则添加一个空文本内容
	if len(toolResult.Content) == 0 {
		toolResult.Content = append(toolResult.Content, struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{Type: "text", Text: ""})
	}
	return toolResult
}

// 转换工具输入模式
func (c *Client) convertInputSchema(schema any) map[string]interface{} {
	if schema == nil {
		return make(map[string]interface{})
	}
	//如果已经是map，返回
	if schemaMap, ok := schema.(map[string]interface{}); ok {
		return schemaMap
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// 返回工具名称列表
func (c *Client) getToolNames() []string {
	names := make([]string, len(c.tools))
	for i, tool := range c.tools {
		names[i] = tool.Name
	}
	return names
}
