# LLM-MCP-RAG Simple

一个使用 Go 构建的轻量级智能代理示例，集成三大核心能力：
- LLM 对话（基于 OpenAI 兼容接口，支持流式与工具调用）
- MCP 外部工具调用（基于官方 MCP Go SDK）
- RAG 检索增强（内存向量库 + 语义检索）

## 目录结构

```
├── agent/           # 代理核心：对话编排、RAG、工具调用
├── chat/            # OpenAI 兼容聊天客户端（流式、工具调用）
├── embedding/       # 嵌入检索：文本向量化 + 相似度搜索
├── vectorstore/     # 内存向量存储，支持余弦相似度
├── mcp/             # MCP 客户端：会话、工具发现、工具调用
├── mcpserver/      # 示例 MCP 服务器（计算器）源码
├── knowledge/       # 示例 知识库文档（自动加载并向量化）
├── config/          # 配置加载与校验（.env）
├── utils/           # 日志与辅助工具
├── types/           # 公共类型定义
├── main.go          # 命令行入口与交互会话
├── mcp_servers.json # MCP 服务配置（名称、命令、参数）
└── .env             # 环境变量配置
```

## 安装与运行

### 1. 环境要求
- Go 1.23+（工具链 `go1.24.3`）
- Windows/Linux/macOS

### 2. 配置环境变量（.env）
按需填写以下变量（与 `config/config.go` 定义严格一致）：

```
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=

EMBEDDING_BASE_URL=
EMBEDDING_KEY=
EMBEDDING_MODEL=

LOG_LEVEL=info
MAX_RETRIES=3
TIMEOUT_SECONDS=30
DEBUG=false

AGENT_NAME=LLM-MCP-RAG-SIMPLE agent
AGENT_SYSTEM_PROMPT=
AGENT_CONTEXT=你拥有强大的RAG检索能力和工具调用能力，可以帮助用户解决各种问题。
```

说明：
- `OPENAI_BASE_URL` 支持 OpenAI,DeepSeek、Qwen、等兼容接口。
- `EMBEDDING_*` 指向你选择的嵌入服务。
- 代理的系统提示词与上下文会在启动时注入到对话历史。

### 3. 安装依赖并构建

```
go mod download
go build -o llm-mcp-rag-simple .
```

### 4. 运行

```
./llm-mcp-rag-simple
```

首次启动会：
- 加载 `knowledge/` 目录下的 `.md`/`.txt` 文档并向量化 (注意:当前为加载到内存，如有需要可自行添加数据库（TODO）)
- 尝试读取 `mcp_servers.json` 并连接配置的 MCP 服务
- 启动交互式命令行：输入问题或使用内置命令

## 交互命令
- `help` 显示帮助
- `clear` 清除对话历史
- `history` 显示历史消息
- `clients` 查看可用 MCP 客户端
- `exit` 退出程序

## 配置 MCP 服务

`mcp_servers.json` 示例：

```json
[
  {
    "Name": "calculator",
    "Command": "mcp-calculator-server.exe",
    "Args": []
  }
]
```
注意:填写正确的mcp服务器路径

启动后，代理会自动：
1. 读取 JSON 并为每个服务创建 MCP 客户端
2. 连接会话并拉取工具列表
3. 在有工具调用需求时调度执行

## 计算器 MCP 服务示例

计算器服务提供基础加减乘除工具，工具入参为：

```json
{
  "name": "calculate",
  "arguments": {
    "operation": "add",
    "a": 10,
    "b": 5
  }
}
```

当 LLM 生成工具调用时，代理会解析流式分片并在会话内执行 MCP 工具，随后将结果写回对话上下文。

## 开发要点

- Agent：`agent/agent.go` 封装查询编排、RAG 检索、工具调用与重试机制
- Chat：`chat/openai.go` 支持流式输出、工具调用（OpenAI Tool）与历史管理
- Embedding：`embedding/embedding.go` 请求外部嵌入 API 并写入 `vectorstore`
- VectorStore：`vectorstore/vectorstore.go` 内存实现、余弦相似度、并发安全
- MCP：`mcp/client.go` 负责会话管理、工具发现与调用，`mcp/servers.go` 解析 JSON 配置
- Config：`config/config.go` 从 `.env` 加载并校验所有配置项
- Utils：`utils/uitls.go` 提供彩色日志与辅助方法

## 常见问题

- 启动时报 `.env` 缺失：确保 `.env` 中 `OPENAI_API_KEY`、`EMBEDDING_BASE_URL`、`EMBEDDING_KEY` 等必填项存在
- 嵌入 API 401/403：检查 `Authorization` 格式是否符合服务商要求（当前代码使用 `Bearer` 头）
- MCP 服务未发现工具：确认可执行文件路径、权限与服务是否正常启动
- 响应速度慢：可切换更快的 API 服务商、降低 `TIMEOUT_SECONDS` 或减少知识库规模

