//go:build !windows

package engine

import "os"

type job struct{}

func newJob() (*job, error)          { return &job{}, nil }
func (j *job) add(*os.Process) error { return nil }
func (j *job) close() error          { return nil }
