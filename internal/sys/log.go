package sys

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

//
// This file implements general logging functionality suitable for use across
// the entire application.
//

var logCache *cache.Cache

func init() {
	logCache = cache.New(5*time.Minute, 10*time.Minute)
}

type level int

const (
	levelMisc level = iota
	levelInfo
	levelDebug
	levelWarn
	levelError
)

func log(level level, msg string, args ...any) {
	t := time.Now().Local().Format("2006-01-02 15:04:05.000")

	s := strings.Builder{}
	s.WriteString(t)
	s.WriteString(" ")
	switch level {
	case levelMisc:
		s.WriteString("[MISC]  ")
	case levelInfo:
		s.WriteString("[INFO]  ")
	case levelDebug:
		s.WriteString("[DEBUG] ")
	case levelWarn:
		s.WriteString("[WARN]  ")
	case levelError:
		s.WriteString("[ERROR] ")
	}
	s.WriteString(msg)

	if len(args) == 1 {
		fmt.Fprintf(&s, " -- %v", args[0])
	}
	if len(args) >= 2 {
		for i := 0; i < len(args)-1; i += 2 {
			fmt.Fprintf(&s, " -- %v: %v", args[i], args[i+1])
		}
		if len(args)%2 == 1 {
			fmt.Fprintf(&s, " -- %v", args[len(args)-1])
		}
	}

	str := s.String()

	switch level {
	case levelMisc:
		fmt.Println(FgBrightBlack(str))
		logCache.Set(t, str, 10*time.Minute)
	case levelInfo:
		fmt.Println(FgCyan(str))
		logCache.Set(t, str, 1*time.Hour)
	case levelDebug:
		fmt.Println(FgBrightBlue(str))
		logCache.Set(t, str, 12*time.Hour)
	case levelWarn:
		fmt.Println(FgBrightYellow(str))
		logCache.Set(t, str, 12*time.Hour)
	case levelError:
		fmt.Println(FgBrightRed(str))
		logCache.Set(t, str, 24*time.Hour)
	}
}

func Log(msg string, args ...any) {
	log(levelMisc, msg, args...)
}

func Info(msg string, args ...any) {
	log(levelInfo, msg, args...)
}

func Debug(msg string, args ...any) {
	log(levelDebug, msg, args...)
}

func Warn(msg string, args ...any) {
	log(levelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	log(levelError, msg, args...)
}

func DumpLogs(w io.Writer) {

	//
	// Since the memory cache is unsorted, we have to sort the map entries by
	// their keys (which should be set to the time of the logging) before
	// writing them. Since the keys are set as the timestamp when the log was
	// written, this should be equivalent to a chronological ordering.
	//

	m := logCache.Items()

	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	bw := bufio.NewWriter(w)
	for _, key := range keys {
		str, ok := m[key].Object.(string)
		if ok {
			bw.Write([]byte(str + "\n"))
		}
	}
	bw.Flush()
}
