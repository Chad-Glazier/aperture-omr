package pdf

import (
	"bytes"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

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
	baseline  uint64
	increment uint64
)

func init() {
	slog.Info("sampling pdf render cost...")
	baseline, increment = sampleMemCostVars()
	slog.Info(
		"sampling complete",
		"baseline", fmt.Sprintf("%dMB", baseline/1024/1024),
		"increment", fmt.Sprintf("%dMB", increment/1024/1024),
	)
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
				1,
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
	// costs than it is to overestimate them.
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

func estimateMemCost(batchSize, parallelization int) uint64 {

	cost := uint64(baseline)
	cost += increment * uint64(batchSize*parallelization-1)

	return cost
}
