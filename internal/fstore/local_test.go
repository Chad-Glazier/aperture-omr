package fstore

import (
	"embed"
	"io"
	"testing"

	"gocv.io/x/gocv"
	"gotest.tools/v3/assert"
)

//go:embed testdata/*
var testData embed.FS

//
// Helper functions
//

func assertMatsAreEqual(t *testing.T, a, b gocv.Mat) {

	assert.Assert(t, a.Type() == b.Type())
	assert.Assert(t, a.Rows() == b.Rows())
	assert.Assert(t, a.Cols() == b.Cols())
	assert.Assert(t, a.Channels() == b.Channels())

	for i := range a.Rows() {
		for j := range a.Cols() {
			for k := range a.Channels() {
				assert.Assert(t,
					a.GetUCharAt3(i, j, k) == b.GetUCharAt3(i, j, k),
				)
			}
		}
	}
}

//
// Tests
//

func TestLocalImageStore(t *testing.T) {

	store, err := NewLocalImageStore(t.TempDir())
	assert.NilError(t, err)

	r, err := testData.Open("testdata/sample_image.jpg")
	assert.NilError(t, err)
	defer r.Close()

	srcImg, err := DecodeImg(r)
	assert.NilError(t, err)

	t.Run("basic read write delete", func(t *testing.T) {
		const name = "sample"

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)

		err = store.Set(name, srcImg)
		assert.NilError(t, err)

		_, err = store.Get(name)
		assert.NilError(t, err)

		store.Delete(name)

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)
	})

	srcImgBytes, err := testData.ReadFile("testdata/sample_image.jpg")
	assert.NilError(t, err)

	t.Run("set bytes", func(t *testing.T) {
		const name = "from_bytes"

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)

		err = store.SetBytes(name, srcImgBytes)
		assert.NilError(t, err)

		r, err := store.Open(name)
		assert.NilError(t, err)
		defer r.Close()

		storedBytes, err := io.ReadAll(r)
		assert.NilError(t, err)
		assert.Assert(t, len(storedBytes) == len(srcImgBytes))
		for i := range storedBytes {
			assert.Assert(t, storedBytes[i] == srcImgBytes[i])
		}

		r.Close()
		store.Delete(name)

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)
	})

	nonImgBytes, err := testData.ReadFile("testdata/not_an_image.txt")
	assert.NilError(t, err)

	t.Run("setting a non-image", func(t *testing.T) {
		const name = "evil_bytes"

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)

		err = store.SetBytes(name, nonImgBytes)
		assert.NilError(t, err)

		r, err := store.Open(name)
		assert.NilError(t, err)
		r.Close()

		_, err = store.Get(name)
		assert.Assert(t, err == ErrDecoding)

		store.Delete(name)

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)
	})
}

func TestLocalMatStore(t *testing.T) {

	store, err := NewLocalMatStore(t.TempDir())
	assert.NilError(t, err)

	buf, err := testData.ReadFile("testdata/sample_image.jpg")
	assert.NilError(t, err)

	mat, err := gocv.IMDecode(buf, gocv.IMReadGrayScale)
	assert.NilError(t, err)

	t.Run("basic read write delete", func(t *testing.T) {
		const name = "sample.m4t"

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)

		err = store.Set(name, mat)
		assert.NilError(t, err)

		_, err = store.Get(name)
		assert.NilError(t, err)

		store.Delete(name)

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)
	})

	t.Run("data preserved", func(t *testing.T) {
		const name = "sample.m4t"

		_, err = store.Get(name)
		assert.Assert(t, err == ErrNotFound)

		err = store.Set(name, mat)
		assert.NilError(t, err)

		storedMat, err := store.Get(name)
		assert.NilError(t, err)

		assertMatsAreEqual(t, mat, storedMat)
	})
}
