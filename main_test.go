package wsl_test

// This file conatains testing functionality

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

const (
	nameSuffix  string = "wsltesting"
	emptyRootFs string = `C:\Users\edu19\Work\images\empty.tar.gz` // Empty non-functional image. It registers instantly.
	jammyRootFs string = `C:\Users\edu19\Work\images\jammy.tar.gz` // Fully functional rootfs
)

type Tester struct {
	*testing.T
	distros []wsl.Distro
	tmpdirs []string
}

func TestMain(m *testing.M) {

	fullCleanup()
	exitVal := m.Run()
	fullCleanup()

	os.Exit(exitVal)
}

func fullCleanup() {
	wsl.Shutdown()
	// Cleanup without nagging
	if cachedDistro != nil {
		cleanUpWslInstancess([]wsl.Distro{*cachedDistro})
	}
	// Cleanup with nagging
	cleanUpTestWslInstances()
}

// NewTester extends Tester with some WSL-specific functionality and cleanup
func NewTester(tst *testing.T) *Tester {
	t := Tester{T: tst}
	t.Cleanup(func() {
		t.cleanUpWslInstances()
		t.cleanUpTempDirectories()
	})
	return &t
}

// NewWslDistro creates a new distro with a mangled name and adds it to list of distros to remove.
// Note that the distro is not registered.
func (t *Tester) NewWslDistro(name string) wsl.Distro {
	d := wsl.Distro{Name: t.mangleName(name)}
	t.distros = append(t.distros, d)
	return d
}

var cachedDistro *wsl.Distro = nil

// CachedDistro provides a distro for non-destructive and generally immutable commands
// without having to create and destroy a new distro for it.
func (t *Tester) CachedDistro() wsl.Distro {
	if cachedDistro == nil {
		cachedDistro = &wsl.Distro{Name: fmt.Sprintf("reusableDistro_TestMain_%s", nameSuffix)}
		err := cachedDistro.Register(jammyRootFs)
		require.NoError(t, err)
	}
	return *cachedDistro
}

// NewTestDir creates a unique directory and adds it to the list of dirs to remove
func (t *Tester) NewTestDir(prefix string) (string, error) {
	clean_prefix := strings.Replace(t.Name()+prefix, "/", "_", -1)
	tmpdir, err := ioutil.TempDir(os.TempDir(), clean_prefix)
	if err != nil {
		return "", err
	}

	t.tmpdirs = append(t.tmpdirs, tmpdir)
	return tmpdir, nil
}

func (t *Tester) cleanUpWslInstances() {
	cleanUpWslInstancess(t.distros)
}

func (t *Tester) cleanUpTempDirectories() {
	for _, dir := range t.tmpdirs {
		dir := dir
		err := os.RemoveAll(dir)
		if err != nil {
			t.Logf("Failed to remove temp directory %s: %v\n", dir, err)
		}
	}
}

// cleanUpTestWslInstances finds all distros with a mangled name and unregisters them
func cleanUpTestWslInstances() {
	testInstances, err := RegisteredTestWslInstances()
	if err != nil {
		return
	}
	if len(testInstances) != 0 {
		fmt.Printf("The following WSL distros were not properly cleaned up: %v\n", testInstances)
	}
	cleanUpWslInstancess(testInstances)
}

func cleanUpWslInstancess(distros []wsl.Distro) {
	for _, i := range distros {

		if r, err := i.IsRegistered(); err == nil && !r {
			return
		}
		err := i.Unregister()
		if err != nil {
			name, test := unmangleName(i.Name)
			fmt.Printf("Failed to clean up test WSL distro (name=%s, test=%s)\n", name, test)
		}

	}
}

// RegisteredTestWslInstances finds all distros with a mangled name
func RegisteredTestWslInstances() ([]wsl.Distro, error) {
	distros := []wsl.Distro{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasSuffix(line, nameSuffix) {
			continue
		}
		distros = append(distros, wsl.Distro{Name: line})
	}

	return distros, nil
}

// mangleName avoids name collisions with existing distros by adding a suffix
func (t Tester) mangleName(name string) string {
	return fmt.Sprintf("%s_%s_%s", name, strings.ReplaceAll(t.Name(), "/", "--"), nameSuffix)
}

// unmangleName retrieves encoded info from a mangled DistroName
func unmangleName(mangledName string) (name string, test string) {
	words := strings.Split(mangledName, "_")
	l := len(words)
	name = strings.Join(words[:l-2], "_")
	test = words[l-2]
	return name, test
}

// registerFromPowershell registers a WSL distro bypassing the wsl.module, for better test segmentation
func (t *Tester) RegisterFromPowershell(i wsl.Distro, image string) {
	tmpdir, err := t.NewTestDir(i.Name)
	require.NoError(t, err)

	cmdString := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", i.Name, tmpdir, jammyRootFs)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // WSL sometimes gets stuck installing
	defer cancel()

	output, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmdString).CombinedOutput()
	require.NoError(t, err, string(output))
}
