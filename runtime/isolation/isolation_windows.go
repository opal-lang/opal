//go:build windows

package isolation

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/opal-lang/opal/core/decorator"
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x00002000
)

type windowsJobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type windowsJobObjectExtendedLimitInformation struct {
	BasicLimitInformation windowsJobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type WindowsJobObjectIsolator struct {
	job syscall.Handle
}

var _ decorator.IsolationContext = (*WindowsJobObjectIsolator)(nil)

func init() {
	registerIsolator(
		func() decorator.IsolationContext { return &WindowsJobObjectIsolator{} },
		func() bool { return true },
	)
}

func (i *WindowsJobObjectIsolator) Isolate(level decorator.IsolationLevel, config decorator.IsolationConfig) error {
	if level == decorator.IsolationLevelNone {
		return nil
	}

	if err := i.attachToCurrentProcessJob(); err != nil {
		return fmt.Errorf("attach process to job object: %w", err)
	}

	if config.MemoryLock {
		_ = i.LockMemory()
	}

	return nil
}

func (i *WindowsJobObjectIsolator) DropNetwork() error {
	return nil
}

func (i *WindowsJobObjectIsolator) RestrictFilesystem(readOnly, writable []string) error {
	return nil
}

func (i *WindowsJobObjectIsolator) DropPrivileges() error {
	return nil
}

func (i *WindowsJobObjectIsolator) LockMemory() error {
	return nil
}

func (i *WindowsJobObjectIsolator) attachToCurrentProcessJob() error {
	if i.job != 0 {
		return nil
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createJobObject := kernel32.NewProc("CreateJobObjectW")
	setInformationJobObject := kernel32.NewProc("SetInformationJobObject")
	assignProcessToJobObject := kernel32.NewProc("AssignProcessToJobObject")
	getCurrentProcess := kernel32.NewProc("GetCurrentProcess")

	r1, _, createErr := createJobObject.Call(0, 0)
	if r1 == 0 {
		return os.NewSyscallError("CreateJobObjectW", createErr)
	}

	job := syscall.Handle(r1)
	limitInfo := windowsJobObjectExtendedLimitInformation{}
	limitInfo.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose

	r1, _, setErr := setInformationJobObject.Call(
		uintptr(job),
		jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limitInfo)),
		unsafe.Sizeof(limitInfo),
	)
	if r1 == 0 {
		_ = syscall.CloseHandle(job)
		return os.NewSyscallError("SetInformationJobObject", setErr)
	}

	r1, _, _ = getCurrentProcess.Call()
	currentProcess := syscall.Handle(r1)

	r1, _, assignErr := assignProcessToJobObject.Call(uintptr(job), uintptr(currentProcess))
	if r1 == 0 {
		_ = syscall.CloseHandle(job)
		return os.NewSyscallError("AssignProcessToJobObject", assignErr)
	}

	i.job = job
	return nil
}
