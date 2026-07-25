package pdf

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/database/sqlc"

	"github.com/shirou/gopsutil/v4/process"
)

//
// In this file we implement functions that (hopefully) help us predict the
// memory cost of rendering a PDF.
//

func getRss() uint64 {
	proc, _ := process.NewProcess(int32(os.Getpid()))
	m, _ := proc.MemoryInfo()
	return m.RSS
}

var (
	baseline  uint64 = 320 << 20 // 320 MB
	increment uint64 = 160 << 20 // 160 MB
)

// Initializes the package by sampling PDF render operations to get an estimate
// of the memory cost. If the given database has cached results from such 
// sampling, then those values will be used instead.
//
// It's important that this function be run while no other major operations are
// in progress. Such operations would pollute the memory profiling and thereby 
// lead to poor estimations.
//
// If this function is never run, then the very generous default estimates will
// be used to predict rendering costs.
func Init(db database.Querier) {

	slog.Info("loading cached pdf render cost...")
	info, err := db.GetCachedSystemInfo(context.Background())
	if err == nil {
		baseline = uint64(info.PdfRenderBaseline)
		increment = uint64(info.PdfRenderIncrement)
		slog.Info(
			"cache loaded", 
			"baseline", fmt.Sprintf("%dMB", baseline/1024/1024),
			"increment", fmt.Sprintf("%dMB", increment/1024/1024),		
		)
		return
	}
	slog.Info("cache empty; sampling pdf render cost...")
	baseline, increment = sampleMemCostVars()
	slog.Info(
		"sampling complete",
		"baseline", fmt.Sprintf("%dMB", baseline/1024/1024),
		"increment", fmt.Sprintf("%dMB", increment/1024/1024),
	)
	slog.Info("caching results")
	db.SetCachedSystemInfo(context.Background(), sqlc.SetCachedSystemInfoParams{
		PdfRenderBaseline: int64(baseline),
		PdfRenderIncrement: int64(increment),
	})

}

//go:embed testdata/samples/*
var samples embed.FS

// Computes values used for estimating the memory cost of a PDF batch rendering
// operation by running a few samples. The first value returned is the baseline
// cost of starting up ImageMagick/GhostScript and rendering a single page. The
// second value returned is the cost of adding one more page.
func sampleMemCostVars() (uint64, uint64) {

	rssValues := [3][3]uint64{}

	for i := range 3 {
		for j := range 3 {
			buf, err := samples.ReadFile(
				fmt.Sprintf("testdata/samples/%d_page.pdf", i+1),
			)
			if err != nil {
				panic(err)
			}

			b, _, err := RenderPageBatches(
				bytes.NewReader(buf),
				MaxDpi,
				i+1,
				0,
			)
			if err != nil {
				panic(err)
			}

			var (
				baseRss uint64 = getRss()
				peakRss uint64 = 0
			)
			ticker := time.NewTicker(5 * time.Millisecond)

		out:
			for {
				select {
				case <-ticker.C:
					peakRss = max(peakRss, getRss())
				case batch := <-b:
					peakRss = max(peakRss, getRss())
					batch.Close()
					break out
				}
			}

			rssValues[i][j] = peakRss - baseRss

			debug.FreeOSMemory()
		}
	}

	//
	// We use the max values because it's just better to overestimate memory
	// costs than it is to underestimate them.
	//

	var (
		maxBaseline  uint64 = 0
		minBaseline  uint64 = 1 << 63
		maxIncrement uint64 = 0
	)

	for _, onePageRss := range rssValues[0] {
		maxBaseline = max(maxBaseline, onePageRss)
		minBaseline = min(minBaseline, onePageRss)
	}
	for i := range rssValues[1:] {

		minRss := uint64(1 << 63)
		for _, rss := range rssValues[i] {
			minRss = min(minRss, rss)
		}

		maxRss := uint64(0)
		for _, rss := range rssValues[i+1] {
			maxRss = max(maxRss, rss)
		}

		maxIncrement = maxRss - minRss
	}

	return maxBaseline, maxIncrement
}

// Gives a generous estimate for the peak memory usage of a batch rendering
// process.
func estimateMemCost(batchSize, parallelization int) uint64 {

	cost := uint64(baseline)
	cost += increment * uint64(batchSize*parallelization-1)

	return cost
}

// Returns the maximum number of concurrent batches that can be rendered 
// without exceeding the allotted memory. If the batch size is too large or the
// allotted memory is too small, then the operation is impossible and the 
// returned value is 0.
func maxParallelization(
	batchSize int,
	allottedMemory uint64,
) int {
	for i := 1; i < runtime.GOMAXPROCS(0); i++ {
		if estimateMemCost(batchSize, i) > allottedMemory {
			return i - 1
		}
	}
	return runtime.GOMAXPROCS(0)
}
