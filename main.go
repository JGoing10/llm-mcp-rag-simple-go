package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"llm-mcp-rag-simple/agent"
	"llm-mcp-rag-simple/chat"
	"llm-mcp-rag-simple/config"
	"llm-mcp-rag-simple/embedding"
	mcpClient "llm-mcp-rag-simple/mcp"
	"llm-mcp-rag-simple/types"
	"llm-mcp-rag-simple/utils"
	"llm-mcp-rag-simple/vectorstore"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("加载配置失败：%v\n", err)
		os.Exit(1)
	}
	utils.LogTitle("LLM-MCP-RAG Agent Starting...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	vectorStore := vectorstore.NewInMemoryVectorStore()
	embeddingRetriever := embedding.NewRetriever(cfg.Embedding.Model, cfg.Embedding.BaseURL, cfg.Embedding.APIKey, vectorStore)

	chatClient := chat.NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.BaseURL, cfg.OpenAI.Model, []types.Tool{}, "", "")

	//创建agent实例
	agentConfig := agent.AgentConfig{
		Name:         cfg.Agent.Name,
		SystemPrompt: cfg.Agent.SystemPrompt,
		Context:      cfg.Agent.Context,
		MaxRetries:   cfg.App.MaxRetries,
		Timeout:      cfg.App.Timeout,
	}
	agentInstance := agent.NewAgent(agentConfig, chatClient, embeddingRetriever, vectorStore)

	//加载知识库
	if err := loadKnowledgeBase(ctx, agentInstance); err != nil {
		utils.LogWarn(fmt.Sprintf("加载知识库失败：%v", err))
	}
	//初始化mcp客户端
	if err := initializeMCPClients(ctx, agentInstance); err != nil {
		utils.LogWarn(fmt.Sprintf("初始化mcp客户端失败：%v", err))
	}

	go func() {
		if err := runInteractiveSession(ctx, agentInstance); err != nil {
			utils.LogError(fmt.Sprintf("运行交互会话失败：%v", err))
			cancel()
		}
	}()
	utils.LogInfo("Agent启动成功! 'help' 查看可用命令")
	<-sigChan
	utils.LogInfo("正在关闭Agent...")
	//清理操作
	if err := agentInstance.Close(); err != nil {
		utils.LogError(fmt.Sprintf("清理过程中出现错误：%v", err))
	}
	utils.LogInfo("GoodBye!")
}

// 从知识目录加载文档到向量数据库
func loadKnowledgeBase(ctx context.Context, agent *agent.Agent) error {
	knowledgeDir := "knowledge"

	if _, err := os.Stat(knowledgeDir); os.IsNotExist(err) {
		utils.LogInfo(fmt.Sprintf("知识库目录不存在：%s", knowledgeDir))
		return nil
	}
	var documents []string
	//遍历目录，收集所有支持的文档文件
	err := filepath.Walk(knowledgeDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//处理.md,.txt文件
		if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".txt")) {
			content, err := os.ReadFile(path)
			if err != nil {
				utils.LogWarn(fmt.Sprintf("读取文件%s失败：%v", path, err))
				return nil
			}
			documents = append(documents, string(content))
			utils.LogDebug(fmt.Sprintf("已加载文档：%s", path))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历知识库目录失败：%w", err)
	}

	//将文档添加到知识库中
	if len(documents) > 0 {
		if err := agent.AddKnowledge(ctx, documents); err != nil {
			return fmt.Errorf("添加文档到知识库失败：%w", err)
		}
		utils.LogInfo(fmt.Sprintf("成功将%d个文档添加到知识库", len(documents)))
	}
	return nil
}

// 初始mcp客户端
func initializeMCPClients(ctx context.Context, agent *agent.Agent) error {
	//从配置文件中加载mcp server配置
	serverConfigs := []mcpClient.ServerConfig{}
	const defaultJSON = "mcp_servers.json"
	if _, err := os.Stat(defaultJSON); err == nil {
		// 文件存在，尝试加载配置
		if configs, err := mcpClient.LoadServerConfigFromJSON(defaultJSON); err != nil {
			utils.LogWarn(fmt.Sprintf("加载mcp server配置%s失败：%v", defaultJSON, err))
		} else {
			utils.LogInfo(fmt.Sprintf("成功加载mcp server配置%s", defaultJSON))
			serverConfigs = configs
		}
	} else {
		// 文件不存在
		utils.LogWarn(fmt.Sprintf("默认mcp server配置文件%s不存在", defaultJSON))
	}

	for _, mcpCfg := range serverConfigs {
		client := mcpClient.NewClient(mcpCfg.Name, mcpCfg.Command, mcpCfg.Args, "1.0.0")

		if err := agent.AddMCPClient(mcpCfg.Name, client); err != nil {
			utils.LogWarn(fmt.Sprintf("添加mcp client【%s】失败：%v", mcpCfg.Name, err))
			client.Close()
			continue
		}
		utils.LogInfo(fmt.Sprintf("成功添加mcp client【%s】", mcpCfg.Name))
	}
	return nil
}

// 运行交互聊天会话
func runInteractiveSession(ctx context.Context, agent *agent.Agent) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Printf("\n>")
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("读取用户输入失败：%w", err)
				}
				return nil
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			//处理特殊命令
			if handled := handleSpecialCommands(input, agent); handled {
				continue
			}

			//用户查询
			response, err := agent.Query(ctx, input)
			if err != nil {
				utils.LogError(fmt.Sprintf("查询失败：%v", err))
				continue
			}
			fmt.Printf("Assistant:\n %s\n", response.Content)
		}

	}
}

// 自定义对话交互命令
func handleSpecialCommands(input string, agent *agent.Agent) bool {
	switch strings.ToLower(input) {
	case "help":
		printHelp()
		return true
	case "clear":
		agent.ClearHistory()
		fmt.Println("历史消息已清除")
		return true
	case "history":
		printHistory(agent)
		return true
	case "clients":
		printMCPClients(agent)
		return true
	case "exit", "quit":
		fmt.Println("GoodBye")
		os.Exit(0)
		return true
	default:
		return false
	}

}

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  help     - Show this help message")
	fmt.Println("  clear    - Clear chat history")
	fmt.Println("  history  - Show chat history")
	fmt.Println("  clients  - Show available MCP clients")
	fmt.Println("  exit     - Exit the application")
	fmt.Println("\nOr just type your question to chat with the agent.")
}
func printHistory(agent *agent.Agent) {
	history := agent.GetMessageHistory()
	if len(history) == 0 {
		fmt.Println("没有历史消息 ")
		return
	}
	fmt.Println("\n 历史消息:")
	for i, msg := range history {
		fmt.Printf("%d. %s: %s\n", i+1, msg.Role, msg.Content)
	}
}
func printMCPClients(agent *agent.Agent) {
	clients := agent.GetMCPClient()
	if len(clients) == 0 {
		fmt.Println("没有可用的MCP客户端")
		return
	}
	fmt.Println("\n可用的MCP客户端:")
	for i, client := range clients {
		fmt.Printf("%d,%s \n", i+1, client)
	}
}
