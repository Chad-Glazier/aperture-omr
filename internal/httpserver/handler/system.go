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

func GetResourceUtilization(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		result := ResourceUtilization{}

		result.CpuHistory = sys.CpuHistory(history)
		result.MemoryHistory = sys.MemHistory(history)

		diskHistory := sys.DiskHistory(1)
		if len(diskHistory) > 0 {
			result.Disk.Usage = diskHistory[0]
		}

		dbSize := s.DBSize()
		nPictures, picturesSize := s.CountPictures()
		nMats, matsSize := s.CountMats()

		result.Disk.OmrUsage.Database = dbSize
		result.Disk.OmrUsage.Matrices = matsSize
		result.Disk.OmrUsage.NumberOfMatrices = nMats
		result.Disk.OmrUsage.Pictures = picturesSize
		result.Disk.OmrUsage.NumberOfPictures = nPictures
		result.Disk.OmrUsage.Total = dbSize + matsSize + picturesSize

		dto.SendDeflatedJson(w, r, result)
	}
}

func GetLogs(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		sys.DumpLogs(w, history)
	}
}
