//go:build linux

package api

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func readHostMetrics() hostMetrics {
	metrics := hostMetrics{GPUMessage: "No GPU telemetry provider detected"}
	if raw, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(raw))
		if len(parts) > 0 {
			if load, err := strconv.ParseFloat(parts[0], 64); err == nil {
				metrics.CPUPercent = minFloat(100, load/float64(runtime.NumCPU())*100)
			}
		}
	}
	if raw, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, available uint64
		for _, line := range strings.Split(string(raw), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			switch fields[0] {
			case "MemTotal:":
				total = value * 1024
			case "MemAvailable:":
				available = value * 1024
			}
		}
		metrics.MemoryTotalBytes, metrics.MemoryUsedBytes = total, total-available
		if total > 0 {
			metrics.MemoryPercent = float64(metrics.MemoryUsedBytes) / float64(total) * 100
		}
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		metrics.DiskTotalBytes = stat.Blocks * uint64(stat.Bsize)
		metrics.DiskUsedBytes = (stat.Blocks - stat.Bfree) * uint64(stat.Bsize)
		if metrics.DiskTotalBytes > 0 {
			metrics.DiskPercent = float64(metrics.DiskUsedBytes) / float64(metrics.DiskTotalBytes) * 100
		}
	}
	if raw, err := os.ReadFile("/proc/net/dev"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "lo" {
				continue
			}
			fields := strings.Fields(parts[1])
			if len(fields) >= 9 {
				rx, _ := strconv.ParseUint(fields[0], 10, 64)
				tx, _ := strconv.ParseUint(fields[8], 10, 64)
				metrics.NetworkRxBytes += rx
				metrics.NetworkTxBytes += tx
			}
		}
	}
	return metrics
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
