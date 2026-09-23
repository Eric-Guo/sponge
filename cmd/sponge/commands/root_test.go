package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionDoesNotDependOnTemplates(t *testing.T) {
	previous := versionFile
	versionFile = filepath.Join(t.TempDir(), "version")
	t.Cleanup(func() { versionFile = previous })
	for _, cached := range []string{"", "v1.16.1\n", "v0.0.0-20260923000000-0123456789ab\n"} {
		if cached != "" {
			require.NoError(t, os.WriteFile(versionFile, []byte(cached), 0600))
		}
		cmd := NewRootCMD()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetArgs([]string{"--version"})
		require.NoError(t, cmd.Execute())
		require.Equal(t, "sponge version v1.15.1 https://github.com/Eric-Guo/sponge fork\n", output.String())
	}
}
