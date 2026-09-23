package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/gobash"
	"github.com/Eric-Guo/sponge/pkg/gofile"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

// UpgradeCommand upgrade sponge binaries
func UpgradeCommand() *cobra.Command {
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade sponge version",
		Long:  "Upgrade sponge version.",
		Example: color.HiBlackString(`  # Upgrade to latest version
  sponge upgrade

  # Upgrade to specified version
  sponge upgrade --version=thruster_generate`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetVersion == "" {
				targetVersion = latestVersion
			}
			ver, err := runUpgrade(targetVersion)
			if err != nil {
				return err
			}
			fmt.Printf("upgraded version to %s successfully.\n", ver)
			return nil
		},
	}

	cmd.Flags().StringVarP(&targetVersion, "version", "v", latestVersion, "upgrade sponge version")
	return cmd
}

func runUpgrade(targetVersion string) (string, error) {
	module, err := downloadSpongeModule(targetVersion)
	if err != nil {
		return "", err
	}
	targetVersion = module.Version
	runningTip := "sponge binary upgrading "
	finishTip := "sponge binary upgraded " + installedSymbol
	failTip := "sponge binary upgrade failed " + lackSymbol
	p := utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	err = runUpgradeCommand(targetVersion)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)

	runningTip = "template code upgrading "
	finishTip = "template code upgraded " + installedSymbol
	failTip = "template code upgrade failed " + lackSymbol
	p = utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	ver, err := copyTemplateModule(module)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)

	runningTip = "built-in plugins upgrading "
	finishTip = "built-in plugins upgraded " + installedSymbol
	failTip = "built-in plugins upgrade failed " + lackSymbol
	p = utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	err = updateSpongeInternalPlugin(ver)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)
	return ver, nil
}

func runUpgradeCommand(targetVersion string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*3) //nolint
	spongeVersion := "github.com/Eric-Guo/sponge/cmd/sponge@" + targetVersion
	result := gobash.Run(ctx, "go", "install", spongeVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}
	return nil
}

// spongeModule is the canonical version and cache directory reported by Go.
// Asking Go avoids assuming GOPATH layout or module-path case escaping.
type spongeModule struct {
	Path    string
	Version string
	Dir     string
	Error   string
}

func downloadSpongeModule(targetVersion string) (spongeModule, error) {
	var module spongeModule
	result, err := gobash.Exec("go", "mod", "download", "-json", "github.com/Eric-Guo/sponge@"+targetVersion)
	if err != nil {
		return module, fmt.Errorf("download sponge module: %w", err)
	}
	if err = json.Unmarshal(result, &module); err != nil {
		return module, fmt.Errorf("decode sponge module: %w", err)
	}
	if module.Error != "" {
		return module, fmt.Errorf("download sponge module: %s", module.Error)
	}
	if module.Path != "github.com/Eric-Guo/sponge" || module.Version == "" || module.Dir == "" {
		return module, fmt.Errorf("incomplete sponge module information")
	}
	return module, nil
}

// Copy templates from the same resolved revision as the binary and plugins.
func copyTemplateModule(module spongeModule) (string, error) {
	srcDir := adaptPathDelimiter(module.Dir)
	targetDir := adaptPathDelimiter(GetSpongeDir() + "/.sponge")

	err := executeCommand("rm", "-rf", targetDir)
	if err != nil {
		return "", err
	}
	err = executeCommand("cp", "-rf", srcDir, targetDir)
	if err != nil {
		return "", err
	}
	err = executeCommand("chmod", "-R", "744", targetDir)
	if err != nil {
		return "", err
	}
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/sponge")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-go-gin")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-go-rpc-tmpl")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-json-field")
	_ = executeCommand("rm", "-rf", targetDir+"/pkg")
	_ = executeCommand("rm", "-rf", targetDir+"/test")
	_ = executeCommand("rm", "-rf", targetDir+"/assets")

	versionNum := module.Version
	err = os.WriteFile(versionFile, []byte(versionNum), 0644)
	if err != nil {
		return "", err
	}

	return versionNum, nil
}

func executeCommand(name string, args ...string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*30) //nolint
	result := gobash.Run(ctx, name, args...)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return fmt.Errorf("execute command failed, %v", result.Err)
	}
	return nil
}

func adaptPathDelimiter(filePath string) string {
	if gofile.IsWindows() {
		filePath = strings.ReplaceAll(filePath, "/", "\\")
	}
	return filePath
}

func updateSpongeInternalPlugin(targetVersion string) error {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Minute) //nolint
	genGinVersion := "github.com/Eric-Guo/sponge/cmd/protoc-gen-go-gin@" + targetVersion
	result := gobash.Run(ctx, "go", "install", genGinVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}

	ctx, _ = context.WithTimeout(context.Background(), 3*time.Minute) //nolint
	genRPCVersion := "github.com/Eric-Guo/sponge/cmd/protoc-gen-go-rpc-tmpl@" + targetVersion
	result = gobash.Run(ctx, "go", "install", genRPCVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}

	ctx, _ = context.WithTimeout(context.Background(), 3*time.Minute) //nolint
	genJSONVersion := "github.com/Eric-Guo/sponge/cmd/protoc-gen-json-field@" + targetVersion
	result = gobash.Run(ctx, "go", "install", genJSONVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}

	return nil
}
