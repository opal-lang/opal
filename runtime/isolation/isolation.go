package isolation

import (
	"errors"

	"github.com/opal-lang/opal/core/decorator"
)

type IsolationContext = decorator.IsolationContext

var (
	defaultIsolatorFactory = func() decorator.IsolationContext {
		return &unsupportedIsolator{reason: "isolation is not supported on this platform"}
	}
	isSupported = func() bool {
		return false
	}
)

func registerIsolator(factory func() decorator.IsolationContext, supported func() bool) {
	if factory != nil {
		defaultIsolatorFactory = factory
	}
	if supported != nil {
		isSupported = supported
	}
}

func NewIsolator() decorator.IsolationContext {
	return defaultIsolatorFactory()
}

func IsSupported() bool {
	return isSupported()
}

type unsupportedIsolator struct {
	reason string
}

var _ decorator.IsolationContext = (*unsupportedIsolator)(nil)

func (i *unsupportedIsolator) Isolate(level decorator.IsolationLevel, config decorator.IsolationConfig) error {
	return errors.New(i.reason)
}

func (i *unsupportedIsolator) DropNetwork() error {
	return errors.New(i.reason)
}

func (i *unsupportedIsolator) RestrictFilesystem(readOnly, writable []string) error {
	return errors.New(i.reason)
}

func (i *unsupportedIsolator) DropPrivileges() error {
	return errors.New(i.reason)
}

func (i *unsupportedIsolator) LockMemory() error {
	return errors.New(i.reason)
}
