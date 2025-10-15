package mcp

// TransportType MCP 传输类型
type TransportType string

const (
	TransportTypeStdio TransportType = "stdio" // 本地进程通信（默认）
	TransportTypeHTTP  TransportType = "http"  // 远程 HTTP/SSE 通信
)

// ServerConfig MCP Server 配置
type ServerConfig struct {
	Name      string        `json:"name" mapstructure:"name"`
	Transport TransportType `json:"transport" mapstructure:"transport"` // 传输类型：stdio 或 http

	// STDIO 传输配置（本地启动）
	Command string            `json:"command,omitempty" mapstructure:"command"`
	Args    []string          `json:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `json:"env,omitempty" mapstructure:"env"`

	// HTTP 传输配置（远程连接）
	URL     string            `json:"url,omitempty" mapstructure:"url"`         // 远程 Server URL
	Headers map[string]string `json:"headers,omitempty" mapstructure:"headers"` // HTTP 请求头（如 Authorization）

	Enabled   bool `json:"enabled" mapstructure:"enabled"`
	AutoStart bool `json:"autoStart" mapstructure:"autoStart"`
}

// Tool 工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"-"` // 所属 Server
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

// ContentItem 内容项
type ContentItem struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// MCPConfig MCP 总体配置
type MCPConfig struct {
	Enabled bool           `json:"enabled" mapstructure:"enabled"`
	Servers []ServerConfig `json:"servers" mapstructure:"servers"`
}
