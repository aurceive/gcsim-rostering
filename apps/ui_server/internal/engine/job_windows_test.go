//go:build windows

package engine

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestJobParentHelper(t *testing.T) {
	if os.Getenv("UISERVER_JOB_PARENT") != "1" {
		return
	}
	j, err := newJob()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestJobChildHelper$")
	child.Env = append(os.Environ(), "UISERVER_JOB_CHILD=1")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	if err := j.add(child.Process); err != nil {
		_ = child.Process.Kill()
		t.Fatal(err)
	}
	fmt.Println(child.Process.Pid)
	for {
		time.Sleep(time.Second)
	}
}

func TestJobChildHelper(t *testing.T) {
	if os.Getenv("UISERVER_JOB_CHILD") != "1" {
		return
	}
	for {
		time.Sleep(time.Second)
	}
}

func TestJobKillsChildOnParentExit(t *testing.T) {
	parent := exec.Command(os.Args[0], "-test.run=^TestJobParentHelper$")
	parent.Env = append(os.Environ(), "UISERVER_JOB_PARENT=1")
	parent.Stderr = os.Stderr
	stdout, err := parent.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = parent.Process.Kill()
		_ = parent.Wait()
	}()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read child PID: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	defer windows.TerminateProcess(handle, 1)
	if err := parent.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	result, err := windows.WaitForSingleObject(handle, 5000)
	if err != nil || result != windows.WAIT_OBJECT_0 {
		t.Fatalf("child did not exit after parent termination: result=%v err=%v", result, err)
	}
}
