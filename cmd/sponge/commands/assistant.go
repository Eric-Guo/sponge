package commands

import (
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/cmd/sponge/commands/assistant"
)

// AssistantCommand AI assistant command
func AssistantCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "assistant",
		Short:         "AI assistants, support generation and merging of Go code, chat, and more",
		Long:          "AI assistant, support generation and merging of Go code, chat, and more.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		assistant.ChatCommand(),
		assistant.GenerateCommand(),
		assistant.MergeAssistantCode(),
		assistant.CleanUpAssistantCode(),
	)
	return cmd
}
