package wsl

import (
	"errors"
)

// kill gets the exit status before closing the process, without checking
// if it has finished or not.
func (c *Cmd) kill() error {
	return errors.New("not implemented")
}

func (c *Cmd) waitProcess() (uint32, error) {
	return 0, errors.New("not implemented")
}
