# MCP工具集成指南

## 什么是MCP？

MCP (Model Context Protocol) 是一个开放标准，用于连接AI应用程序与外部数据源和工具。它允许大语言模型安全地访问和操作外部资源。

## MCP的优势

### 🔧 标准化接口
- 统一的工具调用协议
- 跨平台兼容性
- 易于扩展和维护

### 🛡️ 安全可控
- 权限控制机制
- 沙箱执行环境
- 审计和日志记录

### 🚀 高性能
- 异步通信
- 批量操作支持
- 连接池管理

## 项目中的MCP集成方式
本项目通过 `mcp_servers.json` 声明要连接的 MCP 服务器，程序启动时自动加载并初始化，然后列出工具并在需要时调用。

示例配置（Windows）：
```json
[
  {
    "Name": "calculator",
    "Command": ".\\mcp-calculator-server.exe",
    "Args": []
  }
]
```

运行程序后，可以使用交互命令查看客户端与工具：
- `clients`：列出所有已连接的 MCP 客户端及其工具列表
- `help`：查看可用命令

工具调用流程：
- 程序读取并连接 `mcp_servers.json` 中的每个服务器
- 客户端执行 `tools/list`，缓存工具元数据
- 当 LLM 产生工具调用时，Agent 通过 MCP 客户端执行工具，并将结果回填到对话历史

添加自定义服务器（本仓库示例）：
```bash
# 构建示例计算器 MCP 服务器（Go）
go build -o mcp-calculator-server.exe mcp-Server/calculatorMCPServer.go

# 在 mcp_servers.json 中声明该服务器
[
  {"Name":"calculator","Command":".\\mcp-calculator-server.exe","Args":[]}
]
```

## 支持的MCP工具

### 1. 文件系统工具

**功能**：
- 读取文件内容
- 写入文件
- 列出目录
- 创建/删除文件夹
- 文件搜索

**安装**：
```bash
npm install -g @modelcontextprotocol/server-filesystem
```

**配置示例**：
```go
mcpConfigs := []struct {
    name    string
    command string
    args    []string
}{
    {
        "filesystem", 
        "mcp-server-filesystem", 
        []string{"--root", "."},
    },
}
```

**使用示例**：
```
> 请帮我读取README.md文件的内容
我来为你读取README.md文件的内容...
[调用文件系统工具读取文件]
文件内容如下：...

> 请在output目录创建一个新的文档
我来为你在output目录创建文档...
[调用文件系统工具创建文件]
文档已成功创建。
```

### 2. 网络请求工具

**功能**：
- HTTP GET/POST请求
- API调用
- 网页内容获取
- JSON数据处理

**安装**：
```bash
npm install -g @modelcontextprotocol/server-fetch
```

**配置示例**：
```go
{
    "fetch", 
    "mcp-server-fetch", 
    []string{},
}
```

**使用示例**：
```
> 请帮我获取GitHub API的用户信息
我来为你获取GitHub用户信息...
[调用网络请求工具]
用户信息如下：...

> 请检查这个网站的状态
我来检查网站状态...
[调用网络请求工具]
网站状态：正常运行
```

### 3. 数据库工具

**功能**：
- SQL查询执行
- 数据库连接管理
- 表结构查询
- 数据导入导出

**安装**：
```bash
npm install -g @modelcontextprotocol/server-sqlite
```

**配置示例**：
```go
{
    "database", 
    "mcp-server-sqlite", 
    []string{"--db", "data.sqlite"},
}
```

**使用示例**：
```
> 请查询用户表中的所有数据
我来查询用户表数据...
[调用数据库工具]
查询结果：...

> 请创建一个新的产品记录
我来创建产品记录...
[调用数据库工具]
记录已成功创建。
```

### 4. 计算工具

**功能**：
- 数学计算
- 统计分析
- 数据处理
- 图表生成

**安装**：
```bash
npm install -g @modelcontextprotocol/server-math
```

**配置示例**：
```go
{
    "calculator", 
    "mcp-server-math", 
    []string{},
}
```

**使用示例**：
```
> 请计算复杂的数学表达式
我来为你计算...
[调用计算工具]
计算结果：...

> 请分析这组数据的统计特征
我来分析数据...
[调用计算工具]
统计结果：平均值、标准差、分布情况...
```

## 自定义MCP工具

### 创建简单的MCP服务器

**1. 创建Node.js项目**：
```bash
mkdir my-mcp-server
cd my-mcp-server
npm init -y
npm install @modelcontextprotocol/sdk
```

**2. 实现MCP服务器**：
```javascript
// server.js
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';

const server = new Server(
  {
    name: 'my-custom-server',
    version: '0.1.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// 定义工具
server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'get_weather',
        description: '获取天气信息',
        inputSchema: {
          type: 'object',
          properties: {
            city: {
              type: 'string',
              description: '城市名称',
            },
          },
          required: ['city'],
        },
      },
    ],
  };
});

// 实现工具调用
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'get_weather') {
    const city = args.city;
    // 这里可以调用真实的天气API
    return {
      content: [
        {
          type: 'text',
          text: `${city}的天气：晴天，温度25°C`,
        },
      ],
    };
  }
  
  throw new Error(`Unknown tool: ${name}`);
});

// 启动服务器
const transport = new StdioServerTransport();
await server.connect(transport);
```

**3. 在Go项目中配置**：
```go
{
    "weather", 
    "node", 
    []string{"server.js"},
}
```

### Go语言MCP服务器

使用官方Go SDK创建MCP服务器：

```go
package main

import (
    "context"
    "log"
    
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    // 创建服务器
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "go-tools",
        Version: "1.0.0",
    }, nil)
    
    // 添加工具
    mcp.AddTool(server, &mcp.Tool{
        Name:        "system_info",
        Description: "获取系统信息",
    }, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
        return &mcp.CallToolResult{
            Content: []mcp.Content{
                &mcp.TextContent{Text: "系统信息：Go 1.21, Linux x64"},
            },
        }, nil, nil
    })
    
    // 运行服务器
    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatal(err)
    }
}
```

## 工具配置管理

### 在main.go中配置MCP工具

编辑 `cmd/main.go` 文件中的 `initializeMCPClients` 函数：

```go
func initializeMCPClients(ctx context.Context, agent *agent.Agent) error {
    // MCP客户端配置
    mcpConfigs := []struct {
        name    string
        command string
        args    []string
    }{
        // 文件系统工具
        {"filesystem", "mcp-server-filesystem", []string{"--root", "."}},
        
        // 网络请求工具
        {"fetch", "mcp-server-fetch", []string{}},
        
        // 数据库工具
        {"database", "mcp-server-sqlite", []string{"--db", "data.sqlite"}},
        
        // 自定义工具
        {"weather", "node", []string{"weather-server.js"}},
    }
    
    // 初始化客户端...
}
```

### 环境变量配置

在 `.env` 文件中添加MCP相关配置：

```env
# MCP工具配置
MCP_FILESYSTEM_ROOT=.
MCP_DATABASE_PATH=data.sqlite
MCP_ENABLE_FETCH=true
MCP_ENABLE_CALCULATOR=true
```

## 使用示例

### 文件操作示例

```
用户：请帮我创建一个项目计划文档

助手：我来为你创建项目计划文档。

[调用文件系统工具]
- 创建 project-plan.md 文件
- 写入项目计划模板内容
- 设置适当的文件权限

项目计划文档已成功创建在 project-plan.md 文件中。文档包含了项目概述、时间线、里程碑等关键信息。
```

### 数据查询示例

```
用户：请查询最近一周的用户注册数据

助手：我来查询最近一周的用户注册数据。

[调用数据库工具]
- 连接到用户数据库
- 执行SQL查询：SELECT COUNT(*) FROM users WHERE created_at >= DATE('now', '-7 days')
- 分析查询结果

查询结果显示：最近一周共有156名新用户注册，比上周增长了23%。注册高峰出现在周二和周四。
```

### 网络请求示例

```
用户：请帮我检查GitHub上这个项目的最新版本

助手：我来检查GitHub项目的最新版本信息。

[调用网络请求工具]
- 发送GET请求到GitHub API
- 解析JSON响应数据
- 提取版本信息

项目最新版本是 v2.1.0，发布于2024年1月15日。更新内容包括性能优化、bug修复和新功能添加。
```

## 最佳实践

### 1. 安全考虑
- 限制工具访问权限
- 验证输入参数
- 记录工具调用日志
- 实现错误处理

### 2. 性能优化
- 使用连接池
- 实现缓存机制
- 异步处理长时间操作
- 设置合理的超时时间

### 3. 错误处理
- 提供清晰的错误信息
- 实现重试机制
- 记录详细的错误日志
- 优雅降级处理

### 4. 监控和调试
- 启用详细日志
- 监控工具调用频率
- 分析性能指标
- 设置告警机制

## 故障排除

### 常见问题

**1. MCP服务器启动失败**
```
Error: failed to start MCP server
```
解决方案：
- 检查MCP服务器是否正确安装
- 验证命令路径和参数
- 查看服务器日志

**2. 工具调用超时**
```
Error: tool call timeout
```
解决方案：
- 增加超时时间设置
- 检查网络连接
- 优化工具实现

**3. 权限错误**
```
Error: permission denied
```
解决方案：
- 检查文件/目录权限
- 确认用户权限设置
- 使用适当的访问控制

### 调试技巧

**启用MCP调试**：
```env
DEBUG=true
MCP_DEBUG=true
LOG_LEVEL=debug
```

**查看MCP日志**：
```bash
# 查看MCP服务器输出
tail -f mcp-server.log

# 查看应用程序日志
tail -f app.log
```

**测试MCP连接**：
```bash
# 手动测试MCP服务器
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | mcp-server-filesystem
```