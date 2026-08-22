package sys

import (
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helpers
//

func assertSlicesEqual[T comparable](t *testing.T, a, b []T) {
	t.Helper()

	assert.Assert(t, len(a) == len(b))

	for i := range a {
		assert.Assert(t, a[i] == b[i])
	}
}

//
// Tests
//

func TestRingBuffer(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		r := NewRingBuffer[int](5)

		assert.Assert(t, r.Len() == 0)
		assert.Assert(t, len(r.Get(2)) == 0)
	})

	t.Run("add and get", func(t *testing.T) {
		r := NewRingBuffer[int](5)
		for i := range 3 {
			r.Add(i)
		}

		assert.Assert(t, r.Len() == 3)
		assertSlicesEqual(t, []int{0, 1, 2}, r.Get(3))
	})

	t.Run("capacity", func(t *testing.T) {
		r := NewRingBuffer[int](3)
		for i := range 5 {
			r.Add(i)
		}

		assert.Assert(t, r.Len() == 3)
		assertSlicesEqual(t, r.Get(3), []int{2, 3, 4})
	})

	t.Run("slice is independent", func(t *testing.T) {
		r := NewRingBuffer[int](3)

		for i := range 3 {
			r.Add(i)
		}

		got := r.Get(3)
		got[0] = 100

		assertSlicesEqual(t, r.Get(3), []int{0, 1, 2})
	})

	t.Run("get more than length", func(t *testing.T) {
		r := NewRingBuffer[int](5)

		for i := range 3 {
			r.Add(i)
		}

		assertSlicesEqual(t, r.Get(10), []int{0, 1, 2})
	})

	t.Run("get nothing", func(t *testing.T) {
		r := NewRingBuffer[int](5)

		for i := range 3 {
			r.Add(i)
		}

		assert.Assert(t, len(r.Get(0)) == 0)
		assert.Assert(t, len(r.Get(-1)) == 0)
	})
}
