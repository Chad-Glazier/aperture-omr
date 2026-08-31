package resources

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/database"
	"github.com/Chad-Glazier/aperture-omr/internal/fstore"
	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/google/uuid"
)

//
// In this file, we implement the [ServerResources] interface using local
// resources (i.e., the disk).
//

type local struct {
	DBCnx          io.Closer
	DB             database.Querier
	DBPath         string
	Images         fstore.ImageStore
	Mats           fstore.MatStore
	adminKey       [64]byte
	globalKey      [64]byte
	globalKeyIsSet bool
}

var _ ServerResources = (*local)(nil)

// Provides implementation of ServerResources that uses an SQLite database and
// stores files locally. All data (i.e., the SQLite file and the root directory
// for stored files) will be stored below the specified root directory.
func NewLocal(rootDir string) (*local, error) {

	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}

	var (
		dbPath   = filepath.Join(rootDir, "/database.sqlite3")
		imgsPath = filepath.Join(rootDir, "/images")
		matsPath = filepath.Join(rootDir, "/matrices")
	)

	db, cnx, err := database.Connect(dbPath)
	if err != nil {
		return nil, err
	}

	images, err := fstore.NewLocalImageStore(imgsPath)
	if err != nil {
		cnx.Close()
		return nil, err
	}

	mats, err := fstore.NewLocalMatStore(matsPath)
	if err != nil {
		cnx.Close()
		return nil, err
	}

	res := &local{
		DBCnx:  cnx,
		DB:     db,
		DBPath: dbPath,
		Images: images,
		Mats:   mats,
	}
	return res, nil
}

func (s *local) Close() error {
	return s.DBCnx.Close()
}

//
// Marking Templates
//

func (s *local) SaveMarkingTemplate(tmpl omr.MarkTemplate) (string, error) {
	buf := bytes.Buffer{}
	err := omr.EncodeMarkTemplate(&buf, tmpl)
	if err != nil {
		return "", ErrSerializing
	}

	id := uuid.New().String()
	err = s.DB.CreateMarkingTemplate(
		context.Background(),
		database.CreateMarkingTemplateParams{
			ID:    id,
			Bytes: buf.Bytes(),
		},
	)
	if err != nil {
		return "", ErrDatabaseWrite
	}

	return id, nil
}

func (s *local) LoadMarkingTemplate(id string) (omr.MarkTemplate, error) {

	record, err := s.DB.GetMarkingTemplate(context.Background(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return omr.MarkTemplate{}, ErrNotFound
		}
		return omr.MarkTemplate{}, ErrDatabaseRead
	}

	out, err := omr.DecodeMarkTemplate(bytes.NewReader(record.Bytes))
	if err != nil {
		return omr.MarkTemplate{}, ErrDeserializing
	}

	return out, nil
}

func (s *local) DeleteMarkingTemplate(id string) {
	s.DB.DeleteMarkingTemplate(context.Background(), id)
}

//
// Preprocessing Templates
//

func (s *local) SavePreprocessingTemplate(
	tmpl omr.PreprocessTemplate,
) (string, error) {
	buf := bytes.Buffer{}
	err := omr.EncodePreprocessTemplate(&buf, tmpl)
	if err != nil {
		return "", ErrSerializing
	}

	id := uuid.New().String()
	err = s.DB.CreatePreprocessingTemplate(
		context.Background(),
		database.CreatePreprocessingTemplateParams{
			ID:    id,
			Bytes: buf.Bytes(),
		},
	)
	if err != nil {
		return "", ErrDatabaseWrite
	}

	return id, nil
}

func (s *local) LoadPreprocessingTemplate(
	id string,
) (omr.PreprocessTemplate, error) {

	record, err := s.DB.GetPreprocessingTemplate(context.Background(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return omr.PreprocessTemplate{}, ErrNotFound
		}
		return omr.PreprocessTemplate{}, ErrDatabaseRead
	}

	out, err := omr.DecodePreprocessTemplate(bytes.NewReader(record.Bytes))
	if err != nil {
		return omr.PreprocessTemplate{}, ErrDeserializing
	}

	return out, nil
}

func (s *local) DeletePreprocessingTemplate(id string) {
	s.DB.DeletePreprocessingTemplate(context.Background(), id)
}

//
// Scans
//

func (s *local) SaveScan(
	pages []omr.Mat,
	templateId string,
) (string, error) {

	scanId := uuid.New().String()
	err := s.DB.CreateScan(context.Background(), database.CreateScanParams{
		ID:                      scanId,
		PreprocessingTemplateID: templateId,
		CreatedAtUnixMs:         time.Now().UnixMilli(),
	})
	if err != nil {
		return "", ErrDatabaseWrite
	}

	//
	// Below we implement a function that cleans up all resources that have
	// been saved thus far in the operation. That way if there is a failure
	// partway through we won't leave behind any mess in the persistent
	// storage.
	//

	savedMats := make([]string, 0)
	savedImages := make([]string, 0)
	cleanupFailure := func() {
		for _, key := range savedMats {
			s.Mats.Delete(key)
		}
		for _, key := range savedImages {
			s.Images.Delete(key)
		}
		s.DB.DeletePagesForScan(context.Background(), scanId)
		s.DB.DeleteScan(context.Background(), scanId)
	}

	//
	// Now we iterate over the pages in the scan and store the resources.
	//

	for i, page := range pages {

		matrixId := uuid.New().String() + ".m4t"
		if err := s.Mats.Set(matrixId, page); err != nil {
			cleanupFailure()
			return "", ErrFileStorageWrite
		}
		savedMats = append(savedMats, matrixId)

		pictureId := uuid.New().String() + fstore.ImgFileExt
		w, err := s.Images.Create(pictureId)
		if err != nil {
			cleanupFailure()
			return "", ErrFileStorageWrite
		}
		savedImages = append(savedImages, pictureId)

		_, err = omr.EncodeMatToImage(w, fstore.ImgContentType, page)
		w.Close()
		if err != nil {
			cleanupFailure()
			return "", ErrEncodingImage
		}

		err = s.DB.CreateScanPage(
			context.Background(),
			database.CreateScanPageParams{
				MatrixKey:  matrixId,
				PictureKey: pictureId,
				PageIndex:  int64(i),
				ScanID:     scanId,
			},
		)
		if err != nil {
			cleanupFailure()
			return "", ErrDatabaseWrite
		}
	}

	return scanId, nil
}

func (s *local) LoadScan(scanId string) ([]omr.Mat, error) {
	records, err := s.DB.GetPagesForScan(context.Background(), scanId)
	if err != nil {
		return nil, ErrDatabaseRead
	}
	if len(records) == 0 {
		return nil, ErrNotFound
	}

	mats := make([]omr.Mat, len(records))
	for i, record := range records {
		mat, err := s.Mats.Get(record.MatrixKey)
		if err != nil {
			// The scan is corrupted.
			s.DeleteScan(scanId)
			return nil, ErrNotFound
		}
		mats[i] = mat
	}

	return mats, nil
}

func (s *local) DeleteScan(id string) {
	pages, _ := s.DB.GetPagesForScan(context.Background(), id)
	for _, page := range pages {
		s.Images.Delete(page.PictureKey)
		s.Mats.Delete(page.MatrixKey)
	}
	s.DB.DeletePagesForScan(context.Background(), id)
	s.DB.DeleteScan(context.Background(), id)
}

func (s *local) DeleteAllScansFromBefore(t time.Time) {
	rows, _ := s.DB.GetScansFromBefore(context.Background(), t.UnixMilli())
	for _, row := range rows {
		s.DeleteScan(row.ID)
	}
}

func (s *local) LoadScanImage(
	scanId string,
	pageIdx uint,
) (image.Image, error) {

	page, err := s.DB.GetScanPage(
		context.Background(),
		database.GetScanPageParams{
			ScanID:    scanId,
			PageIndex: int64(pageIdx),
		},
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, ErrDatabaseRead
	}

	img, err := s.Images.Get(page.PictureKey)
	if err != nil {
		return nil, ErrFileStorageRead
	}

	return img, nil
}

func (s *local) OpenScanPicture(
	scanId string,
	pageIdx uint,
) (io.ReadCloser, error) {

	page, err := s.DB.GetScanPage(
		context.Background(),
		database.GetScanPageParams{
			ScanID:    scanId,
			PageIndex: int64(pageIdx),
		},
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, ErrDatabaseRead
	}

	img, err := s.Images.Open(page.PictureKey)
	if err != nil {
		return nil, ErrFileStorageRead
	}

	return img, nil
}

//
// Persistent Storage Metadata
//

func (s *local) CountPictures() (int, uint64) {
	return s.Images.Count()
}

func (s *local) CountMats() (int, uint64) {
	return s.Mats.Count()
}

func (s *local) DBSize() uint64 {
	info, err := os.Stat(s.DBPath)
	if err != nil {
		return 0
	}

	return uint64(info.Size())
}
