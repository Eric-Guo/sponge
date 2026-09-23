package generate

import (
	"embed"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Eric-Guo/sponge/pkg/gofile"
	"github.com/Eric-Guo/sponge/pkg/replacer"
)

const warnSymbol = "⚠ "

func init() {
	rand.Seed(time.Now().UnixNano()) //nolint
}

// Replacers replacer name
var Replacers = map[string]replacer.Replacer{}

// SpongeDir sponge directory
var SpongeDir = getHomeDir() + gofile.GetPathDelimiter() + ".sponge"

// Template information
type Template struct {
	Name     string
	FS       embed.FS
	FilePath string
}

// Init initializing the template
func Init() error {
	// determine if the template file exists, if not, prompt to initialize first
	if !gofile.IsExists(SpongeDir) {
		if isShowCommand() {
			return nil
		}
		return fmt.Errorf("%s not yet initialized, run the command \"sponge init\"", warnSymbol)
	}

	var err error
	if _, ok := Replacers[TplNameSponge]; ok {
		panic(fmt.Sprintf("template name \"%s\" already exists", TplNameSponge))
	}
	Replacers[TplNameSponge], err = replacer.New(SpongeDir)
	if err != nil {
		return err
	}

	return nil
}

// InitFS initializing th FS templates
func InitFS(name string, filepath string, fs embed.FS) {
	var err error
	if _, ok := Replacers[name]; ok {
		panic(fmt.Sprintf("template name \"%s\" already exists", name))
	}
	Replacers[name], err = replacer.NewFS(filepath, fs)
	if err != nil {
		panic(err)
	}
}

func isShowCommand() bool {
	l := len(os.Args)

	// sponge
	if l == 1 {
		return true
	}

	// sponge init or sponge -h
	if l == 2 {
		if os.Args[1] == "init" || os.Args[1] == "upgrade" || os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "-v" || os.Args[1] == "--version" {
			return true
		}
		return false
	}
	if l > 2 {
		return os.Args[1] == "init" || os.Args[1] == "upgrade"
	}

	return false
}

func getHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("can't get home directory'")
		return ""
	}

	return dir
}
