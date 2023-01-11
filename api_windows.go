package wsl

// This file contains windows-only API definitions and imports

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	// WSL api.
	wslAPIDll                          = syscall.NewLazyDLL("wslapi.dll")
	apiWslConfigureDistribution        = wslAPIDll.NewProc("WslConfigureDistribution")
	apiWslGetDistributionConfiguration = wslAPIDll.NewProc("WslGetDistributionConfiguration")
	apiWslLaunch                       = wslAPIDll.NewProc("WslLaunch")
	apiWslLaunchInteractive            = wslAPIDll.NewProc("WslLaunchInteractive")
	apiWslRegisterDistribution         = wslAPIDll.NewProc("WslRegisterDistribution")
	apiWslUnregisterDistribution       = wslAPIDll.NewProc("WslUnregisterDistribution")
)

const (
	lxssRegistry = registry.CURRENT_USER
	lxssPath     = `Software\Microsoft\Windows\CurrentVersion\Lxss\`
)

// Windows' typedefs.
type wBOOL = int     // Windows' BOOL
type wULONG = uint32 // Windows' ULONG
type char = byte     // Windows' CHAR (which is the same as C's char)
type handle = syscall.Handle

func coTaskMemFree(p unsafe.Pointer) {
	windows.CoTaskMemFree(p)
}

func wslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags wslFlags) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", distributionName)
	}

	r1, _, _ := apiWslConfigureDistribution.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(defaultUID),
		uintptr(wslDistributionFlags),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslConfigureDistribution")
	}

	return nil
}

func wslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *wslFlags,
	defaultEnvironmentVariables *map[string]string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", distributionName)
	}

	var (
		envVarsBegin **char
		envVarsLen   uint64 // size_t
	)

	r1, _, _ := apiWslGetDistributionConfiguration.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(distributionVersion)),
		uintptr(unsafe.Pointer(defaultUID)),
		uintptr(unsafe.Pointer(wslDistributionFlags)),
		uintptr(unsafe.Pointer(&envVarsBegin)),
		uintptr(unsafe.Pointer(&envVarsLen)),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslGetDistributionConfiguration")
	}

	*defaultEnvironmentVariables = processEnvVariables(envVarsBegin, envVarsLen)
	return nil
}

func wslIsDistributionRegistered(distributionName string) (bool, error) {
	distros, err := RegisteredDistros()
	if err != nil {
		return false, err
	}

	for _, dist := range distros {
		if dist.Name != distributionName {
			continue
		}
		return true, nil
	}
	return false, nil
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
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", distributionName)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", command)
	}

	var useCwd wBOOL
	if useCurrentWorkingDirectory {
		useCwd = 1
	}

	r1, _, _ := apiWslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		stdIn.Fd(),
		stdOut.Fd(),
		stdErr.Fd(),
		uintptr(unsafe.Pointer(process)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunch")
	}
	if *process == handle(0) {
		return fmt.Errorf("syscall to WslLaunch returned a null handle")
	}
	return nil
}

func wslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool, exitCode *uint32) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return fmt.Errorf("failed to convert command %q to UTF16", command)
	}

	var useCwd wBOOL
	if useCurrentWorkingDirectory {
		useCwd = 1
	}

	r1, _, _ := apiWslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(exitCode)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}

	if *exitCode == WindowsError {
		return fmt.Errorf("error on windows' side on WslLaunchInteractive")
	}

	return nil
}

func wslRegisterDistribution(distributionName string, tarGzFilename string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	tarGzFilenameUTF16, err := syscall.UTF16PtrFromString(tarGzFilename)
	if err != nil {
		return fmt.Errorf("failed to convert rootfs '%q' to UTF16", tarGzFilename)
	}

	r1, _, _ := apiWslRegisterDistribution.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(tarGzFilenameUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to wslRegisterDistribution")
	}

	return nil
}

func wslUnregisterDistribution(distributionName string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	r1, _, _ := apiWslUnregisterDistribution.Call(uintptr(unsafe.Pointer(distroUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}
	return nil
}

// processEnvVariables takes the (**char, length) obtained from Win32's API and returs a
// map[variableName]variableValue. It also deallocates each of the *char strings as well
// as the **char array.
func processEnvVariables(cStringArray **char, len uint64) map[string]string {
	stringPtrs := unsafe.Slice(cStringArray, len)

	env := make(chan struct {
		key   string
		value string
	})

	wg := sync.WaitGroup{}
	for _, cStr := range stringPtrs {
		cStr := cStr
		wg.Add(1)
		go func() {
			defer wg.Done()
			goStr := stringCtoGo(cStr, 32768)
			idx := strings.Index(goStr, "=")
			env <- struct {
				key   string
				value string
			}{
				key:   strings.Clone(goStr[:idx]),
				value: strings.Clone(goStr[idx+1:]),
			}
			coTaskMemFree(unsafe.Pointer(cStr))
		}()
	}

	// Cleanup
	go func() {
		wg.Wait()
		coTaskMemFree(unsafe.Pointer(cStringArray))
		close(env)
	}()

	// Collecting results
	m := map[string]string{}

	for kv := range env {
		m[kv.key] = kv.value
	}

	return m
}

// stringCtoGo converts a null-terminated *char into a string
// maxlen is the max distance that will searched. It is meant
// to prevent or mitigate buffer overflows.
func stringCtoGo(cString *char, maxlen uint64) (goString string) {
	size := strnlen(cString, maxlen)
	return string(unsafe.Slice(cString, size))
}

// strnlen finds the null terminator to determine *char length.
// The null terminator itself is not counted towards the length.
// maxlen is the max distance that will searched. It is meant to
// prevent or mitigate buffer overflows.
func strnlen(ptr *char, maxlen uint64) (length uint64) {
	length = 0
	for ; *ptr != 0 && length <= maxlen; ptr = charNext(ptr) {
		length++
	}
	return length
}

// charNext advances *char by one position.
func charNext(ptr *char) *char {
	return (*char)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + unsafe.Sizeof(char(0))))
}
