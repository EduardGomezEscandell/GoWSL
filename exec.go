package wsl

// This file contains utilities to launch commands into WSL instances.

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Windows' constants
const (
	WindowsError  ExitCode = 4294967295 // Underflowed -1
	ActiveProcess ExitCode = 259
)

// Cmd is a wrapper around the Windows process spawned by WslLaunch
type Cmd struct {
	// Public parameters
	Stdout syscall.Handle
	Stdin  syscall.Handle
	Stderr syscall.Handle
	UseCWD bool

	// Immutable parameters
	instance *Distro
	command  string

	// Book-keeping
	handle syscall.Handle
}

type ExitError struct {
	Code ExitCode
}

func (m *ExitError) Error() string {
	return fmt.Sprintf("exit error: %d", m.Code)
}

// Command returns the Cmd struct to execute the named program with
// the given arguments in the same string.
//
// It sets only the command and stdin/stdout/stderr in the returned structure.
func (i *Distro) Command(command string) Cmd {
	return Cmd{
		Stdin:    syscall.Stdin,
		Stdout:   syscall.Stdout,
		Stderr:   syscall.Stderr,
		UseCWD:   false,
		instance: i,
		handle:   0,
		command:  command,
	}
}

// Start starts the specified WslProcess but does not wait for it to complete.
//
// The Wait method will return the exit code and release associated resources
// once the command exits.
func (p *Cmd) Start() error {
	instanceUTF16, err := syscall.UTF16PtrFromString(p.instance.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.instance)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(p.command)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.command)
	}

	var useCwd wBOOL = 0
	if p.UseCWD {
		useCwd = 1
	}

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(p.Stdin),
		uintptr(p.Stdout),
		uintptr(p.Stderr),
		uintptr(unsafe.Pointer(&p.handle)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunch")
	}
	return nil
}

// Wait blocks execution until the process finishes and returns the process exit status.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p Cmd) Wait() error {
	defer p.Close()
	r1, error := syscall.WaitForSingleObject(p.handle, syscall.INFINITE)
	if r1 != 0 {
		return fmt.Errorf("failed syscall to WaitForSingleObject: %v", error)
	}

	return p.queryStatus()
}

// Run starts the specified WslProcess and waits for it to complete.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p *Cmd) Run() error {
	if err := p.Start(); err != nil {
		return err
	}
	return p.Wait()
}

// Close closes a WslProcess. If it was still running, it is terminated,
// although its Linux counterpart may not.
func (p *Cmd) Close() error {
	defer func() {
		p.handle = 0
	}()
	return syscall.CloseHandle(p.handle)
}

// queryStatus querries Windows for the process' status.
func (p *Cmd) queryStatus() error {
	exit := ExitCode(0)
	err := syscall.GetExitCodeProcess(p.handle, &exit)
	if err != nil {
		return err
	}
	if exit != 0 {
		return &ExitError{Code: exit}
	}
	return nil
}