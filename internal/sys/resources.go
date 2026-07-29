/*
This package provides an interface to collect information about the system in
which the OMR is running. It passively checks certain statistics and caches
them in memory and on disk to provide stable access, tracks certain configured
constants (i.e., user- and runtime-determined resource limits), and exposes
logging functions that can be used globally.
*/
package sys

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
	"ubco-team15/omr/internal/pdf"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

//
// In this file we implement functionality to periodically check the system's
// resource utilization (CPU, memory), cache it, and retrieve it.
//

const (
	PollPeriod  = time.Second // Poll CPU/memory/disk stats every 1 second
	HistorySize = 1 << 12     // Hold up to this many recorded stats.
)

var (
	cpuHistory  *RingBuffer[CpuInfo]
	memHistory  *RingBuffer[MemInfo]
	diskHistory *RingBuffer[DiskInfo]

	// The maximum amount of memory used by the OMR since startup.
	lifetimeMaxMem uint64
)

func init() {

	cpuHistory = NewRingBuffer[CpuInfo](HistorySize)
	memHistory = NewRingBuffer[MemInfo](HistorySize)
	diskHistory = NewRingBuffer[DiskInfo](HistorySize)

	cpu, err := currentCpuInfo()
	if err != nil {
		panic("error getting CPU info: " + err.Error())
	}

	mem, err := currentMemInfo()
	if err != nil {
		panic("error getting memory info: " + err.Error())
	}

	disk, err := currentDiskInfo()
	if err != nil {
		panic("error getting disk info: " + err.Error())
	}

	cpuHistory.Add(cpu)
	memHistory.Add(mem)
	diskHistory.Add(disk)

	lifetimeMaxMem = mem.InUseOmr

	go func() {
		for range time.Tick(PollPeriod) {

			//
			// We ignore errors below, under the assumption that if there are
			// errors, they would've occurred on the initialization. But it's
			// also not a big deal if these values are zero since they don't
			// dictate any behavior.
			//

			cpu, _ := currentCpuInfo()
			cpuHistory.Add(cpu)

			mem, _ := currentMemInfo()
			memHistory.Add(mem)

			disk, _ := currentDiskInfo()
			diskHistory.Add(disk)

			lifetimeMaxMem = max(lifetimeMaxMem, mem.InUseOmr)

		}
	}()
}

var startupTime = time.Now()

// Returns the duration that the application has been running.
func Uptime() time.Duration {
	return time.Since(startupTime)
}

// Returns true if the application is running inside of a Docker container.
func Docker() bool {
	_, err := os.Stat("/docker.env")
	return err != nil
}

// Returns the peak memory usage for the OMR since startup.
func PeakMem() uint64 {
	return lifetimeMaxMem
}

//
// CPU
//

// Gets up to the n most recent recorded CPU stats. CPU stats are collected at
// a rate determined by [PollPeriod]. The maximum number of stats that can be
// retrieved is determined by [HistorySize].
func CpuHistory(n int) []CpuInfo {
	return cpuHistory.Get(n)
}

type CpuInfo struct {
	Description    string  `json:"description"`
	OverallPercent float64 `json:"overallPercent"`
	FrequencyMhz   float64 `json:"mhz"`
	Threads        []struct {
		Percent float64 `json:"percent"`
	} `json:"threads"`
}

func currentCpuInfo() (CpuInfo, error) {
	newInfo := CpuInfo{}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return CpuInfo{}, err
	}

	if len(cpuInfo) > 0 {
		newInfo.Description = cpuInfo[0].ModelName
		newInfo.FrequencyMhz = cpuInfo[0].Mhz
	}

	overall, err := cpu.Percent(0, false)
	if err != nil {
		return CpuInfo{}, err
	}

	if len(overall) > 0 {
		newInfo.OverallPercent = overall[0]
	}

	corePercents, err := cpu.Percent(0, true)
	if err != nil {
		return CpuInfo{}, err
	}

	newInfo.Threads = make([]struct {
		Percent float64 `json:"percent"`
	}, len(corePercents))

	for i, percent := range corePercents {
		newInfo.Threads[i].Percent = percent
	}

	return newInfo, nil
}

//
// Memory
//

// Gets up to the n most recent recorded memory usage stats. Memory stats are
// collected at a rate determined by [PollPeriod]. The maximum number of stats
// that can be retrieved is determined by [HistorySize].
func MemHistory(n int) []MemInfo {
	return memHistory.Get(n)
}

type MemInfo struct {
	InUseOmr   uint64 `json:"inUseOmr"`
	InUseOther uint64 `json:"inUseOther"`
	Free       uint64 `json:"free"`
}

func currentMemInfo() (MemInfo, error) {
	newInfo := MemInfo{}

	vmem, err := mem.VirtualMemory()
	if err != nil {
		return MemInfo{}, nil
	}

	newInfo.Free = vmem.Available

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return MemInfo{}, nil
	}

	rss, err := proc.MemoryInfo()
	if err != nil {
		return MemInfo{}, nil
	}

	newInfo.InUseOmr = rss.RSS
	newInfo.InUseOther = vmem.Used - rss.RSS

	return newInfo, nil
}

//
// Disk
//

// Gets up to the n most recent recorded disk usage stats. Disk stats are
// collected at a rate determined by [PollPeriod]. The maximum number of
// stats that can be retrieved is determined by [HistorySize].
func DiskHistory(n int) []DiskInfo {
	return diskHistory.Get(n)
}

type DiskInfo struct {
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
	Used  uint64 `json:"used"`
}

func currentDiskInfo() (DiskInfo, error) {
	d, err := disk.Usage("/")
	if err != nil {
		return DiskInfo{}, err
	}

	return DiskInfo{
		Total: d.Total,
		Free:  d.Free,
		Used:  d.Used,
	}, nil
}

//
// Below, we implement the means of setting and retrieving system resource
// limits.
//

type ResourceLimits struct {
	Threads int
	Memory  uint64
}

const (
	minMemory = 2 << 30 // 2 GB
)

var limits ResourceLimits

// Configures the OMR's resource limits by first checking the database, then
// the environment variables, and finally falling back to defaults based on the
// current system.
func initLimits() ResourceLimits {

	var (
		maxThreads int
		maxMemory  uint64
	)

	//
	// Max Threads
	//

	// Try the database.
	if lim, err := db.GetCpuLimit(context.Background()); err == nil {
		maxThreads = int(lim.MaxThreads)
	}

	// Try the environment variables.
	if limStr, ok := os.LookupEnv("OMR_MAX_THREADS"); ok {
		lim, err := strconv.Atoi(limStr)
		if err != nil || lim < 1 {
			Warn(
				"ignoring environment variable OMR_MAX_THREADS "+
					"because it is not a positive integer",
				"OMR_MAX_THREADS", limStr,
			)
		} else {
			// If the environment variable is valid and the database gave a
			// valid variable, we use the database value but log a warning.
			if maxThreads != 0 {
				Warn(
					"ignoring environment variable OMR_MAX_THREADS "+
						"because an existing stored value was found",
					"OMR_MAX_THREADS", lim,
					"stored value", maxThreads,
				)
			} else {
				maxThreads = lim
			}
		}
	}

	// Fall back to the default.
	if maxThreads == 0 {
		maxThreads = runtime.GOMAXPROCS(0)
	}

	//
	// Max Memory
	//

	// Try the database.
	if lim, err := db.GetMemoryLimit(context.Background()); err == nil {
		if uint64(lim.MaxMemory) < minMemory {
			Warn(
				"ignoring stored configuration for memory limit "+
					"because it is below the allowed minimum",
				"value found", uint64(lim.MaxMemory),
				"allowed minumum", minMemory,
			)
			db.DeleteMemoryLimit(context.Background(), lim.EntryID)
		} else {
			maxMemory = uint64(lim.MaxMemory)
		}
	}

	// Try the environment variables.
	if limStr, ok := os.LookupEnv("OMR_MAX_MEMORY"); ok {
		lim, err := strconv.ParseUint(limStr, 10, 64)
		if err != nil || lim < minMemory {
			Warn(
				"ignoring environment variable OMR_MAX_MEMORY "+
					"because it is below the allowed minimum",
				"OMR_MAX_MEMORY", limStr,
				"allowed minumum", minMemory,
			)
		} else {
			// If the environment variable is valid and the database gave a
			// valid variable, we use the database value but log a warning.
			if maxThreads != 0 {
				Warn(
					"ignoring environment variable OMR_MAX_MEMORY "+
						"because a configured value was found",
					"OMR_MAX_MEMORY", lim,
					"cached value", maxThreads,
				)
			} else {
				maxMemory = lim
			}
		}
	}

	// Fall back to the default.
	if maxMemory == 0 {
		vmem, _ := mem.VirtualMemory()
		maxMemory = min(
			pdf.EstimateMemCost(2, maxThreads)+512<<20,
			vmem.Available/2,
		)
	}

	return ResourceLimits{
		Threads: maxThreads,
		Memory:  maxMemory,
	}
}

func MaxThreads() int {
	return limits.Threads
}

func SetMaxThreads(threads int) error {
	if threads < 1 {
		return fmt.Errorf("thread count cannot be less than 1")
	}
	if threads > runtime.GOMAXPROCS(0) {
		return fmt.Errorf(
			"thread count cannot be greater than the hardware limit (%d)",
			runtime.GOMAXPROCS(0),
		)
	}
	db.CreateCpuLimit(context.Background(), int64(threads))
	Log(
		"configured thread limit updated",
		"previously", limits.Threads,
		"currently", threads,
	)
	limits.Threads = threads
	return nil
}

func MaxMemory() uint64 {
	return limits.Memory
}

func SetMaxMemory(memory uint64) error {
	if memory < minMemory {
		return fmt.Errorf(
			"memory cannot be set to less than the minimum (%d MB)",
			minMemory/1024/1024,
		)
	}
	db.CreateMemoryLimit(context.Background(), int64(memory))
	Log(
		"configured memory limit updated",
		"previously", fmt.Sprintf("%d MB", limits.Memory/1024/1024),
		"currently", fmt.Sprintf("%d MB", memory/1024/1024),
	)
	limits.Memory = memory
	return nil
}
