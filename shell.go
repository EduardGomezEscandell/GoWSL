package wsl

import (
	"errors"
	"fmt"
)

type shellOptions struct {
	command string
	useCWD  bool
}

// UseCWD is an optional parameter for (*Distro).Shell that makes it so the
// shell is started on the current working directory. Otherwise, it starts
// at the distro's $HOME.
func UseCWD() func(*shellOptions) {
	return func(o *shellOptions) {
		o.useCWD = true
	}
}

// WithCommand is an optional parameter for (*Distro).Shell that allows you
// to shell into WSL with the specified command. Particularly useful to choose
// what shell to use. Otherwise, it use's the distro's default shell.
func WithCommand(cmd string) func(*shellOptions) {
	return func(o *shellOptions) {
		o.command = cmd
	}
}

// Shell is a wrapper around Win32's WslLaunchInteractive, which starts a shell
// on WSL with the specified command. If no command is specified, an interactive
// session is started. This is a synchronous, blocking call.
//
// Can be used with optional helper parameters UseCWD and WithCommand.
func (d *Distro) Shell(opts ...func(*shellOptions)) (err error) {
	defer func() {
		if err == nil {
			return
		}
		if errors.Is(err, ExitError{}) {
			return
		}
		err = fmt.Errorf("error in Shell with distro %q: %v", d.Name, err)
	}()

	r, err := d.IsRegistered()
	if err != nil {
		return err
	}
	if !r {
		return errors.New("distro is not registered")
	}

	options := shellOptions{
		command: "",
		useCWD:  false,
	}
	for _, o := range opts {
		o(&options)
	}

	var exitCode uint32
	err = wslLaunchInteractive(d.Name, options.command, options.useCWD, &exitCode)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return &ExitError{Code: exitCode}
	}

	return nil
}
