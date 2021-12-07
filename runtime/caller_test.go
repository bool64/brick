package runtime_test

import (
	"testing"

	"github.com/acme-corp-tech/brick/runtime"
	"github.com/stretchr/testify/assert"
)

func TestCallerFunc(t *testing.T) {
	parent(t)
}

func parent(tb testing.TB) {
	tb.Helper()

	child(tb)
}

func child(tb testing.TB) {
	tb.Helper()

	assert.Equal(tb, `brick/runtime_test.parent`, runtime.CallerFunc(2))
}

// BenchmarkCaller-4   	  500000	      2043 ns/op	     488 B/op	       7 allocs/op.
func BenchmarkCallerFunc(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parent(b)
	}
}
