package action

import (
	"context"
	"hi-ai-cli/hiai"
	"hi-ai-cli/hiai/manual"
	"hi-ai-cli/logger"
	"hi-ai-cli/setup"
	"strings"

	"github.com/urfave/cli/v3"
)

// NewHiManualAction 创建 manual action执行
func NewHiManualAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		i18n := setup.GetI18n()
		cmdName := cmd.String("cmd")
		question := strings.Join(cmd.Args().Slice(), " ")
		answerConfig := setup.GetConfig().AnswerConfig
		messages := manual.CreateOnceMessagesFromTemplate(cmdName, question, answerConfig.EnableExplain, answerConfig.EnableExtendParams, answerConfig.EnablePlatformPerception, answerConfig.EnableWorkUserAndDir)
		cm := hiai.CreateOpenAIChatModel(ctx)
		streamResult := hiai.Stream(ctx, cm, messages)
		_, _, err := hiai.ReportStream(streamResult)
		if err != nil {
			logger.Errorf("ReportStream failed %v", err)
		}
		logger.Debug(i18n.Exit)
		return nil
	}
}
