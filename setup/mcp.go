package setup

import (
	"context"
	"fmt"
	"wen-ai-cli/mcp"
)

var mcpManager *mcp.Manager

// InitMCP 初始化 MCP Manager
func InitMCP(ctx context.Context) {
	config := GetConfig()

	if !config.MCP.Enabled {
		fmt.Println("[DEBUG] MCP is disabled in config")
		return
	}

	if len(config.MCP.Servers) == 0 {
		fmt.Println("[DEBUG] No MCP servers configured")
		return
	}

	fmt.Println("[INFO] Initializing MCP Manager...")
	mcpManager = mcp.NewManager(config.MCP.Servers)

	if err := mcpManager.Start(ctx); err != nil {
		fmt.Printf("[ERROR] Failed to start MCP Manager: %v\n", err)
		return
	}

	tools := mcpManager.GetAvailableTools()
	fmt.Printf("[INFO] MCP Manager initialized with %d tools available\n", len(tools))
}

// GetMCPManager 获取 MCP Manager 实例
func GetMCPManager() *mcp.Manager {
	return mcpManager
}

// StopMCP 停止 MCP Manager
func StopMCP() {
	if mcpManager != nil {
		fmt.Println("[DEBUG] Stopping MCP Manager...")
		mcpManager.Stop()
	}
}
