package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
	"hi-ai-cli/logger"

	"github.com/mark3labs/mcp-go/client"
	mcpTypes "github.com/mark3labs/mcp-go/mcp"
)

// Manager MCP Server 管理器
type Manager struct {
	servers map[string]*ServerConnection // serverName -> connection
	tools   map[string]*Tool             // toolName -> Tool
	mu      sync.RWMutex
	configs []ServerConfig
}

// ServerConnection 单个 MCP Server 连接
type ServerConnection struct {
	config     ServerConfig
	client     *client.Client
	serverInfo *mcpTypes.InitializeResult
	tools      []mcpTypes.Tool
	connected  bool
	mu         sync.RWMutex
}

// NewManager 创建 MCP Manager
func NewManager(configs []ServerConfig) *Manager {
	return &Manager{
		servers: make(map[string]*ServerConnection),
		tools:   make(map[string]*Tool),
		configs: configs,
	}
}

// Start 启动所有配置的 MCP Servers
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cfg := range m.configs {
		if !cfg.Enabled || !cfg.AutoStart {
			logger.Debugf("Skipping MCP server %s (enabled=%v, autoStart=%v)", cfg.Name, cfg.Enabled, cfg.AutoStart)
			continue
		}

		logger.Debugf("Starting MCP server: %s", cfg.Name)
		conn := &ServerConnection{
			config: cfg,
		}

		if err := conn.Connect(ctx); err != nil {
			logger.Errorf("Failed to start MCP server %s: %v\n", cfg.Name, err)
			continue
		}

		m.servers[cfg.Name] = conn

		// 注册该 Server 的所有工具
		for _, tool := range conn.tools {
			// 将 ToolInputSchema 转换为 map[string]interface{}
			inputSchemaMap := make(map[string]interface{})
			if tool.InputSchema.Type != "" {
				inputSchemaMap["type"] = tool.InputSchema.Type
			}
			if tool.InputSchema.Properties != nil {
				inputSchemaMap["properties"] = tool.InputSchema.Properties
			}
			if len(tool.InputSchema.Required) > 0 {
				inputSchemaMap["required"] = tool.InputSchema.Required
			}

			m.tools[tool.Name] = &Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: inputSchemaMap,
				ServerName:  cfg.Name,
			}
		}

		logger.Debugf("MCP server %s started successfully with %d tools", cfg.Name, len(conn.tools))
	}

	return nil
}

// Stop 停止所有 MCP Servers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, conn := range m.servers {
		logger.Debug("Stopping MCP server: " + name)
		conn.Disconnect()
	}

	m.servers = make(map[string]*ServerConnection)
	m.tools = make(map[string]*Tool)
	return nil
}

// GetAvailableTools 获取所有可用工具列表
func (m *Manager) GetAvailableTools() []*Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]*Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// CallTool 调用指定工具
func (m *Manager) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResult, error) {
	m.mu.RLock()
	tool, exists := m.tools[req.Name]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("tool %s not found", req.Name)
	}

	serverName := tool.ServerName
	conn, exists := m.servers[serverName]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("server %s not found for tool %s", serverName, req.Name)
	}
	m.mu.RUnlock()

	return conn.CallTool(ctx, req.Name, req.Arguments)
}

// GetToolsDescription 获取工具描述（用于 Prompt）
func (m *Manager) GetToolsDescription() string {
	tools := m.GetAvailableTools()
	if len(tools) == 0 {
		return ""
	}

	desc := "以下工具可用于查询实时数据：\n"
	for i, tool := range tools {
		desc += fmt.Sprintf("%d. %s - %s\n", i+1, tool.Name, tool.Description)
	}
	return desc
}

// Connect 连接到 MCP Server
func (sc *ServerConnection) Connect(ctx context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var mcpClient *client.Client
	var err error

	// 根据传输类型创建不同的客户端
	switch sc.config.Transport {
	case TransportTypeHTTP:
		// HTTP 传输
		if sc.config.URL == "" {
			return fmt.Errorf("HTTP transport requires URL")
		}

		logger.Debugf("Connecting to remote MCP server: %s\n", sc.config.URL)

		// 创建 HTTP 客户端（使用 Streamable HTTP 传输）
		mcpClient, err = client.NewStreamableHttpClient(sc.config.URL)
		if err != nil {
			return fmt.Errorf("failed to create HTTP client: %w", err)
		}

	case TransportTypeStdio, "":
		// STDIO 传输
		if sc.config.Command == "" {
			return fmt.Errorf("STDIO transport requires command")
		}

		// 设置环境变量
		env := make([]string, 0, len(sc.config.Env))
		for k, v := range sc.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		// 创建 stdio 客户端
		mcpClient, err = client.NewStdioMCPClient(
			sc.config.Command,
			env,
			sc.config.Args...,
		)
		if err != nil {
			return fmt.Errorf("failed to create STDIO client: %w", err)
		}

	default:
		return fmt.Errorf("unsupported transport type: %s", sc.config.Transport)
	}

	sc.client = mcpClient

	if sc.config.Transport == TransportTypeStdio || sc.config.Transport == "" {
		// 等待进程启动，特别是 npx 可能需要下载包
		time.Sleep(2 * time.Second)
	}

	// 初始化连接（增加超时时间到 30 秒）
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	initRequest := mcpTypes.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpTypes.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpTypes.Implementation{
		Name:    "wen-ai-cli",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcpTypes.ClientCapabilities{}

	serverInfo, err := sc.client.Initialize(initCtx, initRequest)
	if err != nil {
		sc.client.Close()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	sc.serverInfo = serverInfo
	sc.connected = true

	// 获取工具列表
	if serverInfo.Capabilities.Tools != nil {
		toolsRequest := mcpTypes.ListToolsRequest{}
		toolsResult, err := sc.client.ListTools(initCtx, toolsRequest)
		if err != nil {
			fmt.Printf("[ERROR] Failed to list tools from %s: %v\n", sc.config.Name, err)
		} else {
			sc.tools = toolsResult.Tools
		}
	}

	return nil
}

// Disconnect 断开连接
func (sc *ServerConnection) Disconnect() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.client != nil {
		sc.client.Close()
		sc.client = nil
	}
	sc.connected = false
}

// CallTool 调用工具
func (sc *ServerConnection) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolCallResult, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if !sc.connected || sc.client == nil {
		return nil, fmt.Errorf("server %s is not connected", sc.config.Name)
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 构建调用请求
	callRequest := mcpTypes.CallToolRequest{}
	callRequest.Params.Name = toolName
	callRequest.Params.Arguments = args

	// 调用工具
	result, err := sc.client.CallTool(callCtx, callRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	// 转换结果
	var contents []ContentItem
	for _, content := range result.Content {
		switch v := content.(type) {
		case mcpTypes.TextContent:
			contents = append(contents, ContentItem{
				Type: "text",
				Text: v.Text,
			})
		case mcpTypes.ImageContent:
			contents = append(contents, ContentItem{
				Type: "image",
				Data: v.Data,
			})
		case mcpTypes.EmbeddedResource:
			// EmbeddedResource 的 Resource 字段可能是 TextResourceContents 或 BlobResourceContents
			contentItem := ContentItem{
				Type: "resource",
			}

			// 尝试断言为 TextResourceContents
			if textRes, ok := v.Resource.(mcpTypes.TextResourceContents); ok {
				contentItem.URI = textRes.URI
				contentItem.Text = textRes.Text
			} else if blobRes, ok := v.Resource.(mcpTypes.BlobResourceContents); ok {
				contentItem.URI = blobRes.URI
				contentItem.Data = blobRes.Blob
			}

			contents = append(contents, contentItem)
		}
	}

	return &ToolCallResult{
		Content: contents,
		IsError: result.IsError,
	}, nil
}
