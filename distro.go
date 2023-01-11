// Package wsl wraps around the wslApi.dll (and sometimes wsl.exe) for
// safe and idiomatic use within Go projects.
//
// This package is not thread safe.
package wsl

// This file contains utilities to interact with a Distro and its configuration

import (
	"fmt"
	"sort"
)

// Distro is an abstraction around a WSL distro.
type Distro struct {
	Name string
}

// Terminate powers off the distro.
// Equivalent to:
//
//	wsl --terminate <distro>
func (d Distro) Terminate() error {
	return terminate(d.Name)
}

// Shutdown powers off all of WSL, including all other distros.
// Equivalent to:
//
//	wsl --shutdown
func Shutdown() error {
	return shutdown()
}

// SetAsDefault sets a particular distribution as the default one.
// Equivalent to:
//   wsl --set-default <distro>
func (d Distro) SetAsDefault() error {
	return setAsDefault(d.Name)
}

// DefaultDistro gets the current default distribution.
func DefaultDistro() (Distro, error) {
	n, e := defaultDistro()
	return Distro{Name: n}, e
}

// Windows' WSL_DISTRIBUTION_FLAGS
// https://learn.microsoft.com/en-us/windows/win32/api/wslapi/ne-wslapi-wsl_distribution_flags
type wslFlags int

// Allowing underscores in names to keep it as close to Windows as possible.
const (
	flag_NONE                  wslFlags = 0x0 //nolint: revive
	flag_ENABLE_INTEROP        wslFlags = 0x1 //nolint: revive
	flag_APPEND_NT_PATH        wslFlags = 0x2 //nolint: revive
	flag_ENABLE_DRIVE_MOUNTING wslFlags = 0x4 //nolint: revive

	// Per the conversation at https://github.com/microsoft/WSL-DistroLauncher/issues/96
	// the information about version 1 or 2 is on the 4th bit of the flags, which is
	// currently referenced neither by the API nor the documentation.
	flag_undocumented_WSL_VERSION wslFlags = 0x8 //nolint: revive
)

// Configuration is the configuration of the distro.
type Configuration struct {
	Version                     uint8             // Type of filesystem used (lxfs vs. wslfs, relevant only to WSL1)
	DefaultUID                  uint32            // User ID of default user
	InteropEnabled              bool              // Whether interop with windows is enabled
	PathAppended                bool              // Whether Windows paths are appended
	DriveMountingEnabled        bool              // Whether drive mounting is enabled
	undocumentedWSLVersion      uint8             // Undocumented variable. WSL1 vs. WSL2.
	DefaultEnvironmentVariables map[string]string // Environment variables passed to the distro by default
}

// DefaultUID sets the user to the one specified.
func (d *Distro) DefaultUID(uid uint32) error {
	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.DefaultUID = uid
	return d.configure(conf)
}

// InteropEnabled sets the ENABLE_INTEROP flag to the provided value.
// Enabling allows you to launch Windows executables from WSL.
func (d *Distro) InteropEnabled(value bool) error {
	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.InteropEnabled = value
	return d.configure(conf)
}

// PathAppended sets the APPEND_NT_PATH flag to the provided value.
// Enabling it allows WSL to append /mnt/c/... (or wherever your mount
// point is) in front of Windows executables.
func (d *Distro) PathAppended(value bool) error {
	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.PathAppended = value
	return d.configure(conf)
}

// DriveMountingEnabled sets the ENABLE_DRIVE_MOUNTING flag to the provided value.
// Enabling it mounts the windows filesystem into WSL's.
func (d *Distro) DriveMountingEnabled(value bool) error {
	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.DriveMountingEnabled = value
	return d.configure(conf)
}

// GetConfiguration is a wrapper around Win32's WslGetDistributionConfiguration.
// It returns a configuration object with information about the distro.
func (d Distro) GetConfiguration() (c Configuration, e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("error in GetConfiguration: %v", e)
		}
	}()
	var conf Configuration
	var flags wslFlags

	err := wslGetDistributionConfiguration(
		d.Name,
		&conf.Version,
		&conf.DefaultUID,
		&flags,
		&conf.DefaultEnvironmentVariables,
	)

	if err != nil {
		return conf, err
	}

	conf.unpackFlags(flags)
	return conf, nil
}

// String deserializes a distro and its configuration as a yaml string.
func (d Distro) String() string {
	c, err := d.GetConfiguration()
	if err != nil {
		return fmt.Sprintf(`distro: %s
configuration: %v
`, d.Name, err)
	}

	// Get sorted list of environment variables
	envKeys := []string{}
	for k := range c.DefaultEnvironmentVariables {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	fmtEnvs := "\n"
	for _, k := range envKeys {
		fmtEnvs = fmt.Sprintf("%s    - %s: %s\n", fmtEnvs, k, c.DefaultEnvironmentVariables[k])
	}

	// Generate the string
	return fmt.Sprintf(`distro: %s
configuration:
  - Version: %d
  - DefaultUID: %d
  - InteropEnabled: %t
  - PathAppended: %t
  - DriveMountingEnabled: %t
  - undocumentedWSLVersion: %d
  - DefaultEnvironmentVariables:%s`, d.Name, c.Version, c.DefaultUID, c.InteropEnabled, c.PathAppended,
		c.DriveMountingEnabled, c.undocumentedWSLVersion, fmtEnvs)
}

// configure is a wrapper around Win32's WslConfigureDistribution.
// Note that only the following config is mutable:
//   - DefaultUID
//   - InteropEnabled
//   - PathAppended
//   - DriveMountingEnabled
func (d *Distro) configure(config Configuration) error {
	flags, err := config.packFlags()
	if err != nil {
		return err
	}

	return wslConfigureDistribution(d.Name, config.DefaultUID, flags)
}

// unpackFlags examines a winWslFlags object and stores its findings in the Configuration.
func (conf *Configuration) unpackFlags(flags wslFlags) {
	conf.InteropEnabled = false
	if flags&flag_ENABLE_INTEROP != 0 {
		conf.InteropEnabled = true
	}

	conf.PathAppended = false
	if flags&flag_APPEND_NT_PATH != 0 {
		conf.PathAppended = true
	}

	conf.DriveMountingEnabled = false
	if flags&flag_ENABLE_DRIVE_MOUNTING != 0 {
		conf.DriveMountingEnabled = true
	}

	conf.undocumentedWSLVersion = 1
	if flags&flag_undocumented_WSL_VERSION != 0 {
		conf.undocumentedWSLVersion = 2
	}
}

// packFlags generates a winWslFlags object from the Configuration.
func (conf Configuration) packFlags() (wslFlags, error) {
	flags := flag_NONE

	if conf.InteropEnabled {
		flags = flags | flag_ENABLE_INTEROP
	}

	if conf.PathAppended {
		flags = flags | flag_APPEND_NT_PATH
	}

	if conf.DriveMountingEnabled {
		flags = flags | flag_ENABLE_DRIVE_MOUNTING
	}

	switch conf.undocumentedWSLVersion {
	case 1:
	case 2:
		flags = flags | flag_undocumented_WSL_VERSION
	default:
		return flags, fmt.Errorf("unknown WSL version %d", conf.undocumentedWSLVersion)
	}

	return flags, nil
}
