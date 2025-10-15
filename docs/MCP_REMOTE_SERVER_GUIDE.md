# 远程 MCP Server 使用指南

## 📡 概述

Wen AI CLI 现已支持两种 MCP Server 连接方式：

1. **本地 STDIO 模式**：在本地启动 MCP Server 进程，通过标准输入输出通信
2. **远程 HTTP 模式**：连接到远程运行的 MCP Server，通过 HTTP/SSE 协议通信

## 🔄 传输类型对比

| 特性 | STDIO 模式 | HTTP 模式 |
|-----|-----------|----------|
| **部署位置** | 本地进程 | 远程服务器 |
| **启动方式** | CLI 自动启动 | 独立运行 |
| **通信协议** | 标准输入输出 | HTTP/SSE |
| **适用场景** | 本地工具、开发测试 | 生产环境、共享服务 |
| **性能** | 低延迟 | 取决于网络 |
| **可扩展性** | 单机 | 分布式 |

---

## 📝 配置示例

### 方式一：本地 STDIO Server（默认）

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "local-system",
        "transport": "stdio",
        "command": "python3",
        "args": ["/path/to/mcp_server.py"],
        "env": {
          "API_KEY": "your-key"
        },
        "enabled": true,
        "autoStart": true
      }
    ]
  }
}
```

**特点**：
- ✅ 简单快速，适合本地开发
- ✅ 无需部署额外服务
- ✅ 支持任何可执行文件（Python/Node.js/Go）

---

### 方式二：远程 HTTP Server（新增）

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "remote-prometheus",
        "transport": "http",
        "url": "http://mcp-server.example.com:8080/mcp",
        "headers": {
          "Authorization": "Bearer your-api-token",
          "X-Custom-Header": "value"
        },
        "enabled": true,
        "autoStart": true
      }
    ]
  }
}
```

**特点**：
- ✅ 可以连接到远程部署的 MCP Server
- ✅ 支持自定义 HTTP Headers（如 JWT Token）
- ✅ 适合生产环境和团队共享
- ✅ 支持负载均衡和高可用

---

## 🚀 使用场景

### 场景 1：团队共享的监控服务

**需求**：多个开发者需要查询同一个 Prometheus 监控系统

**方案**：部署一个远程 HTTP MCP Server

```json
{
  "name": "team-prometheus",
  "transport": "http",
  "url": "https://mcp.company.com/prometheus",
  "headers": {
    "Authorization": "Bearer team-shared-token"
  },
  "enabled": true,
  "autoStart": true
}
```

**优势**：
- 集中管理 Prometheus 访问权限
- 统一配置，无需每个人本地启动
- 可以在 MCP Server 层实现缓存和限流

---

### 场景 2：云服务 API 集成

**需求**：查询阿里云 SLS 日志，但不想在本地配置 AK/SK

**方案**：在云上部署 MCP Server，本地只需配置 HTTP 连接

```json
{
  "name": "aliyun-sls",
  "transport": "http",
  "url": "https://api.company.com/mcp/sls",
  "headers": {
    "X-API-Key": "user-specific-key"
  },
  "enabled": true,
  "autoStart": true
}
```

**优势**：
- 安全：AK/SK 只存在服务端
- 审计：所有查询日志都可追踪
- 权限控制：基于用户 API Key 进行权限管理

---

### 场景 3：混合部署

**需求**：本地工具用 STDIO，远程服务用 HTTP

```json
{
  "mcp": {
    "enabled": true,
    "servers": [
      {
        "name": "local-system",
        "transport": "stdio",
        "command": "python3",
        "args": ["./local_tools_server.py"],
        "enabled": true,
        "autoStart": true
      },
      {
        "name": "remote-monitoring",
        "transport": "http",
        "url": "https://monitoring.company.com/mcp",
        "headers": {
          "Authorization": "Bearer ${MCP_TOKEN}"
        },
        "enabled": true,
        "autoStart": true
      }
    ]
  }
}
```

**优势**：
- 本地工具快速响应（STDIO）
- 远程数据统一管理（HTTP）
- 灵活组合，按需配置

---

## 🛠️ 部署远程 MCP Server

### Python 示例（使用 FastAPI + MCP）

```python
#!/usr/bin/env python3
"""
远程 HTTP MCP Server 示例
使用 FastAPI 提供 HTTP 接口
"""
import asyncio
from fastapi import FastAPI, Request, Header
from fastapi.responses import StreamingResponse
from mcp.server import Server
from mcp.types import Tool, TextContent
import uvicorn

# 创建 FastAPI 应用
app = FastAPI()

# 创建 MCP Server
mcp_server = Server("remote-monitoring")

@mcp_server.list_tools()
async def list_tools() -> list[Tool]:
    return [
        Tool(
            name="query_metrics",
            description="查询监控指标",
            inputSchema={
                "type": "object",
                "properties": {
                    "metric": {"type": "string"},
                    "time_range": {"type": "string"}
                },
                "required": ["metric"]
            }
        )
    ]

@mcp_server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "query_metrics":
        # 实际查询 Prometheus 或其他监控系统
        result = query_prometheus(arguments["metric"])
        return [TextContent(type="text", text=result)]

# HTTP 端点：处理 MCP 协议
@app.post("/mcp")
async def mcp_endpoint(
    request: Request,
    authorization: str = Header(None)
):
    # 验证 Token
    if not verify_token(authorization):
        return {"error": "Unauthorized"}, 401
    
    # 处理 MCP 请求
    body = await request.json()
    response = await mcp_server.handle_request(body)
    return response

def verify_token(auth_header: str) -> bool:
    # 实现 Token 验证逻辑
    return auth_header == "Bearer valid-token"

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
```

**启动服务**：
```bash
pip install fastapi uvicorn mcp
python remote_mcp_server.py
```

---

### Node.js 示例（使用 Express）

```javascript
import express from 'express';
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { SSEServerTransport } from '@modelcontextprotocol/sdk/server/sse.js';

const app = express();
const mcpServer = new Server({
  name: 'remote-monitoring',
  version: '1.0.0'
}, {
  capabilities: { tools: {} }
});

// 注册工具
mcpServer.setRequestHandler('tools/list', async () => ({
  tools: [{
    name: 'query_metrics',
    description: '查询监控指标',
    inputSchema: {
      type: 'object',
      properties: {
        metric: { type: 'string' }
      }
    }
  }]
}));

mcpServer.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'query_metrics') {
    const result = await queryPrometheus(args.metric);
    return {
      content: [{ type: 'text', text: result }]
    };
  }
});

// HTTP 端点
app.post('/mcp', async (req, res) => {
  // 验证 Token
  const token = req.headers.authorization;
  if (!verifyToken(token)) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  
  // 处理 MCP 请求
  const transport = new SSEServerTransport('/mcp', res);
  await mcpServer.connect(transport);
});

app.listen(8080, () => {
  console.log('MCP Server running on port 8080');
});
```

---

## 🔐 安全最佳实践

### 1. 使用 HTTPS
```json
{
  "url": "https://mcp.company.com/api",
  "headers": {
    "Authorization": "Bearer ${MCP_TOKEN}"
  }
}
```

### 2. Token 认证
```bash
# 设置环境变量
export MCP_TOKEN="your-secure-token"

# 配置中引用
"headers": {
  "Authorization": "Bearer ${MCP_TOKEN}"
}
```

### 3. IP 白名单
在 MCP Server 端实现：
```python
ALLOWED_IPS = ["10.0.0.0/8", "192.168.1.0/24"]

def verify_client_ip(client_ip: str) -> bool:
    return client_ip in ALLOWED_IPS
```

### 4. 请求限流
```python
from slowapi import Limiter

limiter = Limiter(key_func=get_remote_address)

@app.post("/mcp")
@limiter.limit("100/minute")
async def mcp_endpoint(request: Request):
    # ...
```

---

## 📊 监控和日志

### 服务端监控
```python
from prometheus_client import Counter, Histogram

mcp_requests = Counter('mcp_requests_total', 'Total MCP requests')
mcp_latency = Histogram('mcp_request_latency', 'MCP request latency')

@app.post("/mcp")
async def mcp_endpoint(request: Request):
    mcp_requests.inc()
    
    with mcp_latency.time():
        response = await handle_mcp_request(request)
    
    return response
```

### 客户端日志
Wen AI CLI 会自动记录：
- MCP Server 连接状态
- 工具调用时间
- 错误和重试信息

---

## 🐛 故障排除

### 连接失败

**问题**：无法连接到远程 MCP Server

**排查**：
```bash
# 1. 检查网络连通性
curl http://your-server:8080/mcp

# 2. 检查 Token 是否正确
curl -H "Authorization: Bearer your-token" \
     http://your-server:8080/mcp

# 3. 查看 CLI 日志
tail -f ~/.wenai/logs/app.log
```

### 认证失败

**问题**：401 Unauthorized

**解决**：
1. 确认 Token 配置正确
2. 检查 Headers 格式：`"Authorization": "Bearer token"`
3. 确认服务端 Token 验证逻辑

### 超时问题

**问题**：工具调用超时

**解决**：
- 默认超时 30 秒，可能需要增加
- 在服务端实现缓存机制
- 优化查询性能

---

## 📈 性能优化

### 1. 连接池
```python
# 服务端使用异步处理
import asyncio

async def handle_concurrent_requests():
    tasks = [process_request(req) for req in pending_requests]
    results = await asyncio.gather(*tasks)
    return results
```

### 2. 缓存
```python
from functools import lru_cache
import time

@lru_cache(maxsize=100)
def query_with_cache(metric: str, ttl: int = 60):
    # 缓存 60 秒
    result = query_actual_data(metric)
    return result
```

### 3. CDN 加速
对于公开的 MCP Server，可以通过 CDN 加速：
```
Client → CDN → Origin MCP Server
```

---

## 🌟 生产环境部署清单

- [ ] 使用 HTTPS 加密传输
- [ ] 实现 Token 认证
- [ ] 配置请求限流
- [ ] 添加监控和告警
- [ ] 实现日志审计
- [ ] 配置负载均衡
- [ ] 设置健康检查
- [ ] 准备灾备方案

---

## 💡 总结

### 何时使用 STDIO 模式？
- ✅ 本地开发和测试
- ✅ 个人使用的简单工具
- ✅ 不需要共享的功能

### 何时使用 HTTP 模式？
- ✅ 生产环境部署
- ✅ 团队共享服务
- ✅ 需要集中管理权限
- ✅ 云服务 API 集成
- ✅ 需要高可用和负载均衡

---

**现在您可以灵活选择本地或远程 MCP Server，满足不同场景的需求！** 🚀

