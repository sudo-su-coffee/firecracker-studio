//go:build !linux

package runtime

func kvmStatus() string { return "unsupported on this host" }
