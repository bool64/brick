package runtime

import (
	"math"
	"runtime"
	"time"
)

// StableHeapInUse returns the stable size of used heap in bytes.
//
// Heap is considered stable if it changes for less than 10KB between measurements.
func StableHeapInUse() int64 {
	var (
		m         = runtime.MemStats{}
		prevInUse uint64
		prevNumGC uint32
	)

	for {
		runtime.ReadMemStats(&m)

		if prevNumGC != 0 && m.NumGC > prevNumGC && math.Abs(float64(m.HeapInuse-prevInUse)) < 10*1024 {
			break
		}

		prevInUse = m.HeapInuse
		prevNumGC = m.NumGC

		time.Sleep(50 * time.Millisecond)

		runtime.GC()
	}

	return int64(m.HeapInuse)
}
