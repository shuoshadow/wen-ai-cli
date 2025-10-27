package cmd

import (
	"hi-ai-cli/action"
	"hi-ai-cli/setup"

	"github.com/gookit/i18n"
	"github.com/urfave/cli/v3"
)

// NewChatCmd 创建 chat 命令
func NewChatCmd() *cli.Command {
	return &cli.Command{
		Name:   setup.ChatCmd,
		Usage:  i18n.Dtr("chatMode"),
		Action: action.NewHiChatAction(),
	}
}
