package pdf

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"runtime"
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

//go:embed testdata/samples/*
var samples embed.FS

// Stores values used to predict the memory cost of a PDF rendering operation.
type MemCostVars struct {
	Baseline  uint64 // The cost of rendering a single page PDF.
	Increment uint64 // The added cost from rendering one more page.
}

var memCost = MemCostVars{
	Baseline:  320 << 20, // 320 MB
	Increment: 160 << 20, // 160 MB
}

// Configures the pdf package to estimate rendering costs with the given values
// instead of the defaults.
func SetMemoryCostVars(newMemCost MemCostVars) {
	memCost = newMemCost
}

// Computes values used for estimating the memory cost of a PDF batch rendering
// operation by running a few samples. On failure, this function panics.
func MustRunSampling() MemCostVars {

	rssValues := [3][3]uint64{}

	for i := range 3 {
		for j := range 3 {
			buf, err := samples.ReadFile(
				fmt.Sprintf("testdata/samples/%d_page.pdf", i+1),
			)
			if err != nil {
				panic(err)
			}

			b, _, err := RenderPageBlocks(
				bytes.NewReader(buf),
				MaxDpi,
				uint(i)+1,
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

	return MemCostVars{
		Baseline:  maxBaseline,
		Increment: maxIncrement,
	}
}

// Gives a generous estimate for the peak memory usage of a batch rendering
// process.
func EstimateMemCost(batchSize, parallelization int) uint64 {
	cost := memCost.Baseline
	cost += memCost.Increment * uint64(batchSize*parallelization-1)
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
		if EstimateMemCost(batchSize, i) > allottedMemory {
			return i - 1
		}
	}
	return runtime.GOMAXPROCS(0)
}
