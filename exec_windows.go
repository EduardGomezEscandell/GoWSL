package wsl

import (
	"fmt"
	"syscall"
	"time"
)

func (c *Cmd) waitProcess() (uint32, error) {
	event, statusError := syscall.WaitForSingleObject(c.handle, syscall.INFINITE)
	if statusError != nil {
		return WindowsError, fmt.Errorf("failed syscall to WaitForSingleObject: %v", statusError)
	}
	if event != syscall.WAIT_OBJECT_0 {
		return WindowsError, fmt.Errorf("failed syscall to WaitForSingleObject, non-zero exit status %d", event)
	}

	// NOTE(brainman): It seems that sometimes process is not dead
	// when WaitForSingleObject returns. But we do not know any
	// other way to wait for it. Sleeping for a while seems to do
	// the trick sometimes.
	// See https://golang.org/issue/25965 for details.
	time.Sleep(5 * time.Millisecond)

	status, statusError := c.status()
	ok := statusError == nil && status == 0

	if err := syscall.CloseHandle(c.handle); !ok && err != nil {
		return WindowsError, err
	}
	return status, statusError
}

// status querries Windows for the process' status.
func (c *Cmd) status() (exit uint32, err error) {
	// Retrieving from cache in case the process has been closed
	if c.exitStatus != nil {
		return *c.exitStatus, nil
	}

	err = syscall.GetExitCodeProcess(c.handle, &exit)
	if err != nil {
		return WindowsError, fmt.Errorf("failed to retrieve exit status: %v", err)
	}
	return exit, nil
}

// kill gets the exit status before closing the process, without checking
// if it has finished or not.
func (c *Cmd) kill() error {
	status, err := c.status()
	c.exitStatus = nil
	if err == nil {
		c.exitStatus = &status
	}
	return syscall.TerminateProcess(c.handle, ActiveProcess)
}
