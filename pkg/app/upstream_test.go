package app

import (
	"errors"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
)

func TestSplitCommandLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		command   string
		want      []string
		expectErr bool
	}{
		{
			name:    "simple command",
			command: "/bin/echo",
			want:    []string{"/bin/echo"},
		},
		{
			name:    "command with arguments",
			command: "/usr/bin/env bash -c \"echo hello world\"",
			want:    []string{"/usr/bin/env", "bash", "-c", "echo hello world"},
		},
		{
			name:    "handles escaped space",
			command: "/bin/echo some\\ value",
			want:    []string{"/bin/echo", "some value"},
		},
		{
			name:      "unterminated quote",
			command:   "/bin/echo \"unterminated",
			expectErr: true,
		},
		{
			name:      "unterminated escape",
			command:   "/bin/echo trailing\\",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := splitCommandLine(tc.command)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\nwant: %#v\ngot:  %#v", tc.want, got)
			}
		})
	}
}

func TestNormalizeCommand(t *testing.T) {
	t.Parallel()

	rbenvCommand := "/opt/rbenv/bin/rbenv exec bundle exec --keep-file-descriptors puma -C /srv/app/puma.rb"
	base, args, err := normalizeCommand(rbenvCommand, nil)
	if err != nil {
		t.Fatalf("normalizeCommand returned error: %v", err)
	}

	expectedBase := "/opt/rbenv/bin/rbenv"
	expectedArgs := []string{"exec", "bundle", "exec", "--keep-file-descriptors", "puma", "-C", "/srv/app/puma.rb"}

	if base != expectedBase {
		t.Fatalf("unexpected base command, want %q got %q", expectedBase, base)
	}

	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("unexpected arguments\nwant: %#v\ngot:  %#v", expectedArgs, args)
	}

	_, _, err = normalizeCommand("  ", nil)
	if !errors.Is(err, errEmptyCommand) {
		t.Fatalf("expected upstream command is empty error, got %v", err)
	}
}

func TestUpstreamServerLifecycle(t *testing.T) {
	if _, err := logger.Init(logger.WithLevel("error")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix process signals")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	server := NewUpstreamServer(UpstreamConfig{Enabled: true, Command: strconv.Quote(binary), Args: []string{"-test.run=^TestUpstreamHelper$"}, Env: map[string]string{"SPONGE_UPSTREAM_HELPER": "1"}})
	result := make(chan error, 1)
	go func() { result <- server.Start() }()
	defer server.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for {
		server.mu.Lock()
		started := server.cmd != nil
		server.mu.Unlock()
		if started {
			break
		}
		select {
		case err := <-result:
			t.Fatalf("process exited before start: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("upstream did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("upstream did not stop")
	}
	// A concurrent application shutdown must also prevent a later startup.
	stopped := NewUpstreamServer(UpstreamConfig{Enabled: true, Command: "does-not-exist"})
	if err := stopped.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := stopped.Start(); err != nil {
		t.Fatal(err)
	}
}

func TestUpstreamHelper(t *testing.T) {
	if os.Getenv("SPONGE_UPSTREAM_HELPER") != "1" {
		return
	}
	for {
		time.Sleep(time.Second)
	}
}

func TestUpstreamEnvironmentAndEmptyArguments(t *testing.T) {
	command, args, err := normalizeCommand(`echo "" 'two words'`, []string{"last"})
	if err != nil {
		t.Fatal(err)
	}
	if command != "echo" || !reflect.DeepEqual(args, []string{"", "two words", "last"}) {
		t.Fatalf("unexpected command: %q %#v", command, args)
	}
	t.Setenv("PORT", "inherited")
	server := NewUpstreamServer(UpstreamConfig{TargetPort: 4321, Env: map[string]string{"RAILS_ENV": "test"}})
	env := strings.Join(server.buildEnv(), "\n")
	if !strings.Contains(env, "PORT=4321") || !strings.Contains(env, "RAILS_ENV=test") {
		t.Fatal(env)
	}
	server.cfg.TargetBindSocket = "/tmp/upstream.sock"
	if strings.Contains(strings.Join(server.buildEnv(), "\n"), "PORT=4321") {
		t.Fatal("socket configuration overrode PORT")
	}
}
