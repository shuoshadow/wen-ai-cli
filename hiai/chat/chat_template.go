package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"hi-ai-cli/common"
	"hi-ai-cli/logger"
	"hi-ai-cli/setup"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var systemMessage = `{baseInfo}

{workUserAndDir}

{workPlatform}

{k8sContext}

{availableTools}

{workFlow}

{answerDescription}

{answerFormat}`

var baseInfoRole = `- 角色：命令行界面（CLI）专家、Kubernetes 运维专家和系统命令生成顾问。`

var baseInfoBackground = `- 背景: 用户在使用不同的操作系统和Shell工具时，特别是在 Kubernetes 容器环境中进行运维操作时，面临命令差异和复杂参数理解的挑战，需要一个助手来生成准确且高效的命令，并提供清晰的说明。`

var baseInfoIntro = `- 简介: 你是一位精通多种操作系统（如Linux、Windows、macOS）、Shell工具（如Bash、Zsh、Fish、PowerShell等）和 Kubernetes 运维的专家，对命令行操作、容器技术和云原生运维有着深入的理解和丰富的实践经验，能够根据用户的需求快速生成最佳命令或使用合适的工具。`

var baseInfoConstraint = `- 约束: 生成的命令应准确无误，符合目标系统和Shell工具的语法规范，说明应清晰易懂，适合不同技术水平的用户。`

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
	   d. 使用工具时，只需在"概述"部分简要说明要使用的工具和原因，不要生成"待执行脚本"部分。
	7. 【重要】MCP 工具参数智能推断规则：
	   a. 当工具需要 namespace 参数时：
	      - 如果用户明确指定了命名空间，使用指定的命名空间
	      - 如果当前在 K8s 环境中且已知当前命名空间，优先使用当前命名空间
	      - 如果用户没有指定且不在 K8s 环境中，或需要跨命名空间查询，使用 "default"
	   b. 当工具需要 kind 参数时（K8s 资源类型）：
	      - 从用户问题中提取资源类型关键词：
	        * "pod" / "Pod" / "容器" → kind: "pod"
	        * "deployment" / "Deployment" / "部署" → kind: "deployment"
	        * "service" / "Service" / "服务" → kind: "service"
	        * "statefulset" / "StatefulSet" → kind: "statefulset"
	        * "daemonset" / "DaemonSet" → kind: "daemonset"
	      - 如果用户说 "pod是xxx" / "xxx的pod" / "xxx容器"，kind 应该为 "pod"
	      - 如果无法确定，对于工作负载相关查询，默认使用 kind: "pod"
	   c. 当工具需要 container/pod/workload 名称时：
	      - 从用户问题中提取关键词（如 "hop容器" → workload: "hop", kind: "pod"）
	      - "pod是hop" / "hop的pod" → workload: "hop", kind: "pod"
	      - 支持模糊匹配，不需要完整名称
	   d. 参数缺失处理：
	      - 对于可选参数，可以不传递
	      - 对于必需参数（如 namespace、kind），必须从上下文推断合理的默认值
	      - 确保所有必需参数都有值后再调用工具，避免因参数缺失导致调用失败
	      - 只有在完全无法推断必需参数时，才询问用户`

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

	// 添加常见使用示例
	toolsDesc.WriteString("\n-> 工具使用示例：\n")
	toolsDesc.WriteString("  【示例1】查询hop容器/pod的内存：\n")
	toolsDesc.WriteString(`    用户问题："hop容器内存多少" 或 "pod是hop的容器内存"`)
	toolsDesc.WriteString("\n")
	toolsDesc.WriteString(`    正确调用：{"name": "workload resource usage", "arguments": {"workload": "hop", "kind": "pod", "namespace": "all", "resource_type": "memory"}}`)
	toolsDesc.WriteString("\n")
	toolsDesc.WriteString("  【示例2】查询某个deployment的CPU使用：\n")
	toolsDesc.WriteString(`    用户问题："xxx部署的CPU使用率"`)
	toolsDesc.WriteString("\n")
	toolsDesc.WriteString(`    正确调用：{"name": "workload resource usage", "arguments": {"workload": "xxx", "kind": "deployment", "namespace": "all", "resource_type": "cpu"}}`)
	toolsDesc.WriteString("\n")

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

// getBaseInfo 根据配置动态生成baseInfo
func getBaseInfo(enableExplain bool, enableExtendParams bool) string {
	var parts []string
	parts = append(parts, baseInfoRole)
	parts = append(parts, baseInfoBackground)
	parts = append(parts, baseInfoIntro)

	// 根据配置动态生成技能和目标描述
	var skill, goal string
	if enableExplain && enableExtendParams {
		skill = "- 技能: 你具备操作系统原理、Shell脚本编程、命令行工具使用、Kubernetes运维以及文档编写的能力，能够准确解析用户需求，生成适用于目标系统的命令，并提供命令说明和扩展参数的详细解释。"
		goal = "- 目标: 根据用户指定的目标系统和当前使用的命令行工具，生成最佳执行命令，并提供命令说明和扩展参数说明。在Kubernetes环境中，优先使用MCP工具进行运维操作。"
	} else if enableExplain {
		skill = "- 技能: 你具备操作系统原理、Shell脚本编程、命令行工具使用、Kubernetes运维以及文档编写的能力，能够准确解析用户需求，生成适用于目标系统的命令，并提供命令说明。"
		goal = "- 目标: 根据用户指定的目标系统和当前使用的命令行工具，生成最佳执行命令，并提供命令说明。在Kubernetes环境中，优先使用MCP工具进行运维操作。"
	} else if enableExtendParams {
		skill = "- 技能: 你具备操作系统原理、Shell脚本编程、命令行工具使用、Kubernetes运维以及文档编写的能力，能够准确解析用户需求，生成适用于目标系统的命令，并提供扩展参数的详细解释。"
		goal = "- 目标: 根据用户指定的目标系统和当前使用的命令行工具，生成最佳执行命令，并提供扩展参数说明。在Kubernetes环境中，优先使用MCP工具进行运维操作。"
	} else {
		skill = "- 技能: 你具备操作系统原理、Shell脚本编程、命令行工具使用以及Kubernetes运维的能力，能够准确解析用户需求，生成适用于目标系统的命令。"
		goal = "- 目标: 根据用户指定的目标系统和当前使用的命令行工具，生成最佳执行命令。在Kubernetes环境中，优先使用MCP工具进行运维操作。"
	}

	parts = append(parts, skill)
	parts = append(parts, goal)
	parts = append(parts, baseInfoConstraint)

	return strings.Join(parts, "\n\n")
}

// getK8sContext 获取K8s环境上下文信息
func getK8sContext() string {
	// 检测是否在容器中运行
	if !common.IsInContainer() {
		return ""
	}

	k8sCtx := common.GetK8sContext()
	if len(k8sCtx) == 0 {
		return ""
	}

	var contextInfo []string
	contextInfo = append(contextInfo, "-> Kubernetes 环境信息：")

	currentNamespace := ""
	if podName, ok := k8sCtx["pod_name"]; ok {
		contextInfo = append(contextInfo, fmt.Sprintf("  - 当前 Pod: %s", podName))
	}
	if namespace, ok := k8sCtx["namespace"]; ok {
		currentNamespace = namespace
		contextInfo = append(contextInfo, fmt.Sprintf("  - 当前命名空间: %s", namespace))
	}
	if nodeName, ok := k8sCtx["node_name"]; ok {
		contextInfo = append(contextInfo, fmt.Sprintf("  - 节点: %s", nodeName))
	}
	if currentCtx, ok := k8sCtx["current_context"]; ok {
		contextInfo = append(contextInfo, fmt.Sprintf("  - 集群上下文: %s", currentCtx))
	}
	if clusterAdmin, ok := k8sCtx["cluster_admin"]; ok && clusterAdmin == "true" {
		contextInfo = append(contextInfo, "  - 权限级别: 集群管理员")
	} else {
		contextInfo = append(contextInfo, "  - 权限级别: 受限")
	}

	contextInfo = append(contextInfo, "")
	contextInfo = append(contextInfo, "【重要提示】由于当前在 Kubernetes 容器环境中运行，建议：")
	contextInfo = append(contextInfo, "  1. 优先使用可用的 MCP 工具进行 K8s 相关操作")
	contextInfo = append(contextInfo, "  2. 生成的命令应考虑容器环境的限制和特性")
	contextInfo = append(contextInfo, "  3. 对于跨命名空间的操作，需要检查权限")

	// 添加命名空间使用建议
	if currentNamespace != "" {
		contextInfo = append(contextInfo, fmt.Sprintf("  4. 使用MCP工具时，如未指定namespace参数，可以使用当前命名空间'%s'或使用'all'查询所有命名空间", currentNamespace))
	} else {
		contextInfo = append(contextInfo, "  4. 使用MCP工具时，如需跨命名空间查询，建议使用namespace='all'")
	}

	return strings.Join(contextInfo, "\n")
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
		"baseInfo":          getBaseInfo(enableExplain, enableExtendParams),
		"workFlow":          getWorkFlow(enablePlatformPerception, enableWorkUserAndDir, enableExplain, enableExtendParams),
		"workPlatform":      getWorkPlatform(enablePlatformPerception),
		"workUserAndDir":    getWorkUserAndDir(enableWorkUserAndDir),
		"k8sContext":        getK8sContext(),
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
		"baseInfo":          getBaseInfo(enableExplain, enableExtendParams),
		"workFlow":          getWorkFlow(enablePlatformPerception, enableWorkUserAndDir, enableExplain, enableExtendParams),
		"workPlatform":      getWorkPlatform(enablePlatformPerception),
		"workUserAndDir":    getWorkUserAndDir(enableWorkUserAndDir),
		"k8sContext":        getK8sContext(),
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
