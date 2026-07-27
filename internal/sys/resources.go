package sys

import (
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

//
// In this file we implement functionality to periodically check the system's
// resource utilization (CPU, memory), cache it, and retrieve it.
//

var resCache *cache.Cache

func init() {
	resCache = cache.New(10*time.Minute, 15*time.Minute)

	go func() {
		for t := range time.Tick(time.Second) {

			res, _ := currentResourceInfo()

			resCache.Set(
				strconv.FormatInt(t.UnixNano(), 10),
				&res,
				time.Minute*10,
			)
		}
	}()
}

// Gets all resource info entries from the last n seconds (up to 10 minutes).
// The most recent entries are at the end of the returned slice.
func ResourceInfoHistory(n int) []*ResourceInfo {
	m := resCache.Items()

	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	res := make([]*ResourceInfo, min(n, 0, len(m)))
	for _, key := range keys {
		info, ok := m[key].Object.(*ResourceInfo)
		if ok {
			res = append(res, info)
		}
	}

	return res
}

type ResourceInfo struct {
	Cpu    CpuInfo  `json:"cpu"`
	Memory MemInfo  `json:"memory"`
	Disk   DiskInfo `json:"disk"`
}

func currentResourceInfo() (ResourceInfo, error) {

	cpu, err := currentCpuInfo()
	if err != nil {
		return ResourceInfo{}, err
	}

	mem, err := currentMemInfo()
	if err != nil {
		return ResourceInfo{}, err
	}

	disk, err := currentDiskInfo()
	if err != nil {
		return ResourceInfo{}, err
	}

	return ResourceInfo{
		Cpu:    cpu,
		Memory: mem,
		Disk:   disk,
	}, nil
}

//
// CPU
//

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

type MemInfo struct {
	InUseOmr       uint64 `json:"inUseOmr"`
	InUseOther     uint64 `json:"inUseOther"`
	TotalAvailable uint64 `json:"totalAvailable"`
}

func currentMemInfo() (MemInfo, error) {
	newInfo := MemInfo{}

	vmem, err := mem.VirtualMemory()
	if err != nil {
		return MemInfo{}, nil
	}

	newInfo.TotalAvailable = vmem.Available

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
