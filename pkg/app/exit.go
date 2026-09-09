package app

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-dev-frame/sponge/pkg/prof"
)

type serviceResult struct {
	server IServer
	err    error
}

// RunWithExitCode runs services with process-supervisor semantics: any completed
// service triggers cleanup, including an upstream that exits successfully. It
// returns the child's status after cleanup so main can pass it to os.Exit.
// Run retains the existing framework lifecycle for applications that prefer it.
func (a *App) RunWithExitCode() int {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGTRAP)
	defer signal.Stop(signals)
	return a.runWithSignals(signals)
}

func (a *App) runWithSignals(signals <-chan os.Signal) int {
	results := make(chan serviceResult, len(a.servers))
	var running sync.WaitGroup
	for _, server := range a.servers {
		running.Go(func() { results <- serviceResult{server: server, err: server.Start()} })
	}
	if len(a.servers) == 0 {
		return 0
	}
	var first serviceResult
	profile := prof.NewProfile()
	var terminating os.Signal
watch:
	for {
		select {
		case first = <-results:
			break watch
		case sig := <-signals:
			if sig == syscall.SIGTRAP {
				profile.StartOrStop()
				continue
			}
			terminating = sig
			for _, server := range a.servers {
				if receiver, ok := server.(interface{ SetShutdownSignal(os.Signal) }); ok {
					receiver.SetShutdownSignal(sig)
				}
			}
			break watch
		}
	}

	// Release every resource even if one closer fails.
	var closeErr error
	for _, closeFn := range a.closes {
		closeErr = errors.Join(closeErr, closeFn())
	}
	running.Wait()
	if first.err != nil {
		fmt.Fprintln(os.Stderr, first.err)
	}
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, closeErr)
	}
	if child, ok := first.server.(*UpstreamServer); ok {
		if code := child.ExitCode(); code != 0 {
			return code
		}
	}
	if first.err != nil || closeErr != nil {
		return 1
	}
	if terminating != nil {
		for _, server := range a.servers {
			if child, ok := server.(*UpstreamServer); ok {
				return child.ExitCode()
			}
		}
		if sig, ok := terminating.(syscall.Signal); ok {
			return 128 + int(sig)
		}
	}
	return 0
}
