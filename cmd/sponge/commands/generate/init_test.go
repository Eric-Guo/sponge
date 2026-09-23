package generate

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestVersionWithoutInstalledTemplates(t *testing.T) {
	previousDir, previousArgs := SpongeDir, os.Args
	SpongeDir = t.TempDir() + "/missing"
	t.Cleanup(func() { SpongeDir, os.Args = previousDir, previousArgs })
	for _, flag := range []string{"--version", "-v", "--help"} {
		os.Args = []string{"sponge", flag}
		require.NoError(t, Init())
	}
	os.Args = []string{"sponge", "web"}
	require.ErrorContains(t, Init(), "sponge init")
}
