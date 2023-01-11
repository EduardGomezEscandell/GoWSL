package wsl

// This file contains utilities to create, destroy, stop WSL distros,
// as well as utilities to query this status.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Register is a wrapper around Win32's WslRegisterDistribution.
// It creates a new distro with a copy of the given tarball as
// its filesystem.
func (d *Distro) Register(rootFsPath string) (e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("error registering %q: %v", d.Name, e)
		}
	}()

	rootFsPath, err := fixPath(rootFsPath)
	if err != nil {
		return err
	}

	r, err := d.IsRegistered()
	if err != nil {
		return errors.New("failed to detect if it is already installed")
	}
	if r {
		return errors.New("already registered")
	}

	return wslRegisterDistribution(d.Name, rootFsPath)
}

// RegisteredDistros returns a slice of the registered distros.
func RegisteredDistros() ([]Distro, error) {
	return registeredDistros()
}

// IsRegistered returns a boolean indicating whether a distro is registered or not.
func (d Distro) IsRegistered() (registered bool, e error) {
	b, err := wslIsDistributionRegistered(d.Name)
	if err != nil {
		return b, fmt.Errorf("failed to detect if %q is registered: %v", d.Name, err)
	}
	return b, nil
}

// Unregister is a wrapper around Win32's WslUnregisterDistribution.
// It irreparably destroys a distro and its filesystem.
func (d *Distro) Unregister() (e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("failed to unregister %q: %v", d.Name, e)
		}
	}()

	r, err := d.IsRegistered()
	if err != nil {
		return err
	}
	if !r {
		return errors.New("not registered")
	}

	return wslUnregisterDistribution(d.Name)
}

// fixPath deals with the fact that WslRegisterDistribuion is
// a bit picky with the path format.
func fixPath(relative string) (string, error) {
	abs, err := filepath.Abs(filepath.FromSlash(relative))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file %q does not exist", abs)
	}
	return abs, nil
}
