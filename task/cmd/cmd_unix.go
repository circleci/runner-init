//go:build !windows

package cmd

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	"github.com/circleci/ex/o11y"
)

func forwardSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for sig := range ch {
			_ = cmd.Process.Signal(sig)
		}
	}()
}

func switchUser(ctx context.Context, cmd *exec.Cmd, username string) {
	usr, err := user.Lookup(username)

	if err == nil {
		cmd.Env = append(cmd.Env, "HOME="+usr.HomeDir)

		uid, _ := strconv.Atoi(usr.Uid)
		gid, _ := strconv.Atoi(usr.Gid)
		//nolint:gosec // G115: we only support POSIX right now, so this won't overflow
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	} else {
		o11y.LogError(ctx, "failed to lookup user", err, o11y.Field("username", username))
	}
}

func additionalSetup(_ context.Context, cmd *exec.Cmd) {
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	cmd.Cancel = func() error {
		// Kill the child process group
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
