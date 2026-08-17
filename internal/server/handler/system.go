package handler

import (
	"net/http"
	"runtime"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const defaultLimit = 30

//
// Public System Info
//

func OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, "./api/openapi.yaml")
}

func Ping(w http.ResponseWriter, r *http.Request) {}

func GetResourceUtilization(s resources.ServerResources) http.HandlerFunc {
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

func GetLogs(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		w.Header().Add("Content-Type", "text/plain")
		sys.DumpLogs(w, int(q.Limit))
	}
}

func GetCpuInfo(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		samples := sys.CpuHistory(int(q.Limit))
		result := dto.DetailedCpuInfo{
			UsageSamples: make([]dto.CpuUsageSample, len(samples)),
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

func GetMemoryInfo(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.LimitQuery](w, r)
		if !ok {
			return
		}

		samples := sys.MemHistory(int(q.Limit))

		dto.SendCompressedJson(w, r, dto.DetailedMemoryInfo{
			UsageSamples: samples,
		})

	}
}

//
// Admin Endpoints
//

func CheckAdminKey(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticated := s.CheckAdminKey(r); !authenticated {
			http.Error(w,
				"incorrect admin key",
				http.StatusUnauthorized,
			)
		}
	}
}
