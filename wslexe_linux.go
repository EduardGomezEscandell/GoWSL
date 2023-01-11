package wsl

import (
	"errors"
)

func shutdown() error {
	return errors.New("not implemented")
}

func terminate(distroName string) error {
	return errors.New("not implemented")
}

func registeredDistros() (distros []Distro, err error) {
	return nil, errors.New("not implemented")
}
