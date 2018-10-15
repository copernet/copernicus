// +build !windows,!plan9

package limits

import (
	"syscall"
	"testing"
)

func TestSetLimits(t *testing.T) {
	if err := SetLimits(); err == nil {
		var fileno syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &fileno)
		if err != nil {
			t.Fatalf("Getrlimit failed :%v\n", err)
		}
		if fileno.Cur < fileLimitMin {
			t.Fatalf("current limit should be at least %d\n", fileLimitMin)
		}
	}
}
