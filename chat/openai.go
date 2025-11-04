package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"llm-mcp-rag-simple/types"
	"llm-mcp-rag-simple/utils"
	"strings"
)

type OpenaiClient struct {
	client   *openai.Client
	model    string
	messages []openai.ChatCompletionMessage // 对话历史消息列表，维护完整的对话上下文
	tools    []openai.Tool
}

func NewOpenAIClient(apiKye, baseURL, model string, tools []types.Tool, systemPrompt, context string) *OpenaiClient {
	config := openai.DefaultConfig(apiKye)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	client := &OpenaiClient{
		client:   openai.NewClientWithConfig(config),
		model:    model,
		messages: make([]openai.ChatCompletionMessage, 0),
		tools:    convertTools(tools),
	}
	//添加系统提示词
	if systemPrompt != "" {
		client.messages = append(client.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	//上下文信息
	if context != "" {
		client.messages = append(client.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: context,
		})

	}
	return client
}

// chat
func (c *OpenaiClient) Chat(ctx context.Context, prompt string) (*types.ChatResponse, error) {
	utils.LogTitle("CHAT")
	if prompt != "" {
		c.messages = append(c.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.messages,
		Stream:   true,
	}

	if len(c.tools) > 0 {
		req.Tools = c.tools
	}
	//创建流式响应流
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建流式响应失败: %w", err)
	}
	defer stream.Close()

	//初始化响应收集变量
	var content strings.Builder                   //收集完整的响应内容
	var toolCalls []types.ToolCall                // 收集工具请求
	toolCallsMap := make(map[int]*types.ToolCall) // 用于组装分片的工具调用数据

	utils.LogTitle("RESPONSE")

	//处理流式响应
	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				//流结束，正常退出
				break
			}
			return nil, fmt.Errorf("接收流式响应失败: %w", err)
		}
		if len(response.Choices) == 0 {
			continue
		}
		delta := response.Choices[0].Delta

		//处理文本内容
		if delta.Content != "" {
			content.WriteString(delta.Content)
			fmt.Print(delta.Content) //实时显示生成的内容
		}

		//处理工具调用
		if len(delta.ToolCalls) > 0 {
			for _, toolCallDelta := range delta.ToolCalls {
				index := *toolCallDelta.Index

				//新工具调用，初始化结构
				if _, exists := toolCallsMap[index]; !exists {
					toolCallsMap[index] = &types.ToolCall{
						ID: "",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{Name: "", Arguments: ""},
					}
				}

				currentCall := toolCallsMap[index]

				//积累工具调用数据,字段可能多次发出，需要拼接,比如id为call_123 ,可能切分为两次 ，call _123需要拼接
				if toolCallDelta.ID != "" {
					currentCall.ID += toolCallDelta.ID
				}
				if toolCallDelta.Function.Name != "" {
					currentCall.Function.Name += toolCallDelta.Function.Name
				}
				if toolCallDelta.Function.Arguments != "" {
					currentCall.Function.Arguments += toolCallDelta.Function.Arguments
				}
			}
		}
	}

	//将回复添加到对话历史
	assistantMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content.String(),
	}
	//有工具调用，添加到消息中
	if len(toolCalls) > 0 {
		openaiToolCalls := make([]openai.ToolCall, len(toolCalls))
		for i, tc := range toolCalls {
			openaiToolCalls[i] = openai.ToolCall{
				ID:   tc.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		assistantMsg.ToolCalls = openaiToolCalls
	}
	//更新对话历史
	c.messages = append(c.messages, assistantMsg)

	return &types.ChatResponse{
		Content:   content.String(),
		ToolCalls: toolCalls,
	}, nil
}

// 将工具执行结果添加到对话中
func (c *OpenaiClient) AppendToolResult(toolCallID, toolOutput string) {
	c.messages = append(c.messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    toolOutput,
		ToolCallID: toolCallID,
	})
}

// 设置或添加系统提示词
func (c *OpenaiClient) SetSystemPrompt(prompt string) {
	for i, msg := range c.messages {
		//删除消息历史中的系统消息，确保对话中只有一个系统提示词
		if msg.Role == openai.ChatMessageRoleSystem {
			c.messages = append(c.messages[:i], c.messages[i+1:]...)
			break
		}
	}
	//添加新的系统提示词
	if prompt != "" {
		systemMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompt,
		}
		c.messages = append([]openai.ChatCompletionMessage{systemMsg}, c.messages...)
	}
}

// 对话添加上下文
func (c *OpenaiClient) SetContext(context string) {
	if context != "" {
		c.messages = append(c.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: context,
		})
	}
}

// 返回当前消息历史
func (c *OpenaiClient) GetMessageHistory() []types.ChatMessage {
	messages := make([]types.ChatMessage, len(c.messages))
	for i, msg := range c.messages {
		messages[i] = types.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return messages
}

// 重置对话（清除对话历史，保留系统消息，确保新对话且角色设定不丢失）
func (c *OpenaiClient) ClearHistory() {
	var systemMessages []openai.ChatCompletionMessage
	//提取系统消息,只保留系统提示词
	for _, msg := range c.messages {
		if msg.Role == openai.ChatMessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		}
	}
	c.messages = systemMessages
}

// 将内部工具类型转换为OpenAI工具格式
func convertTools(tools []types.Tool) []openai.Tool {
	openaiTools := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		// JSON序列化和反序列化确保数据结构的正确转换
		parametersBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			fmt.Errorf("解析出错: %w", err)
			return nil
		}
		var parameters map[string]interface{}
		json.Unmarshal(parametersBytes, &parameters)

		openaiTools[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		}
	}
	return openaiTools
}
