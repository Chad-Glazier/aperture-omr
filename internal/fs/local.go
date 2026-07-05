package fs

import (
	"compress/lzw"
	"encoding/binary"
	"image"
	"image/draw"
	"io"
	"os"
	"path/filepath"

	"gocv.io/x/gocv"
)

//
// This file implements the Store interface by using local storage.
//

type localStore struct {
	// The root directory of the storage.
	root string
}

var _ Store = (*localStore)(nil)
var _ MatSaveLoader = (*localStore)(nil)

// Opens the named root directory as a local file store.
func NewLocalStore(root string) *localStore {
	return &localStore{
		root: root,
	}
}

func (s *localStore) ImgExists(key string) bool {
	_, err := os.Stat(filepath.Join(s.root, key))
	return err == nil
}

func (s *localStore) GetImg(key string) (image.Image, error) {
	f, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := DecodeImg(f)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (s *localStore) PutImg(key string, img image.Image) error {
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}

	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return err
	}
	defer w.Close()

	err = EncodeImg(w, img)
	return err
}

func (s *localStore) DeleteImg(key string) error {
	return os.Remove(filepath.Join(s.root, key))
}

func (s *localStore) ImgSnippet(
	key string,
	x, y, width, height int,
) (image.Image, error) {
	img, err := s.GetImg(key)
	if err != nil {
		return nil, err
	}

	rect := image.Rect(0, 0, width, height)

	cropped := image.NewRGBA(rect)

	draw.Draw(
		cropped,
		rect,
		img,
		image.Point{X: x, Y: y},
		draw.Src,
	)

	return cropped, nil
}

//
// MatSaveLoader implementation.
//

//
// OpenCV matrices have the following data that needs to be stored in order to
// recreate them:
// - the rows and columns in the matrix,
// - the matrix type flag, and
// - the bytes that store the data.
// In order to store this data, we will write the it into a binary file,
// writing the values for the matrix in the order that they are listed above.
//
//        [int32][int32][int32][bytes...]
//         │      │      │      └─ the bytes buffer for the matrix
//         │      │      └──────── the matrix type flag
//         │      └─────────────── the number of columns
//         └────────────────────── the number of rows
//
// The integers are stored in little endian format. The bytes buffer is 
// compressed with the LZW algorithm.
//

func (s *localStore) MatSave(key string, mat *gocv.Mat) error {

	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}

	buf, err := mat.DataPtrUint8()
	if err != nil {
		return err
	}

	prefix := make([]byte, 12)
	binary.Encode(prefix[0:4], binary.LittleEndian, int32(mat.Rows()))
	binary.Encode(prefix[4:8], binary.LittleEndian, int32(mat.Cols()))
	binary.Encode(prefix[8:12], binary.LittleEndian, int32(mat.Type()))

	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := w.Write(prefix); err != nil {
		return err
	}

	lzwWriter := lzw.NewWriter(w, lzw.LSB, 8)
	defer lzwWriter.Close()

	if _, err := lzwWriter.Write(buf); err != nil {
		return err
	}

	return nil
}

func (s *localStore) MatLoad(key string) (*gocv.Mat, error) {

	f, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	prefix := make([]byte, 12)
	if _, err := f.Read(prefix); err != nil {
		return nil, err
	}
	var rows, cols, mt int32
	binary.Decode(prefix[0:4], binary.LittleEndian, &rows)
	binary.Decode(prefix[4:8], binary.LittleEndian, &cols)
	binary.Decode(prefix[8:12], binary.LittleEndian, &mt)

	lzwReader := lzw.NewReader(f, lzw.LSB, 8)
	defer lzwReader.Close()

	buf, err := io.ReadAll(lzwReader)
	if err != nil {
		return nil, err
	}

	mat, err := gocv.NewMatFromBytes(
		int(rows),
		int(cols),
		gocv.MatType(mt),
		buf,
	)
	if err != nil {
		return nil, err
	}

	return &mat, nil
}
