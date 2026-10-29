//go:build windows

package engine

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type job struct {
	handle windows.Handle
}

func newJob() (*job, error) {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create job object: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(handle, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		windows.CloseHandle(handle)
		return nil, fmt.Errorf("configure job object: %w", err)
	}
	return &job{handle: handle}, nil
}

func (j *job) add(process *os.Process) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		return fmt.Errorf("open child process: %w", err)
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(j.handle, handle); err != nil {
		return fmt.Errorf("assign child to job: %w", err)
	}
	return nil
}

func (j *job) close() error {
	return windows.CloseHandle(j.handle)
}
