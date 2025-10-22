package wenai

import (
	"encoding/json"
	"io"
	"log"
	"regexp"
	"strings"
	"wen-ai-cli/common"
	"wen-ai-cli/model"
	"wen-ai-cli/setup"

	"github.com/cloudwego/eino/schema"
)

func ReportStream(sr *schema.StreamReader[*schema.Message]) (*schema.Message, *model.HiddenParams, error) {
	defer sr.Close()

	// 创建使用自定义内容颜色的打印器
	printer := common.NewStreamPrinterWithAllOptions(false, true, setup.CliName, setup.CliVersion)

	i := 0
	result := &model.HiddenParams{}
	shellCode := ""
	fullContentBuilder := strings.Builder{}
	for {
		message, err := sr.Recv()
		if err == io.EOF {
			// 处理最后一段
			printer.Print("\n")
			printer.Flush()

			fullContent := fullContentBuilder.String()

			// 解析代码块 - 支持多种格式：```code、```bash、```sh、```shell 等
			re := regexp.MustCompile("(?s)```(?:code|bash|sh|shell|zsh)\\s*(.*?)```")
			matches := re.FindAllStringSubmatch(fullContent, -1)
			if len(matches) > 0 {
				// 提取第一个代码块作为主要命令
				// 因为通常第一个是主要命令，后续的可能是备选方案或安装步骤
				firstMatch := matches[0]
				if len(firstMatch) > 1 {
					shellCode = strings.TrimSpace(firstMatch[1])
				}
				// 如果有多个代码块，记录日志供调试
				if len(matches) > 1 {
					log.Printf("Found %d code blocks, using the first one. All blocks: %v", len(matches), matches)
				}
			}
			if shellCode != "" {
				result.ShellCode = shellCode
				// 解析shellCode,<下载文件的URL,url>序列化成hideParams
				re := regexp.MustCompile(`<([\p{Han}a-zA-Z0-9]+),(\w+)>`)
				matches := re.FindAllStringSubmatch(shellCode, -1)
				for _, match := range matches {
					paramName := match[1]
					paramType := match[2]
					result.NeedFillParams = append(result.NeedFillParams, model.ParamInfo{
						Param: paramName,
						Type:  paramType,
					})
				}
			}

			// 解析工具调用
			toolCallRegex := regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
			toolCallMatches := toolCallRegex.FindAllStringSubmatch(fullContent, -1)
			for _, match := range toolCallMatches {
				if len(match) > 1 {
					toolCallJSON := strings.TrimSpace(match[1])
					var toolCall model.ToolCall
					if err := json.Unmarshal([]byte(toolCallJSON), &toolCall); err == nil {
						result.ToolCalls = append(result.ToolCalls, toolCall)
					}
				}
			}

			fullMessage := &schema.Message{
				Role:    "assistant",
				Content: fullContent,
			}
			return fullMessage, result, nil
		}
		if err != nil {
			log.Fatalf("recv failed: %v", err)
		}
		content := message.Content
		fullContentBuilder.WriteString(content)
		printer.Print(content)
		i++
	}
}
