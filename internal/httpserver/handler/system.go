package handler

import (
	"net/http"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/sys"
)

const history = 30

type ResourceUtilization struct {
	CpuHistory    []sys.CpuInfo `json:"cpuHistory"`
	MemoryHistory []sys.MemInfo `json:"memoryHistory"`
	Disk          struct {
		Usage    sys.DiskInfo `json:"usage"`
		OmrUsage struct {
			Database         uint64 `json:"database"`
			NumberOfMatrices int    `json:"numberOfMatrices"`
			Matrices         uint64 `json:"matrices"`
			NumberOfPictures int    `json:"numberOfPictures"`
			Pictures         uint64 `json:"pictures"`
			Total            uint64 `json:"total"`
		} `json:"omrUsage"`
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

		resourceHistory := sys.ResourceInfoHistory(history)
		dbSize := s.DBSize()
		nPictures, picturesSize := s.CountPictures()
		nMats, matsSize := s.CountMats()

		result := ResourceUtilization{}

		result.CpuHistory = make([]sys.CpuInfo, len(resourceHistory))
		result.MemoryHistory = make([]sys.MemInfo, len(resourceHistory))
		for i := range resourceHistory {
			result.CpuHistory[i] = resourceHistory[i].Cpu
			result.MemoryHistory[i] = resourceHistory[i].Memory
		}

		result.Disk.Usage = resourceHistory[len(resourceHistory)-1].Disk

		result.Disk.OmrUsage.Database = dbSize
		result.Disk.OmrUsage.Matrices = matsSize
		result.Disk.OmrUsage.NumberOfMatrices = nMats
		result.Disk.OmrUsage.Pictures = picturesSize
		result.Disk.OmrUsage.NumberOfPictures = nPictures
		result.Disk.OmrUsage.Total = dbSize + matsSize + picturesSize

		dto.SendJson(w, result)

	}
}

func GetLogs(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		sys.DumpLogs(w)
	}
}
