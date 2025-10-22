package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"wen-ai-cli/common"
	"wen-ai-cli/logger"
	"wen-ai-cli/setup"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var systemMessage = `{baseInfo}

{workUserAndDir}

{workPlatform}

{availableTools}

{workFlow}

{answerDescription}

{answerFormat}`

var baseInfo = `- 角色：命令行界面（CLI）专家和系统命令生成顾问。

- 背景: 用户在使用不同的操作系统和Shell工具时，面临命令差异和复杂参数理解的挑战，需要一个助手来生成准确且高效的命令，并提供清晰的说明。

- 简介: 你是一位精通多种操作系统（如Linux、Windows、macOS）和Shell工具（如Bash、Zsh、Fish、PowerShell等）的专家，对命令行操作有着深入的理解和丰富的实践经验，能够根据用户的需求快速生成最佳命令，并提供详细的说明。

- 技能: 你具备操作系统原理、Shell脚本编程、命令行工具使用以及文档编写的能力，能够准确解析用户需求，生成适用于目标系统的命令。

- 目标: 根据用户指定的目标系统和当前使用的命令行工具，生成最佳执行命令。

- 约束: 生成的命令应准确无误，符合目标系统和Shell工具的语法规范，说明应清晰易懂，适合不同技术水平的用户。`

var workPlatform = `-> 目标系统信息：操作系统是"{systemInfo}"，命令行工具是"{shellPlatform}"`

var workUserAndDir = `-> 目标用户是："{workUser}"，目标用户工作目录是："{workDir}"`

var workFlowSteps = []struct {
	Step    string
	Enable  bool
	Default bool
}{
	{
		Step:    "根据提供的目标用户信息、工作目录，判断是否需要使用sudo执行命令、是否需要添加当前工作目录路径，智能的在回答中调整脚本（比如root用户，是不需要在回答的命令中添加sudo，而非root用户则需要添加sudo）",
		Enable:  true,
		Default: true,
	},
	{
		Step:    "根据提供目标系统环境等信息，按照用户需求，生成适配于当前系统环境的一个最佳实践命令脚本。",
		Enable:  true,
		Default: true,
	},
	{
		Step:    "如果用户需求需要多个命令才能完成，请将多个命令使用当前平台支持的\"多命令连接符号\"连接，例如：opkg update && opkg install <包名称,string>",
		Enable:  true,
		Default: true,
	},
	{
		Step:    "按照指定格式，输出命令、命令分析和常用参数。",
		Enable:  true,
		Default: true,
	},
}

var answerDescription = `- 回答说明: 
	1. 最佳脚本必须使用<code>和</code>包裹。	
	2. <code>标签最佳脚本中，如需用户补充参数值必须使用 < 和 > 符号包裹，且格式为: <参数解释,此参数类型[可选：url,string,number]>。
	3. code内容示例：<code> curl -o <本地文件名称,string> <下载文件的URL,url> </code>
	4. <placeholder></placeholder>标签中的内容为占位说明，必须按照占位说明进行替换，且不保留<placeholder>标签。
	5. 如果用户与你存在多轮对话，你的历史回答可能是错误的，或者回答格式不符合参考格式标准，请结合历史对话内容和最新用户意图，在能够解答用户问题的前提下，必须使用完整的正确的参考格式回答。
	6. 【重要】工具使用优先级规则：
	   a. 如果用户需求可以通过可用工具完成（如查询实时数据、监控指标、日志、服务状态等），必须优先且仅使用工具，不要生成命令脚本。
	   b. 只有在没有合适的工具或工具无法解决问题时，才生成命令脚本。
	   c. 工具使用格式：
	      <tool_call>
	      {
	        "name": "工具名称",
	        "arguments": {
	          "参数名": "参数值"
	        }
	      }
	      </tool_call>
	   d. 使用工具时，只需在"概述"部分简要说明要使用的工具和原因，不要生成"待执行脚本"部分。`

var answerFormat = `-> 参考回答格式：

【如果使用MCP工具】：
## 概述：
<placeholder>简要说明要使用哪个工具以及原因</placeholder>

<tool_call>
{
  "name": "工具名称",
  "arguments": {
    "参数名": "参数值"
  }
}
</tool_call>

【如果生成命令脚本】：
## 概述：
<placeholder>此处替换为能够解答用户问题的脚本概述或原因</placeholder>

## 待执行脚本：
<code>
	<placeholder>你给出的最佳脚本</placeholder>
</code>
{scriptExplain}
{extendParams}`

var scriptExplain = `## 脚本分析：
<placeholder>工具名称：工具用途
1. -a: <参数1解释>
2. -b: <参数2解释>
3. <以此类推></placeholder>`

var extendParams = `## 常用参数：
<placeholder>
1. -x: <常用参数1解释>
2. -y: <常用参数2解释>
3. <以此类推，最多5个>
</placeholder>`

// getAvailableTools 获取可用工具描述
func getAvailableTools() string {
	mcpManager := setup.GetMCPManager()
	if mcpManager == nil {
		return ""
	}

	tools := mcpManager.GetAvailableTools()
	if len(tools) == 0 {
		return ""
	}

	var toolsDesc strings.Builder
	toolsDesc.WriteString("-> 可用工具列表（用于查询实时数据）：\n")
	for i, tool := range tools {
		toolsDesc.WriteString(fmt.Sprintf("  %d. %s - %s\n", i+1, tool.Name, tool.Description))
	}

	return toolsDesc.String()
}

func createTemplate() prompt.ChatTemplate {
	// 创建模板，使用 FString 格式
	return prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage(systemMessage),

		// 插入需要的对话历史（新对话的话这里不填）
		schema.MessagesPlaceholder("chatHistory", true),

		// 用户消息模板
		schema.UserMessage("{question}"),
	)
}

func getAnswerFormat(enableExplain bool, enableExtendParams bool) string {
	result := answerFormat
	result = strings.Replace(result, "<code>", "```code", -1)
	result = strings.Replace(result, "</code>", "```", -1)
	if enableExplain {
		result = strings.Replace(result, "{scriptExplain}", scriptExplain, -1)
	} else {
		result = strings.Replace(result, "{scriptExplain}", "", -1)
	}
	if enableExtendParams {
		result = strings.Replace(result, "{extendParams}", extendParams, -1)
	} else {
		result = strings.Replace(result, "{extendParams}", "", -1)
	}

	return result
}

func getWorkPlatform(enablePlatformPerception bool) string {
	// 获取配置信息，是否启用平台感知
	if !enablePlatformPerception {
		return ""
	}
	systemInfo, err := common.GetSystemInfo()
	if err != nil {
		logger.Errorf("get system info failed: %v\n", err)
	}
	shellPlatform, err := common.GetShellPlatform()
	if err != nil {
		logger.Errorf("get shell platform failed: %v\n", err)
	}
	result := workPlatform
	result = strings.Replace(result, "{systemInfo}", systemInfo, -1)
	result = strings.Replace(result, "{shellPlatform}", shellPlatform, -1)
	return result
}

func getWorkUserAndDir(enableWorkUserAndDir bool) string {
	if !enableWorkUserAndDir {
		return ""
	}
	user, err := common.GetUser()
	if err != nil {
		logger.Errorf("get user failed: %v\n", err)
	}
	pwd, err := common.GetPwd()
	if err != nil {
		logger.Errorf("get pwd failed: %v\n", err)
	}
	result := workUserAndDir
	result = strings.Replace(result, "{workUser}", user, -1)
	result = strings.Replace(result, "{workDir}", pwd, -1)
	return result
}

func getWorkFlow(enablePlatformPerception bool, enableWorkUserAndDir bool, enableExplain bool, enableExtendParams bool) string {
	// 重置所有步骤为默认状态
	for i := range workFlowSteps {
		workFlowSteps[i].Enable = workFlowSteps[i].Default
	}

	// 根据enable参数设置工作流程步骤的启用状态
	if !enableWorkUserAndDir {
		workFlowSteps[0].Enable = false
	}
	if !enablePlatformPerception {
		workFlowSteps[1].Enable = false
	}

	// 动态调整第4步的描述
	step4Text := "按照指定格式，输出命令"
	if enableExplain && enableExtendParams {
		step4Text = "按照指定格式，输出命令、命令分析和常用参数。"
	} else if enableExplain {
		step4Text = "按照指定格式，输出命令和命令分析。"
	} else if enableExtendParams {
		step4Text = "按照指定格式，输出命令和常用参数。"
	} else {
		step4Text = "按照指定格式，输出命令。"
	}
	workFlowSteps[3].Step = step4Text

	var steps []string
	for i, step := range workFlowSteps {
		if step.Enable {
			steps = append(steps, fmt.Sprintf("  %d. %s", i+1, step.Step))
		}
	}
	return "- 工作流程:\n" + strings.Join(steps, "\n")
}

func CreateOnceMessagesFromTemplate(question string, enableExplain bool, enableExtendParams bool, enablePlatformPerception bool, enableWorkUserAndDir bool) []*schema.Message {
	template := createTemplate()
	// 使用模板生成消息
	messages, err := template.Format(context.Background(), map[string]any{
		"baseInfo":          baseInfo,
		"workFlow":          getWorkFlow(enablePlatformPerception, enableWorkUserAndDir, enableExplain, enableExtendParams),
		"workPlatform":      getWorkPlatform(enablePlatformPerception),
		"workUserAndDir":    getWorkUserAndDir(enableWorkUserAndDir),
		"availableTools":    getAvailableTools(),
		"answerDescription": answerDescription,
		"answerFormat":      getAnswerFormat(enableExplain, enableExtendParams),
		"question":          question,
		// 对话历史
		"chatHistory": []*schema.Message{},
	})
	if err != nil {
		log.Fatalf("format template failed: %v", err)
	}
	return messages
}

func CreateMoreMessagesFromTemplate(question string, chatHistory []*schema.Message, enableExplain bool, enableExtendParams bool, enablePlatformPerception bool, enableWorkUserAndDir bool) []*schema.Message {
	template := createTemplate()
	// 使用模板生成消息
	messages, err := template.Format(context.Background(), map[string]any{
		"baseInfo":          baseInfo,
		"workFlow":          getWorkFlow(enablePlatformPerception, enableWorkUserAndDir, enableExplain, enableExtendParams),
		"workPlatform":      getWorkPlatform(enablePlatformPerception),
		"workUserAndDir":    getWorkUserAndDir(enableWorkUserAndDir),
		"availableTools":    getAvailableTools(),
		"answerDescription": answerDescription,
		"answerFormat":      getAnswerFormat(enableExplain, enableExtendParams),
		"question":          question,
		// 对话历史
		"chatHistory": chatHistory,
	})
	if err != nil {
		log.Fatalf("format template failed: %v", err)
	}
	return messages
}
