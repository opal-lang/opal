//go:build darwin

package isolation

import (
	"fmt"
	"syscall"

	"github.com/opal-lang/opal/core/decorator"
)

type DarwinSandboxIsolator struct{}

var _ decorator.IsolationContext = (*DarwinSandboxIsolator)(nil)

func init() {
	registerIsolator(
		func() decorator.IsolationContext { return &DarwinSandboxIsolator{} },
		func() bool { return true },
	)
}

func (i *DarwinSandboxIsolator) Isolate(level decorator.IsolationLevel, config decorator.IsolationConfig) error {
	if level == decorator.IsolationLevelNone {
		return nil
	}

	if err := i.DropPrivileges(); err != nil {
		return fmt.Errorf("drop privileges: %w", err)
	}

	if config.MemoryLock {
		_ = i.LockMemory()
	}

	return nil
}

func (i *DarwinSandboxIsolator) DropNetwork() error {
	return nil
}

func (i *DarwinSandboxIsolator) RestrictFilesystem(readOnly, writable []string) error {
	return nil
}

func (i *DarwinSandboxIsolator) DropPrivileges() error {
	groups, err := syscall.Getgroups()
	if err != nil {
		return fmt.Errorf("read supplementary groups: %w", err)
	}

	if len(groups) == 0 {
		return nil
	}

	if err := syscall.Setgroups([]int{}); err != nil {
		return fmt.Errorf("drop supplementary groups: %w", err)
	}

	return nil
}

func (i *DarwinSandboxIsolator) LockMemory() error {
	if err := syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
		return fmt.Errorf("mlockall: %w", err)
	}

	return nil
}
