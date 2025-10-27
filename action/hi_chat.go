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

// NewHiChatAction 创建 chat action执行
func NewHiChatAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// 获取配置信息
		answerConfig := setup.GetConfig().AnswerConfig
		// 获取语言包
		i18n := setup.GetI18n()
		// 初始化聊天历史记录
		chatHistory := []*schema.Message{}
		// 将命令行参数拼接为问题
		question := strings.Join(cmd.Args().Slice(), " ")
		questionTimes := 0
		if question == "" {
			firstQuestion, err := execute.InputString(i18n.UserInput)
			if err != nil {
				logger.Errorf("Prompt failed %v", err)
				return nil
			}

			// 记录用户输入
			logger.Debugf(i18n.UserInput, firstQuestion)
			question = firstQuestion
		}

		// 进入主循环，持续与用户交互
		for {
			questionTimes++
			// 打印对话轮次
			execute.PrintQuestionTimes(question, questionTimes)
			// 创建聊天消息模板

			messages := chat.CreateMoreMessagesFromTemplate(question, chatHistory, answerConfig.EnableExplain, answerConfig.EnableExtendParams, answerConfig.EnablePlatformPerception, answerConfig.EnableWorkUserAndDir)
			// 创建OpenAI聊天模型
			cm := hiai.CreateOpenAIChatModel(ctx)
			// 获取流式处理结果
			streamResult := hiai.Stream(ctx, cm, messages)
			// 解析流式结果，获取完整消息和隐藏参数
			fullMessage, hiddenParams, err := hiai.ReportStream(streamResult)
			if err != nil {
				logger.Errorf("ReportStream failed %v", err)
			}

			// 检查是否有工具调用
			if hiddenParams.HasToolCalls() {
				mcpManager := setup.GetMCPManager()
				if mcpManager != nil {
					logger.Info("[正在调用工具查询实时数据...]")

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
							logger.Info(errorMsg)
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
								logger.Infof("\n✓ 工具 [%s] 返回结果:", toolCall.Name)
								logger.Info("─────────────────────────────────")
								logger.Info(contentText)
								logger.Info("─────────────────────────────────")
								toolResults = append(toolResults, fmt.Sprintf("工具 %s 返回:\n%s", toolCall.Name, contentText))
							} else {
								noDataMsg := fmt.Sprintf("⚠️  工具 %s 未返回有效文本数据 (有 %d 个内容项，但都不是文本)", toolCall.Name, len(result.Content))
								logger.Info(noDataMsg)
								logger.Debugf("Content items: %+v", result.Content)
								toolResults = append(toolResults, noDataMsg)
							}
						} else {
							noContentMsg := fmt.Sprintf("⚠️  工具 %s 返回为空", toolCall.Name)
							logger.Info(noContentMsg)
							toolResults = append(toolResults, noContentMsg)
						}
					}

					// 将工具结果添加到消息历史，让 AI 基于真实数据重新生成答案
					if len(toolResults) > 0 {
						messages = append(messages, fullMessage)
						messages = append(messages, &schema.Message{
							Role:    "user",
							Content: fmt.Sprintf("以上是工具调用的真实结果：\n%s\n\n请基于这些真实数据，生成一个完整、准确的答案。重要要求：\n1. 如果工具返回了具体数据，请直接基于数据回答用户问题，给出一个最佳的kubectl命令\n2. 只提供一个代码块，不要提供多个备选方案或安装步骤\n3. 如果工具返回了足够的数据，直接使用这些数据，不需要再用kubectl命令查询\n4. 如果工具调用失败或数据不足，再给出kubectl命令作为替代\n5. 必须严格按照回答格式输出", strings.Join(toolResults, "\n\n")),
						})

						logger.Info("\n[基于查询结果生成最终答案...]")
						streamResult = hiai.Stream(ctx, cm, messages)
						fullMessage, hiddenParams, err = hiai.ReportStream(streamResult)
						if err != nil {
							logger.Errorf("ReportStream failed %v", err)
						}
					}
				}
			}

			// 打印帮助信息
			var helpPrinter = execute.PrintHelp()
			inputQuetion, err := execute.InputString(i18n.UserInput)
			helpPrinter.Clear0()
			if err != nil {
				logger.Errorf("Prompt failed %v", err)
				return nil
			}

			// 记录用户输入
			logger.Debugf(i18n.UserInputFormat, inputQuetion)

			// 处理退出命令
			if inputQuetion == "q" || inputQuetion == "Q" {
				logger.Debug(i18n.Exit)
				return nil
			}

			// 处理功能命令
			if inputQuetion == "f" || inputQuetion == "F" {
				// 如果有需要填充的参数
				if hiddenParams.HasParameters() {
					// 创建操作选择提示
					result, err := execute.Prompt(i18n.SelectOperation, []string{i18n.FillParamsAndRun, i18n.AdjustAndRun, i18n.Exit})
					if err != nil {
						logger.Errorf("Prompt failed %v", err)
						return nil
					}

					// 记录用户选择
					logger.Debugf(i18n.YourChoice, result)

					// 根据选择执行相应操作
					if result == i18n.FillParamsAndRun {
						shellCode, shouldExecute := common.HandleParamsCompletion(hiddenParams)
						if shouldExecute {
							execute.ExecuteScript(shellCode)
						}
					} else if result == i18n.AdjustAndRun {
						script, shouldExecute := common.HandleScriptAdjustment(hiddenParams.ShellCode)
						if shouldExecute {
							execute.ExecuteScript(script)
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
						if result == i18n.AdjustAndRun {
							script, shouldExecute := common.HandleScriptAdjustment(hiddenParams.ShellCode)
							if shouldExecute {
								execute.ExecuteScript(script)
							}
						} else {
							logger.Debug(i18n.Exit)
						}
					} else {
						// 如果脚本不为空，则提示用户，说明可以执行
						logger.Debug(i18n.CanExecute)
						// 创建操作选择提示
						result, err := execute.Prompt(i18n.SelectOperation, []string{i18n.RunNow, i18n.AdjustAndRun, i18n.Exit})
						if err != nil {
							logger.Errorf("Prompt failed %v", err)
							return nil
						}

						// 记录用户选择
						logger.Debugf(i18n.YourChoice, result)

						// 根据选择执行相应操作
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

			// 其他情况，继续对话，并更新聊天历史记录
			// 保留最近10条消息
			chatHistory = messages[max(1, len(messages)-10):]
			// 添加最新消息到历史记录
			chatHistory = append(chatHistory, fullMessage)
			// 更新问题为最新输入
			question = inputQuetion
		}
	}
}
