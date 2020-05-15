package helpers

import (
	"fmt"
	"os"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

//Check logger for errors
func Check(e error, panic bool, logs string, trace TraceData) {
	if e != nil && panic {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
	if e != nil && !panic {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//PrintDownloadPercent self explanatory
func PrintDownloadPercent(done chan int64, path string, total int64) {
	var stop = false
	if total == -1 {
		log.Warn("-1 Content length, can't load download bar, will download silently")
		return
	}
	for {
		select {
		case <-done:
			stop = true
		default:
			file, err := os.Open(path)
			Check(err, true, "Opening file path", Trace())
			fi, err := file.Stat()
			Check(err, true, "Getting file statistics", Trace())
			size := fi.Size()
			if size == 0 {
				size = 1
			}
			var percent = float64(size) / float64(total) * 100
			if percent != 100 {
				fmt.Printf("\r%.0f%% %s", percent, path)
			}
		}
		if stop {
			break
		}
		time.Sleep(time.Second)
	}
}
