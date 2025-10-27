package setup

import (
	"context"
	"hi-ai-cli/logger"
	"hi-ai-cli/mcp"
)

var mcpManager *mcp.Manager

// InitMCP 初始化 MCP Manager
func InitMCP(ctx context.Context) {
	config := GetConfig()

	if !config.MCP.Enabled {
		logger.Debug("MCP is disabled in config")
		return
	}

	if len(config.MCP.Servers) == 0 {
		logger.Debug("No MCP servers configured, skipping MCP Manager initialization")
		return
	}

	logger.Debug("Initializing MCP Manager...")
	mcpManager = mcp.NewManager(config.MCP.Servers)

	if err := mcpManager.Start(ctx); err != nil {
		logger.Errorf("Failed to start MCP Manager: %v", err)
		return
	}

	tools := mcpManager.GetAvailableTools()
	logger.Debugf("MCP Manager started successfully with %d tools", len(tools))
}

// GetMCPManager 获取 MCP Manager 实例
func GetMCPManager() *mcp.Manager {
	return mcpManager
}

// StopMCP 停止 MCP Manager
func StopMCP() {
	if mcpManager != nil {
		logger.Debug("Stopping MCP Manager...")
		mcpManager.Stop()
	}
}
