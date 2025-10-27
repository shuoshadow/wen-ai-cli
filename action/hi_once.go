package action

import (
	"context"
	"fmt"
	"hi-ai-cli/common"
	"hi-ai-cli/execute"
	"hi-ai-cli/hiai"
	"hi-ai-cli/hiai/chat"
	"hi-ai-cli/logger"
	"hi-ai-cli/mcp"
	"hi-ai-cli/setup"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/urfave/cli/v3"
)

// NewHiOnceAction 创建 hi once action执行
func NewHiOnceAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		i18n := setup.GetI18n()
		question := strings.Join(cmd.Args().Slice(), " ")
		answerConfig := setup.GetConfig().AnswerConfig
		messages := chat.CreateOnceMessagesFromTemplate(question, answerConfig.EnableExplain, answerConfig.EnableExtendParams, answerConfig.EnablePlatformPerception, answerConfig.EnableWorkUserAndDir)
		cm := hiai.CreateOpenAIChatModel(ctx)
		streamResult := hiai.Stream(ctx, cm, messages)
		fullMessage, hiddenParams, err := hiai.ReportStream(streamResult)
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
						errorMsg := fmt.Sprintf("❌ 工具 %s 调用失败: %v", toolCall.Name, err)
						toolResults = append(toolResults, errorMsg)
						fmt.Println(errorMsg)
						continue
					}

					// 添加详细日志
					logger.Debugf("Tool %s returned %d content items, isError: %v", toolCall.Name, len(result.Content), result.IsError)

					// 提取工具返回的文本内容并展示给用户
					if len(result.Content) > 0 {
						var contentText string
						for i, content := range result.Content {
							logger.Debugf("Content[%d]: type=%s, text_len=%d", i, content.Type, len(content.Text))
							if content.Text != "" {
								contentText += content.Text + "\n"
							}
						}

						if contentText != "" {
							fmt.Printf("\n✓ 工具 [%s] 返回结果:\n", toolCall.Name)
							fmt.Println("─────────────────────────────────")
							fmt.Println(contentText)
							fmt.Println("─────────────────────────────────")
							toolResults = append(toolResults, fmt.Sprintf("工具 %s 返回:\n%s", toolCall.Name, contentText))
						} else {
							noDataMsg := fmt.Sprintf("⚠️  工具 %s 未返回有效文本数据 (有 %d 个内容项，但都不是文本)", toolCall.Name, len(result.Content))
							fmt.Println(noDataMsg)
							logger.Debugf("Content items: %+v", result.Content)
							toolResults = append(toolResults, noDataMsg)
						}
					} else {
						noContentMsg := fmt.Sprintf("⚠️  工具 %s 返回为空", toolCall.Name)
						fmt.Println(noContentMsg)
						toolResults = append(toolResults, noContentMsg)
					}
				}

				// 将工具结果添加到消息历史，让 AI 基于真实数据重新生成答案
				if len(toolResults) > 0 {
					messages = append(messages, fullMessage)
					messages = append(messages, &schema.Message{
						Role:    "user",
						Content: fmt.Sprintf("以上是工具调用的真实结果：\n%s\n\n请基于这些真实数据，直接回答用户的问题。重要要求：\n1. 直接展示和解释工具返回的数据，用清晰的格式整理展示\n2. **不要生成任何 kubectl 命令**，因为用户本地没有 kubectl\n3. **不要包含「待执行脚本」部分**，只需要概述和数据展示\n4. 如果工具返回了足够的信息，直接用这些信息回答问题\n5. 如果工具调用失败，说明失败原因，但仍然不要生成 kubectl 命令", strings.Join(toolResults, "\n\n")),
					})

					fmt.Println("\n[基于查询结果生成最终答案...]")
					streamResult = hiai.Stream(ctx, cm, messages)
					fullMessage, hiddenParams, err = hiai.ReportStream(streamResult)
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
