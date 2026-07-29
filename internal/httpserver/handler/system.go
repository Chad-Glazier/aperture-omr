package handler

import (
	"net/http"
	"strconv"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/sys"
)

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
	MemoryPeak uint64 `json:"memoryPeak"`
	Uptime     uint64 `json:"uptime"`
}

const defaultLimit = 30

func GetResourceUtilization(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		limit := defaultLimit
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limitVal, err := strconv.Atoi(limitStr)
			if err != nil || limitVal < 1 {
				http.Error(
					w,
					"limit parameter must be a positive integer",
					http.StatusBadRequest,
				)
				return
			}
			limit = limitVal
		}

		result := ResourceUtilization{}

		result.CpuHistory = sys.CpuHistory(limit)
		result.MemoryHistory = sys.MemHistory(limit)

		diskHistory := sys.DiskHistory(1)
		if len(diskHistory) > 0 {
			result.Disk.Usage = diskHistory[0]
		}

		result.MemoryPeak = sys.PeakMem()
		result.Uptime = uint64(sys.Uptime().Seconds())

		dbSize := s.DBSize()
		nPictures, picturesSize := s.CountPictures()
		nMats, matsSize := s.CountMats()

		result.Disk.OmrUsage.Database = dbSize
		result.Disk.OmrUsage.Matrices = matsSize
		result.Disk.OmrUsage.NumberOfMatrices = nMats
		result.Disk.OmrUsage.Pictures = picturesSize
		result.Disk.OmrUsage.NumberOfPictures = nPictures
		result.Disk.OmrUsage.Total = dbSize + matsSize + picturesSize

		dto.SendCompressedJson(w, r, result)
	}
}

func GetLogs(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		limit := defaultLimit
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limitVal, err := strconv.Atoi(limitStr)
			if err != nil || limitVal < 1 {
				http.Error(
					w,
					"limit parameter must be a positive integer",
					http.StatusBadRequest,
				)
				return
			}
			limit = limitVal
		}

		w.Header().Add("Content-Type", "text/plain")
		sys.DumpLogs(w, limit)
	}
}


