package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadSpongeModule(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock go command uses a shell script")
	}
	const revision = "v0.0.0-20260923000000-0123456789ab"
	want := spongeModule{Path: "github.com/Eric-Guo/sponge", Version: revision, Dir: "/custom/cache/github.com/!eric-!guo/sponge@" + revision}
	for _, tt := range []struct {
		name      string
		module    spongeModule
		wantError bool
	}{
		{"branch resolves to pseudo-version", want, false},
		{"download error", spongeModule{Error: "unknown revision"}, true},
		{"missing directory", spongeModule{Path: want.Path, Version: revision}, true},
		{"wrong module", spongeModule{Path: "example.com/other", Version: revision, Dir: want.Dir}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			data, err := json.Marshal(tt.module)
			require.NoError(t, err)
			script := "#!/bin/sh\n[ \"$*\" = 'mod download -json github.com/Eric-Guo/sponge@thruster_generate' ] || exit 1\ncat <<'JSON'\n" + string(data) + "\nJSON\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0755))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			got, err := downloadSpongeModule(latestVersion)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}

func TestInternalPluginsUseTemplateRevision(t *testing.T) {
	previous := templateVersion
	t.Cleanup(func() { templateVersion = previous })
	templateVersion = "v0.0.0-20260923000000-0123456789ab"
	for _, name := range []string{"protoc-gen-go-gin", "protoc-gen-go-rpc-tmpl", "protoc-gen-json-field"} {
		require.Equal(t, "github.com/Eric-Guo/sponge/cmd/"+name+"@"+templateVersion, adaptInternalCommand(name, installPluginCommands[name]))
	}
	require.Equal(t, installPluginCommands["protoc-gen-go"], adaptInternalCommand("protoc-gen-go", installPluginCommands["protoc-gen-go"]))
}
