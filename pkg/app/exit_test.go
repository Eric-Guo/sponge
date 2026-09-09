package app

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunWithExitCodeForCompletedChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a Unix shell")
	}
	for _, tt := range []struct {
		command string
		code    int
	}{
		{"exit 0", 0}, {"exit 7", 7}, {"kill -TERM $$", 143}, {"kill -INT $$", 130},
	} {
		t.Run(tt.command, func(t *testing.T) {
			child := NewUpstreamServer(UpstreamConfig{Enabled: true, Command: "/bin/sh", Args: []string{"-c", tt.command}})
			closed := false
			a := New([]IServer{child}, []Close{child.Stop, func() error { closed = true; return nil }})
			require.Equal(t, tt.code, a.runWithSignals(make(chan os.Signal)))
			require.True(t, closed)
		})
	}
}

func TestParentSignalIsRelayedAndChildStatusReturned(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix signals")
	}
	child := NewUpstreamServer(UpstreamConfig{Enabled: true, Command: "/bin/sleep", Args: []string{"60"}, StopSignal: "SIGTERM"})
	signals := make(chan os.Signal, 1)
	result := make(chan int, 1)
	go func() { result <- New([]IServer{child}, []Close{child.Stop}).runWithSignals(signals) }()
	t.Cleanup(func() { _ = child.Stop() })
	require.Eventually(t, func() bool {
		child.mu.Lock()
		defer child.mu.Unlock()
		return child.cmd != nil
	}, 5*time.Second, time.Millisecond)
	signals <- syscall.SIGINT
	select {
	case code := <-result:
		require.Equal(t, 130, code, "SIGINT must not be replaced with the configured SIGTERM")
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor did not terminate")
	}
}
