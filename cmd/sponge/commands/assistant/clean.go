package assistant

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/gofile"
)

// CleanUpAssistantCode clean up all assistant generated code
func CleanUpAssistantCode() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up all assistant generated code",
		Long:  "Clean up all assistant generated code.",
		Example: color.HiBlackString(`  # Clean up all assistant generated code in current directory
  sponge assistant clean --dir=/path/to/directory`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			assistantTypes := []string{typeDeepSeek, typeGemini, typeChatGPT}
			for _, assistantType := range assistantTypes {
				deleteFiles, err := gofile.ListFiles(dir, gofile.WithSuffix(getAssistantSuffixed(assistantType)))
				if err != nil {
					return err
				}
				deleteGenFiles(deleteFiles, assistantType)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "input directory")
	_ = cmd.MarkFlagRequired("dir")

	return cmd
}
