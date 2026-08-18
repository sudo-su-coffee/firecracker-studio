//go:build windows

package api

import "runtime"

func readHostMetrics() hostMetrics {
	return hostMetrics{
		GPUMessage:       "Windows host telemetry is not available in this build",
		GPUAvailable:     false,
		CPUPercent:       0,
		MemoryTotalBytes: 0,
		MemoryUsedBytes:  0,
		MemoryPercent:    0,
		DiskTotalBytes:   0,
		DiskUsedBytes:    0,
		DiskPercent:      0,
		NetworkRxBytes:   0,
		NetworkTxBytes:   0,
	}
}

var _ = runtime.GOOS
