package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"unsafe"

	"github.com/circleci/ex/o11y"
	"golang.org/x/sys/windows"
)

func forwardSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	go func() {
		defer signal.Stop(ch)
		for range ch {
			_ = cmd.Process.Kill()
		}
	}()
}

func switchUser(ctx context.Context, _ *exec.Cmd, user string) {
	o11y.Log(ctx, "switching users is unsupported on windows", o11y.Field("username", user))
}

func additionalSetup(_ context.Context, cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags = windows.CREATE_NEW_PROCESS_GROUP
}

func (c *Command) start() error {
	g, err := newProcessExitGroup()
	if err != nil {
		return fmt.Errorf("failed to create new process group: %w", err)
	}

	c.cmd.Cancel = func() error {
		return g.Dispose()
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	if err := g.AddProcess(c.cmd.Process); err != nil {
		return fmt.Errorf("failed to add process to group: %w", err)
	}

	return nil
}

type processExitGroup windows.Handle

func newProcessExitGroup() (processExitGroup, error) {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {
		return 0, err
	}

	return processExitGroup(handle), nil
}

func (g processExitGroup) Dispose() error {
	return windows.CloseHandle(windows.Handle(g))
}

func (g processExitGroup) AddProcess(p *os.Process) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return windows.AssignProcessToJobObject(windows.Handle(g), handle)
}
