package cmd

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
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
	cmd.SysProcAttr.CreationFlags = windows.CREATE_NEW_PROCESS_GROUP
}

func (c *Command) start() error {
	g, err := newProcessExitGroup()
	if err != nil {
		return err
	}

	c.cmd.Cancel = func() error {
		return g.Dispose()
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	return g.AddProcess(c.cmd.Process)
}

// Process struct matches os.Process layout to extract handle via unsafe pointer
// https://stackoverflow.com/a/56739249
type process struct {
	Pid    int
	_      uint8
	_      atomic.Uint64
	_      sync.RWMutex
	Handle uintptr
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
	return windows.AssignProcessToJobObject(
		windows.Handle(g),
		windows.Handle((*process)(unsafe.Pointer(p)).Handle))
}
