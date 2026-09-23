package commands

import (
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/cmd/sponge/commands/patch"
)

// PatchCommand patch server code
func PatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "patch",
		Short:         "Patch the generated code",
		Long:          `Patch the generated code.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		patch.DeleteJSONOmitemptyCommand(),
		patch.GenerateDBInitCommand(),
		patch.GenTypesPbCommand(),
		patch.CopyProtoCommand(),
		patch.CopyThirdPartyProtoCommand(),
		patch.CopyGOModCommand(),
		patch.ModifyDuplicateErrorCodeNumCommand(),
		patch.ModifyDuplicateErrorCodeOffsetCommand(),
		patch.AdaptMonoRepoCommand(),
		patch.ModifyProtoPackageCommand(),
	)

	return cmd
}
