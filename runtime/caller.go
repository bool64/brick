// Package runtime provides observability helpers.
package runtime

import (
	"math"
	"path"
	"runtime"
	"runtime/debug"
	"time"
)

// CallerFunc returns trimmed path and name of parent function.
func CallerFunc(skip ...int) string {
	skipFrames := 2
	if len(skip) == 1 {
		skipFrames = skip[0]
	}

	pc, _, _, ok := runtime.Caller(skipFrames)
	if !ok {
		return ""
	}

	f := runtime.FuncForPC(pc)

	pathName := path.Base(path.Dir(f.Name())) + "/" + path.Base(f.Name())

	return pathName
}

// StableHeapInUse measures heap and triggers GC until HeapInUse stabilizes with 100 KB precision.
//
// This function can be useful to benchmark memory efficiency of particular data layouts.
func StableHeapInUse() uint64 {
	var (
		m         = runtime.MemStats{}
		prevInUse uint64
	)

	for {
		runtime.ReadMemStats(&m)

		if math.Abs(float64(m.HeapInuse-prevInUse)) < 100*1024 {
			break
		}

		prevInUse = m.HeapInuse

		time.Sleep(50 * time.Millisecond)
		runtime.GC()
		debug.FreeOSMemory()
	}

	return m.HeapInuse
}
