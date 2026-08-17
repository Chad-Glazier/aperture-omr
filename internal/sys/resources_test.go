package sys

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestUptime(t *testing.T) {
	uptimeA := Uptime()
	assert.Assert(t, uptimeA > 0)

	time.Sleep(50*time.Millisecond)
	uptimeB := Uptime()
	assert.Assert(t, uptimeB > uptimeA)
}

func TestCurrentCpuInfo(t *testing.T) {
	info, err := currentCpuInfo()
	assert.NilError(t, err)
	assert.Assert(t, info.Description != "")
	assert.Assert(t, info.OverallPercent >= 0 && info.OverallPercent <= 100)
	assert.Assert(t, info.FrequencyMhz > 0)
	assert.Assert(t, len(info.Threads) > 0)

	for _, thread := range info.Threads {
		assert.Assert(t, thread.Percent >= 0 && thread.Percent <= 100)
	}
}

func TestCurrentMemInfo(t *testing.T) {
	info, err := currentMemInfo()
	assert.NilError(t, err)
	assert.Assert(t, info.Free > 0)
	assert.Assert(t, info.InUseOther > 0)
}

func TestCurrentDiskInfo(t *testing.T) {
	info, err := currentDiskInfo()
	assert.NilError(t, err)
	assert.Assert(t, info.Total > 0)
	assert.Assert(t, info.Free < info.Total)
	assert.Assert(t, info.Used < info.Total)
}

func TestCpuHistory(t *testing.T) {
	h := CpuHistory(100)
	assert.Assert(t, len(h) <= 100)

	for _, info := range h {
		assert.Assert(t, info.Description != "")
		assert.Assert(t, info.OverallPercent >= 0 && info.OverallPercent <= 100)
	}
}

func TestMemHistory(t *testing.T) {
	h := MemHistory(100)

	if len(h) > 100 {
		t.Fatalf("returned %d entries, expected at most 100", len(h))
	}

	for i, info := range h {
		if info.Free == 0 {
			t.Errorf("entry %d has zero total memory", i)
		}
	}
}

func TestDiskHistory(t *testing.T) {
	h := DiskHistory(100)

	if len(h) > 100 {
		t.Fatalf("returned %d entries, expected at most 100", len(h))
	}

	for i, info := range h {
		if info.Total == 0 {
			t.Errorf("entry %d has zero total disk", i)
		}

		if Docker() {

			//
			// Docker may give inconsistent counts for free/used/total memory,
			// so the following condition is not guaranteed to hold. If there
			// is a substitute condition or an alternative, consistent
			// implementation, I do not know of it.
			//

			continue
		}

		if info.Free+info.Used != info.Total {
			t.Errorf("entry %d disk accounting invalid", i)
		}
	}
}

func TestHistoryRequestZero(t *testing.T) {
	assert.Assert(t, len(CpuHistory(0)) == 0)
	assert.Assert(t, len(MemHistory(0)) == 0)
	assert.Assert(t, len(DiskHistory(0)) == 0)
	assert.Assert(t, len(CpuHistory(-1)) == 0)
	assert.Assert(t, len(MemHistory(-1)) == 0)
	assert.Assert(t, len(DiskHistory(-1)) == 0)
}
