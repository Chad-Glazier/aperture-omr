package handler

import (
	"net/http"
	"runtime"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const defaultLimit = 30

//
// Public System Info
//

func GetResourceUtilization(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		var result dto.ResourceUtilization

		result.CpuHistory = sys.CpuHistory(int(q.Limit))
		result.MemoryHistory = sys.MemHistory(int(q.Limit))

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

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		w.Header().Add("Content-Type", "text/plain")
		sys.DumpLogs(w, int(q.Limit))
	}
}

type CpuUsageSample struct {
	Overall   float64   `json:"overall"`
	PerThread []float64 `json:"perThread"`
}

type DetailedCpuInfo struct {
	Description  string           `json:"description"`
	FrequencyMhz float64          `json:"frequencyMhz"`
	MaxThreads   int              `json:"maxThreads"`
	UsageSamples []CpuUsageSample `json:"usageSamples"`
}

func GetCpuInfo(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		samples := sys.CpuHistory(int(q.Limit))
		result := DetailedCpuInfo{
			UsageSamples: make([]CpuUsageSample, len(samples)),
		}
		for i, s := range samples {
			result.UsageSamples[i].Overall = s.OverallPercent
			result.UsageSamples[i].PerThread = make([]float64, len(s.Threads))
			for j, t := range s.Threads {
				result.UsageSamples[i].PerThread[j] = t.Percent
			}
		}
		lastSample := samples[len(samples)-1]

		result.Description = lastSample.Description
		result.FrequencyMhz = lastSample.FrequencyMhz
		result.MaxThreads = runtime.GOMAXPROCS(0)

		dto.SendCompressedJson(w, r, result)
	}
}

type DetailedMemoryInfo struct {
	UsageSamples []sys.MemInfo `json:"usageSamples"`
}

func GetMemoryInfo(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		samples := sys.MemHistory(int(q.Limit))

		dto.SendCompressedJson(w, r, DetailedMemoryInfo{
			UsageSamples: samples,
		})

	}
}

//
// Admin Endpoints
//

func CheckAdminKey(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticated := s.CheckAdminKey(r); !authenticated {
			http.Error(w,
				"incorrect admin key",
				http.StatusUnauthorized,
			)
		}
	}
}
