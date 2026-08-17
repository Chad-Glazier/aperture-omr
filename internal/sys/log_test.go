package sys

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDumpLogsEmpty(t *testing.T) {
	logCache.Flush()
	var buf bytes.Buffer

	DumpLogs(&buf, 10)

	assert.Assert(t, buf.String() == "")
}

func TestLogAppearsInDump(t *testing.T) {
	logCache.Flush()
	var buf bytes.Buffer

	Log("foul tarnished")
	DumpLogs(&buf, 1)

	got := buf.String()

	assert.Assert(t, strings.Contains(got, "foul tarnished"))
}

func TestDumpLogsLimit(t *testing.T) {
	logCache.Flush()

	Log("too hot")
	Info("too cold")
	Warn("just right")

	var buf bytes.Buffer
	DumpLogs(&buf, 2)

	out := buf.String()
	assert.Assert(t, !strings.Contains(out, "too hot"))
	assert.Assert(t, strings.Contains(out, "too cold"))
	assert.Assert(t, strings.Contains(out, "just right"))
}

func TestDumpLogsZero(t *testing.T) {
	logCache.Flush()
	Error("hello")

	var buf bytes.Buffer
	DumpLogs(&buf, 0)

	assert.Assert(t, len(buf.Bytes()) == 0)
}

func TestDumpLogsNegative(t *testing.T) {
	logCache.Flush()
	Debug("hello")

	var buf bytes.Buffer
	DumpLogs(&buf, -1)

	assert.Assert(t, len(buf.Bytes()) == 0)
}
