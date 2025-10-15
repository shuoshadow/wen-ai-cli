package action

import (
	"context"
	"fmt"
	"strings"
	"wen-ai-cli/common"
	"wen-ai-cli/execute"
	"wen-ai-cli/logger"
	"wen-ai-cli/mcp"
	"wen-ai-cli/setup"
	"wen-ai-cli/wenai"
	"wen-ai-cli/wenai/chat"

	"github.com/cloudwego/eino/schema"
	"github.com/urfave/cli/v3"
)

// NewWenOnceAction 创建 wen once action执行
func NewWenOnceAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		i18n := setup.GetI18n()
		question := strings.Join(cmd.Args().Slice(), " ")
		answerConfig := setup.GetConfig().AnswerConfig
		messages := chat.CreateOnceMessagesFromTemplate(question, answerConfig.EnableExplain, answerConfig.EnableExtendParams, answerConfig.EnablePlatformPerception, answerConfig.EnableWorkUserAndDir)
		cm := wenai.CreateOpenAIChatModel(ctx)
		streamResult := wenai.Stream(ctx, cm, messages)
		fullMessage, hiddenParams, err := wenai.ReportStream(streamResult)
		if err != nil {
			logger.Errorf("ReportStream failed %v", err)
		}

		// 检查是否有工具调用
		if hiddenParams.HasToolCalls() {
			mcpManager := setup.GetMCPManager()
			if mcpManager != nil {
				fmt.Println("\n[正在调用工具查询实时数据...]")

				// 执行所有工具调用
				var toolResults []string
				for _, toolCall := range hiddenParams.ToolCalls {
					logger.Debugf("Calling tool: %s with args: %v", toolCall.Name, toolCall.Arguments)

					result, err := mcpManager.CallTool(ctx, &mcp.ToolCallRequest{
						Name:      toolCall.Name,
						Arguments: toolCall.Arguments,
					})

					if err != nil {
						logger.Errorf("Tool call failed: %v", err)
						toolResults = append(toolResults, fmt.Sprintf("工具 %s 调用失败: %v", toolCall.Name, err))
						continue
					}

					// 提取工具返回的文本内容
					if len(result.Content) > 0 {
						toolResults = append(toolResults, fmt.Sprintf("工具 %s 返回结果:\n%s", toolCall.Name, result.Content[0].Text))
					}
				}

				// 将工具结果添加到消息历史，让 AI 基于真实数据重新生成答案
				if len(toolResults) > 0 {
					messages = append(messages, fullMessage)
					messages = append(messages, &schema.Message{
						Role:    "user",
						Content: fmt.Sprintf("工具调用结果：\n%s\n\n请基于以上真实数据，重新生成完整的回答。", strings.Join(toolResults, "\n\n")),
					})

					fmt.Println("\n[基于查询结果生成答案...]")
					streamResult = wenai.Stream(ctx, cm, messages)
					fullMessage, hiddenParams, err = wenai.ReportStream(streamResult)
					if err != nil {
						logger.Errorf("ReportStream failed %v", err)
					}
				}
			}
		}

		fmt.Println("--------------------------------")
		if hiddenParams.HasParameters() {
			// 如果存在需要填充的参数，则提示用户，说明可以填充参数
			result, err := execute.Prompt(i18n.SelectOperation, []string{i18n.FillParamsAndRun, i18n.AdjustAndRun, i18n.Exit})
			if err != nil {
				logger.Errorf("Prompt failed %v", err)
				return nil
			}
			logger.Debugf(i18n.YourChoice, result)
			if result == i18n.FillParamsAndRun {
				shellCode, shouldExecute := common.HandleParamsCompletion(hiddenParams)
				if shouldExecute {
					execute.ExecuteScript(shellCode)
				} else {
					return nil
				}
			} else if result == i18n.AdjustAndRun {
				script, shouldExecute := common.HandleScriptAdjustment(hiddenParams.ShellCode)
				if shouldExecute {
					execute.ExecuteScript(script)
				} else {
					return nil
				}
			} else {
				logger.Debug(i18n.Exit)
			}
		} else {
			if hiddenParams.ShellCode == "" {
				// 如果脚本为空，则提示用户，说明无法解析答案
				// 按照微调脚本进行处理
				result, err := execute.Prompt(i18n.SelectOperation, []string{i18n.Exit})
				if err != nil {
					logger.Errorf("Prompt failed %v", err)
					return nil
				}
				logger.Debugf(i18n.YourChoice, result)
				logger.Debug(i18n.Exit)
			} else {
				// 如果脚本不为空，则提示用户，说明可以执行
				logger.Debug(i18n.CanExecute)
				result, err := execute.Prompt(i18n.SelectOperation, []string{i18n.RunNow, i18n.AdjustAndRun, i18n.Exit})
				if err != nil {
					logger.Errorf("Prompt failed %v", err)
					return nil
				}
				logger.Debugf(i18n.YourChoice, result)
				if result == i18n.RunNow {
					execute.ExecuteScript(hiddenParams.ShellCode)
				} else if result == i18n.AdjustAndRun {
					script, shouldExecute := common.HandleScriptAdjustment(hiddenParams.ShellCode)
					if shouldExecute {
						execute.ExecuteScript(script)
					}
				} else {
					logger.Debug(i18n.Exit)
				}
			}
		}
		return nil
	}
}
