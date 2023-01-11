package wsl

import (
	"errors"
	"os"
)

type handle = uintptr

func wslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags wslFlags) error {
	return errors.New("not implemented")
}

func wslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *wslFlags,
	defaultEnvironmentVariables *map[string]string) error {
	return errors.New("not implemented")
}

func wslIsDistributionRegistered(distributionName string) (bool, error) {
	return false, errors.New("not implemented")
}

func wslLaunch(
	distributionName string,
	command string,
	useCurrentWorkingDirectory bool,
	stdIn *os.File,
	stdOut *os.File,
	stdErr *os.File,
	process *handle,
) error {
	return errors.New("not implemented")
}

func wslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool, exitCode *uint32) error {
	return errors.New("not implemented")
}

func wslRegisterDistribution(distributionName string, tarGzFilename string) error {
	return errors.New("not implemented")
}

func wslUnregisterDistribution(distributionName string) error {
	return errors.New("not implemented")
}
