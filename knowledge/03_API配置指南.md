# API配置指南

## 概述

本项目支持多种大语言模型API服务商，通过配置不同的 `OPENAI_BASE_URL` 和 `OPENAI_API_KEY` 即可切换使用。

### 环境变量清单（与代码一致）
- `OPENAI_API_KEY`（必填）：主模型的 API 密钥
- `OPENAI_BASE_URL`（可选）：主模型的 API 基础地址
- `OPENAI_MODEL`（可选）：主模型名称
- `EMBEDDING_BASE_URL`（必填）：嵌入模型的 API 基础地址
- `EMBEDDING_KEY`（必填）：嵌入模型的 API 密钥
- `EMBEDDING_MODEL`（可选）：嵌入模型名称
- `LOG_LEVEL`（必填）：日志级别，可选 `debug|info|warning|error`
- `MAX_RETRIES`（必填）：最大重试次数（>0）
- `TIMEOUT_SECONDS`（必填）：请求超时时间（>0，秒）
- `DEBUG`（必填）：是否启用调试模式（`true|false`）
- `AGENT_NAME`（可选）：Agent 名称
- `AGENT_SYSTEM_PROMPT`（可选）：Agent 的系统提示词（为空时使用内置默认）
- `AGENT_CONTEXT`（可选）：全局上下文描述

## 支持的API服务商

### 1. DeepSeek API (推荐)

**优势**：
- 性价比极高，价格是GPT-4的1/10
- 支持中文，理解能力强
- API稳定，响应速度快
- 支持长上下文 (32K tokens)

**配置**：
```env
OPENAI_API_KEY=sk-your-deepseek-api-key
OPENAI_BASE_URL=https://api.deepseek.com/v1
OPENAI_MODEL=deepseek-chat
```

**获取密钥**：
1. 访问 https://platform.deepseek.com/
2. 注册账号并完成实名认证
3. 在API密钥页面创建新密钥
4. 充值账户（支持支付宝/微信）

**价格**：约 ¥0.001/1K tokens（输入），¥0.002/1K tokens（输出）

### 2. 通义千问 API

**优势**：
- 阿里云官方服务，稳定可靠
- 中文能力强，适合国内用户
- 支持多种模型规格
- 有免费额度

**配置**：
```env
OPENAI_API_KEY=your-qwen-api-key
OPENAI_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
OPENAI_MODEL=qwen-turbo
```

**可用模型**：
- `qwen-turbo` - 快速模型
- `qwen-plus` - 平衡模型
- `qwen-max` - 最强模型

**获取密钥**：
1. 访问 https://dashscope.aliyuncs.com/
2. 登录阿里云账号
3. 开通灵积服务
4. 创建API-KEY

### 3. 智谱AI GLM API

**优势**：
- 清华大学技术，中文能力优秀
- 支持多模态（文本、图像）
- 价格合理
- 支持函数调用

**配置**：
```env
OPENAI_API_KEY=your-zhipu-api-key
OPENAI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
OPENAI_MODEL=glm-4
```

**可用模型**：
- `glm-4` - 最新版本
- `glm-4-flash` - 快速版本
- `glm-3-turbo` - 经济版本

**获取密钥**：
1. 访问 https://open.bigmodel.cn/
2. 注册并登录账号
3. 在API密钥页面创建密钥
4. 充值账户

### 4. 月之暗面 Kimi API

**优势**：
- 超长上下文支持 (200K+ tokens)
- 中文理解能力强
- 适合处理长文档
- 支持联网搜索

**配置**：
```env
OPENAI_API_KEY=your-moonshot-api-key
OPENAI_BASE_URL=https://api.moonshot.cn/v1
OPENAI_MODEL=moonshot-v1-8k
```

**可用模型**：
- `moonshot-v1-8k` - 8K上下文
- `moonshot-v1-32k` - 32K上下文
- `moonshot-v1-128k` - 128K上下文

**获取密钥**：
1. 访问 https://platform.moonshot.cn/
2. 注册并登录账号
3. 在控制台创建API密钥
4. 充值账户

### 5. OpenAI API (原版)

**优势**：
- 最强的模型能力
- 生态最完善
- 支持最新功能
- 文档最详细

**配置**：
```env
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4
```

**可用模型**：
- `gpt-4` - 最强模型
- `gpt-4-turbo` - 快速版本
- `gpt-3.5-turbo` - 经济版本

**获取密钥**：
1. 访问 https://platform.openai.com/
2. 注册账号（需要海外手机号）
3. 绑定信用卡
4. 创建API密钥

## 嵌入模型配置

### 1. 硅基流动 (推荐中文)

**优势**：
- 专门优化的中文嵌入模型
- 价格便宜
- 支持多种开源模型
- API稳定

**配置**：
```env
EMBEDDING_BASE_URL=https://api.siliconflow.cn/v1
EMBEDDING_KEY=your-siliconflow-api-key
EMBEDDING_MODEL=BAAI/bge-large-zh-v1.5
```

**推荐模型**：
- `BAAI/bge-large-zh-v1.5` - 中文最佳
- `BAAI/bge-m3` - 多语言支持
- `text-embedding-ada-002` - OpenAI兼容

### 2. 通义千问嵌入

**配置**：
```env
EMBEDDING_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
EMBEDDING_KEY=your-qwen-api-key
EMBEDDING_MODEL=text-embedding-v3
```

### 3. OpenAI嵌入

**配置**：
```env
EMBEDDING_BASE_URL=https://api.openai.com/v1
EMBEDDING_KEY=your-openai-api-key
EMBEDDING_MODEL=text-embedding-3-small
```

## 本地模型配置

### 使用 Ollama

**安装 Ollama**：
1. 访问 https://ollama.ai/
2. 下载并安装
3. 启动服务：`ollama serve`

**下载模型**：
```bash
# 下载中文模型
ollama pull qwen:7b
ollama pull bge-large-zh

# 下载英文模型
ollama pull llama2
ollama pull mistral
```

**配置**：
```env
# 主模型
OPENAI_API_KEY=ollama
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_MODEL=qwen:7b

# 嵌入模型
EMBEDDING_BASE_URL=http://localhost:11434/v1
EMBEDDING_KEY=ollama
EMBEDDING_MODEL=bge-large-zh
```

## 配置优化建议

### 高性价比配置
```env
# 主模型：DeepSeek (便宜强大)
OPENAI_API_KEY=sk-your-deepseek-key
OPENAI_BASE_URL=https://api.deepseek.com/v1
OPENAI_MODEL=deepseek-chat

# 嵌入模型：硅基流动 (中文优化)
EMBEDDING_BASE_URL=https://api.siliconflow.cn/v1
EMBEDDING_KEY=your-siliconflow-key
EMBEDDING_MODEL=BAAI/bge-large-zh-v1.5
```

### 高性能配置
```env
# 主模型：GPT-4 (最强能力)
OPENAI_API_KEY=sk-your-openai-key
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4-turbo

# 嵌入模型：OpenAI (速度快)
EMBEDDING_BASE_URL=https://api.openai.com/v1
EMBEDDING_KEY=sk-your-openai-key
EMBEDDING_MODEL=text-embedding-3-large
```

### 本地化配置
```env
# 主模型：本地Ollama
OPENAI_API_KEY=ollama
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_MODEL=qwen:7b

# 嵌入模型：本地模型
EMBEDDING_BASE_URL=http://localhost:11434/v1
EMBEDDING_KEY=ollama
EMBEDDING_MODEL=bge-large-zh
```

## 故障排除

### 常见错误

**1. API密钥无效**
```
Error: invalid API key
```
解决方案：检查API密钥是否正确，是否有足够余额

**2. 网络连接失败**
```
Error: connection timeout
```
解决方案：检查网络连接，考虑使用代理或VPN

**3. 模型不存在**
```
Error: model not found
```
解决方案：检查模型名称是否正确，是否有权限访问

**4. 配额超限**
```
Error: rate limit exceeded
```
解决方案：等待限制重置，或升级API套餐

### 调试技巧

**启用调试模式**：
```env
DEBUG=true
LOG_LEVEL=debug
```

**测试API连接**：
```bash
curl -H "Authorization: Bearer your-api-key" \
     -H "Content-Type: application/json" \
     -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"Hello"}]}' \
     https://api.deepseek.com/v1/chat/completions
```

**检查配置**：
```bash
# 查看环境变量
env | grep -E "(OPENAI|EMBEDDING)"

# 测试程序启动
go run main.go
```