package statsutil

import (
	"github.com/docker/docker/api/types/container"
)

// CalculateCPUPercent calculates the CPU usage percentage from Docker stats.
func CalculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

// GetNetworkRx returns total received bytes across all network interfaces.
func GetNetworkRx(stats *container.StatsResponse) uint64 {
	var total uint64
	for _, v := range stats.Networks {
		total += v.RxBytes
	}
	return total
}

// GetNetworkTx returns total transmitted bytes across all network interfaces.
func GetNetworkTx(stats *container.StatsResponse) uint64 {
	var total uint64
	for _, v := range stats.Networks {
		total += v.TxBytes
	}
	return total
}

// GetBlockRead returns total bytes read from block devices.
func GetBlockRead(stats *container.StatsResponse) uint64 {
	var total uint64
	for _, bioEntry := range stats.BlkioStats.IoServiceBytesRecursive {
		if bioEntry.Op == "Read" || bioEntry.Op == "read" {
			total += bioEntry.Value
		}
	}
	return total
}

// GetBlockWrite returns total bytes written to block devices.
func GetBlockWrite(stats *container.StatsResponse) uint64 {
	var total uint64
	for _, bioEntry := range stats.BlkioStats.IoServiceBytesRecursive {
		if bioEntry.Op == "Write" || bioEntry.Op == "write" {
			total += bioEntry.Value
		}
	}
	return total
}
