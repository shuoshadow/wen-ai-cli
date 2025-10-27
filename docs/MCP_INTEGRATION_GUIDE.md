# MCP 集成使用指南

## 📚 概述

Hi AI CLI 现已支持 Model Context Protocol (MCP)，可以通过 MCP Server 查询实时数据（监控、日志、服务状态等），让 AI 基于真实数据生成更准确的命令和建议。

## 🏗️ 架构说明

```
用户提问
    ↓
AI 分析需求
    ↓
判断是否需要查询实时数据
    ↓ 需要
调用 MCP 工具获取真实数据
    ↓
AI 基于真实数据生成答案和命令
    ↓
用户确认执行
```

## ⚙️ 配置方法

### 1. 配置文件位置

配置文件位于：`~/.hiai/conf.json`

### 2. 启用 MCP

编辑配置文件，添加 MCP 配置段：

```json
{
  "defaultLang": "zh-CN",
  "openai": {
    "apiKey": "your-api-key",
    "baseURL": "https://api.openai.com/v1",
    "model": "gpt-4"
  },
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "local-system",
        "command": "python3",
        "args": ["/path/to/hi-ai-cli/mcp/servers/local_system_server.py"],
        "env": {},
        "enabled": true,
        "autoStart": true
      }
    ]
  }
}
```

### 3. 配置说明

- **enabled**: 是否启用 MCP 功能
- **servers**: MCP Server 列表
  - **name**: Server 名称（唯一标识）
  - **command**: 启动命令（python3, node, java 等）
  - **args**: 命令参数（Server 脚本路径等）
  - **env**: 环境变量（API Key、Endpoint 等）
  - **enabled**: 是否启用此 Server
  - **autoStart**: 是否自动启动

## 🛠️ 内置 MCP Server

### 本地系统工具 Server

**位置**: `mcp/servers/local_system_server.py`

**提供的工具**：

1. **check_service_status** - 查询系统服务状态
   - 参数：`service_name` (string) - 服务名称，如 nginx, docker

2. **check_process** - 查询进程信息
   - 参数：`process_name` (string) - 进程名称或关键字

3. **check_port** - 查询端口占用情况
   - 参数：`port` (number) - 端口号

4. **disk_usage** - 查询磁盘使用情况
   - 参数：`path` (string, 可选) - 路径，默认为根目录

### 安装依赖

```bash
# 安装 Python MCP SDK
pip install mcp

# 或使用 requirements.txt
pip install -r mcp/servers/requirements.txt
```

## 📖 使用示例

### 示例 1：查询服务状态

```bash
$ hi "nginx 服务运行正常吗？"

[INFO] Initializing MCP Manager...
[INFO] Starting MCP server: local-system
[INFO] MCP server local-system started successfully with 4 tools

AI 正在分析...
我将查询 nginx 服务的状态。

[正在调用工具查询实时数据...]
[调用工具: check_service_status]

[基于查询结果生成答案...]

## 概述：
nginx 服务当前处于运行状态，一切正常。

## 服务详情：
- 状态：active (running)
- 进程 ID：1234
- 启动时间：2天前
- 内存使用：45.2MB

## 待执行脚本：
```code
systemctl status nginx
```

> [立即运行] [微调运行] [退出]
```

### 示例 2：查询端口占用

```bash
$ hi "8080 端口被什么程序占用了？"

[正在调用工具查询实时数据...]
[调用工具: check_port]

## 概述：
8080 端口被 java 进程占用。

## 端口详情：
COMMAND  PID   USER
java     5678  root

## 建议命令：
```code
lsof -i :8080
netstat -tulnp | grep 8080
```
```

### 示例 3：多轮对话

```bash
$ hi chat

> 帮我检查 docker 服务状态

[调用工具: check_service_status]

Docker 服务正在运行...

> 那查看一下有哪些容器在运行

[调用工具: check_process]

找到以下 Docker 容器进程...
```

## 🔧 创建自定义 MCP Server

### Python 示例（推荐）

```python
#!/usr/bin/env python3
import asyncio
from mcp.server import Server
from mcp.types import Tool, TextContent

server = Server("my-custom-server")

@server.list_tools()
async def list_tools() -> list[Tool]:
    return [
        Tool(
            name="my_tool",
            description="工具描述",
            inputSchema={
                "type": "object",
                "properties": {
                    "param1": {"type": "string", "description": "参数说明"}
                },
                "required": ["param1"]
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "my_tool":
        result = do_something(arguments["param1"])
        return [TextContent(type="text", text=result)]

async def main():
    from mcp.server.stdio import stdio_server
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            server.create_initialization_options()
        )

if __name__ == "__main__":
    asyncio.run(main())
```

### Node.js 示例

```javascript
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';

const server = new Server({
  name: 'my-custom-server',
  version: '1.0.0'
}, {
  capabilities: { tools: {} }
});

server.setRequestHandler('tools/list', async () => ({
  tools: [{
    name: 'my_tool',
    description: '工具描述',
    inputSchema: {
      type: 'object',
      properties: {
        param1: { type: 'string', description: '参数说明' }
      },
      required: ['param1']
    }
  }]
}));

server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'my_tool') {
    const result = doSomething(args.param1);
    return {
      content: [{ type: 'text', text: result }]
    };
  }
});

const transport = new StdioServerTransport();
await server.connect(transport);
```

### 添加到配置

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "my-custom-server",
        "command": "python3",
        "args": ["/path/to/my_server.py"],
        "env": {
          "API_KEY": "your-api-key",
          "ENDPOINT": "https://api.example.com"
        },
        "enabled": true,
        "autoStart": true
      }
    ]
  }
}
```

## 🌟 高级场景示例

### Prometheus 监控查询

```python
@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="query_prometheus",
            description="查询 Prometheus 指标数据",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "PromQL 查询语句"},
                    "service": {"type": "string", "description": "服务名称"}
                }
            }
        )
    ]
```

配置：
```json
{
  "name": "prometheus",
  "command": "python3",
  "args": ["./prometheus_server.py"],
  "env": {
    "PROMETHEUS_URL": "http://localhost:9090"
  }
}
```

使用：
```bash
$ hi "查看 API 服务的 QPS"
# AI 会自动调用 Prometheus 查询工具获取真实数据
```

### 阿里云 SLS 日志查询

```python
@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="query_sls_logs",
            description="查询阿里云 SLS 日志",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {"type": "string"},
                    "time_range": {"type": "string"}
                }
            }
        )
    ]
```

配置：
```json
{
  "name": "aliyun-sls",
  "env": {
    "SLS_ENDPOINT": "cn-hangzhou.log.aliyuncs.com",
    "SLS_ACCESS_KEY_ID": "xxx",
    "SLS_ACCESS_KEY_SECRET": "xxx",
    "SLS_PROJECT": "my-project"
  }
}
```

## 🐛 故障排除

### MCP Server 未启动

检查日志：
```bash
$ hi "test" 2>&1 | grep MCP
[ERROR] Failed to start MCP server xxx: ...
```

常见原因：
1. Python/Node 环境未安装
2. MCP SDK 未安装：`pip install mcp`
3. 脚本路径错误
4. 权限不足：`chmod +x server.py`

### 工具调用失败

启用调试日志：
```json
{
  "logger": {
    "console": {
      "level": "debug"
    }
  }
}
```

查看详细错误信息。

## 📊 性能建议

1. **控制 Server 数量**：只启动必要的 Server
2. **设置超时**：工具调用默认超时 30 秒
3. **缓存结果**：在 MCP Server 内部实现缓存
4. **异步处理**：使用异步 I/O 提升性能

## 🔐 安全建议

1. **API Key 管理**：使用环境变量，不要硬编码
2. **权限控制**：限制 MCP Server 的系统权限
3. **输入验证**：在 Server 端验证所有输入参数
4. **日志审计**：记录所有工具调用

## 📚 参考资源

- [MCP 官方文档](https://modelcontextprotocol.io/)
- [MCP Python SDK](https://github.com/modelcontextprotocol/python-sdk)
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)
- [MCP Server 示例](https://github.com/modelcontextprotocol/servers)

## ❓ 常见问题

**Q: 如何知道 AI 会调用哪些工具？**
A: AI 会在 Prompt 中看到所有可用工具的描述，根据用户问题智能选择。

**Q: 可以同时运行多个 MCP Server 吗？**
A: 可以，在配置文件中添加多个 Server 配置即可。

**Q: 支持哪些编程语言？**
A: 任何支持 STDIO 通信的语言都可以（Python、Node.js、Java、Go 等）。

**Q: 工具调用会影响响应速度吗？**
A: 会略微增加响应时间（取决于工具执行速度），但能提供更准确的答案。

---

**祝您使用愉快！有问题欢迎反馈。** 🚀

