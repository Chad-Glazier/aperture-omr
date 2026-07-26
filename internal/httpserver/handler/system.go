package handler

import (
	"net/http"
	"os"
	"ubco-team15/omr/internal/httpserver/dto"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type ResourceUtilization struct {
	CPU struct {
		Description string `json:"description"`
		OverallPercent float64 `json:"overallPercent"`
		FrequencyMhz float64 `json:"mhz"`
		Threads []struct{
			Percent float64 `json:"percent"`
		} `json:"threads"`
	} `json:"cpu"`
	Memory struct {
		InUseOmr       uint64 `json:"inUseOmr"`
		InUseOther     uint64 `json:"inUseOther"`
		TotalAvailable uint64 `json:"totalAvailable"`
	} `json:"memory"`
	Disk struct {
		Database         uint64 `json:"database"`
		NumberOfMatrices int `json:"numberOfMatrices"`
		Matrices         uint64 `json:"matrices"`
		NumberOfPictures int `json:"numberOfPictures"`
		Pictures         uint64 `json:"pictures"`

		Total uint64 `json:"total"`
		Used  uint64 `json:"used"`
	} `json:"disk"`
}

func sendErr(w http.ResponseWriter) {
	http.Error(
		w, 
		"unexpected error calculating system resources", 
		http.StatusInternalServerError,
	)
}

func GetResourceUtilization(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Get server resource utilization:
		// - Memory (RSS bytes; total available memory)
		// - CPU (per core; cumulative; cpu stats)
		// - Disk
		//   - templates/scans stored (abstract data)
		//   - matrices/pictures stored + bytes (raw data)
		//   - bytes for database rows
		//   - total bytes and total available disk space
		//

		result := &ResourceUtilization{}

		//
		// CPU
		//

		cpuInfo, err := cpu.Info()
		if err != nil {
			sendErr(w)
			return
		}

		if len(cpuInfo) > 0 {
			result.CPU.Description = cpuInfo[0].ModelName
			result.CPU.FrequencyMhz = cpuInfo[0].Mhz
		}

		overall, err := cpu.Percent(0, false)
		if err != nil {
			sendErr(w)
			return
		}

		if len(overall) > 0 {
			result.CPU.OverallPercent = overall[0]
		}

		corePercents, err := cpu.Percent(0, true)
		if err != nil {
			sendErr(w)
			return
		}

		result.CPU.Threads = make([]struct {
			Percent float64 `json:"percent"`
		}, len(corePercents))

		for i, percent := range corePercents {
			result.CPU.Threads[i].Percent = percent
		}

		//
		// Memory
		//

		vmem, err := mem.VirtualMemory()
		if err != nil {
			sendErr(w)
			return
		}

		result.Memory.TotalAvailable = vmem.Available

		proc, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			sendErr(w)
			return
		}

		rss, err := proc.MemoryInfo()
		if err != nil {
			sendErr(w)
			return
		}

		result.Memory.InUseOmr = rss.RSS
		result.Memory.InUseOther = vmem.Used - rss.RSS

		//
		// Disk
		//

		result.Disk.Database = s.DBSize()
		result.Disk.NumberOfPictures, result.Disk.Pictures = s.CountPictures()
		result.Disk.NumberOfMatrices, result.Disk.Matrices = s.CountMats()

		d, err := disk.Usage("/")
		result.Disk.Total = d.Total
		result.Disk.Used = d.Used

		dto.SendJson(w, result)

	}
}
