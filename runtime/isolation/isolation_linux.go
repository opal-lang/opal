//go:build linux

package isolation

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/builtwithtofu/sigil/core/decorator"
)

type LinuxNamespaceIsolator struct {
	namespaces []int
}

var _ decorator.IsolationContext = (*LinuxNamespaceIsolator)(nil)

func init() {
	registerIsolator(
		func() decorator.IsolationContext { return &LinuxNamespaceIsolator{} },
		linuxNamespacesSupported,
	)
}

func NewLinuxNamespaceIsolator() *LinuxNamespaceIsolator {
	return &LinuxNamespaceIsolator{}
}

func linuxNamespacesSupported() bool {
	if _, err := os.Stat("/proc/self/ns"); err != nil {
		return false
	}

	return true
}

func (i *LinuxNamespaceIsolator) Isolate(level decorator.IsolationLevel, config decorator.IsolationConfig) error {
	if level == decorator.IsolationLevelNone {
		return nil
	}

	if !IsSupported() {
		return errors.New("linux namespace isolation is not supported on this platform")
	}

	if err := i.DropPrivileges(); err != nil {
		return fmt.Errorf("drop privileges: %w", err)
	}

	if config.MemoryLock {
		if err := i.LockMemory(); err != nil {
			return fmt.Errorf("lock memory: %w", err)
		}
	}

	if level >= decorator.IsolationLevelStandard {
		if err := i.unshare(syscall.CLONE_NEWUSER | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS); err != nil {
			return fmt.Errorf("create user/pid/mount namespaces: %w", err)
		}

		switch config.NetworkPolicy {
		case decorator.NetworkPolicyDeny, decorator.NetworkPolicyLoopbackOnly:
			if err := i.DropNetwork(); err != nil {
				return fmt.Errorf("drop network: %w", err)
			}
		}

		switch config.FilesystemPolicy {
		case decorator.FilesystemPolicyReadOnly:
			if err := i.RestrictFilesystem([]string{"/"}, nil); err != nil {
				return fmt.Errorf("restrict filesystem read-only: %w", err)
			}
		case decorator.FilesystemPolicyEphemeral:
			if err := i.RestrictFilesystem(nil, []string{"/tmp"}); err != nil {
				return fmt.Errorf("restrict filesystem ephemeral: %w", err)
			}
		}
	}

	return nil
}

func (i *LinuxNamespaceIsolator) DropNetwork() error {
	if !IsSupported() {
		return errors.New("network namespace isolation is not supported on this platform")
	}

	if err := i.unshare(syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("create network namespace: %w", err)
	}

	return nil
}

func (i *LinuxNamespaceIsolator) RestrictFilesystem(readOnly, writable []string) error {
	if !IsSupported() {
		return errors.New("mount namespace isolation is not supported on this platform")
	}

	if err := i.unshare(syscall.CLONE_NEWNS); err != nil {
		return fmt.Errorf("create mount namespace: %w", err)
	}

	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("set mount propagation private: %w", err)
	}

	for _, path := range readOnly {
		if err := remount(path, syscall.MS_RDONLY); err != nil {
			return fmt.Errorf("remount %q read-only: %w", path, err)
		}
	}

	for _, path := range writable {
		if err := remount(path, 0); err != nil {
			return fmt.Errorf("remount %q writable: %w", path, err)
		}
	}

	return nil
}

func (i *LinuxNamespaceIsolator) DropPrivileges() error {
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

func (i *LinuxNamespaceIsolator) LockMemory() error {
	if err := syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
		return fmt.Errorf("mlockall: %w", err)
	}

	return nil
}

func (i *LinuxNamespaceIsolator) unshare(flags int) error {
	if err := syscall.Unshare(flags); err != nil {
		return err
	}

	i.namespaces = append(i.namespaces, flags)
	return nil
}

func remount(path string, extraFlags uintptr) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	if err := syscall.Mount(path, path, "", uintptr(syscall.MS_BIND|syscall.MS_REC), ""); err != nil {
		return err
	}

	return syscall.Mount(path, path, "", uintptr(syscall.MS_BIND|syscall.MS_REMOUNT)|extraFlags, "")
}
