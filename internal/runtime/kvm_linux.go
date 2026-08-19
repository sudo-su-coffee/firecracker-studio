//go:build linux

package runtime

import (
	"fmt"
	"os"
	"syscall"
)

const kvmGetAPIVersion = 0xAE00

func kvmStatus() string {
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return "permission denied"
	}
	defer file.Close()
	version, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), kvmGetAPIVersion, 0)
	if errno != 0 {
		return fmt.Sprintf("unusable (%v)", errno)
	}
	if int64(version) <= 0 {
		return "unusable (invalid API version)"
	}
	return fmt.Sprintf("ready (api %d)", version)
}
