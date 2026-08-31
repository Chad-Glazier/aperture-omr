package resources

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"image"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/database"
	"github.com/Chad-Glazier/aperture-omr/internal/fstore"
	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/google/uuid"
	"gocv.io/x/gocv"
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
	tmpl *dto.PreprocessTemplate,
) (string, error) {

	//
	// The anchor images are base64-encoded in the template struct. We will
	// decode them and convert them to preprocessed matrices before storing,
	// since that makes future use of them more efficient. This also means that
	// we do not need to keep their base64 versions inside of the JSON body
	// when we store it, so we zero those fields before writing to the
	// database.
	//

	binarizeConf := scanner.Config{
		BlurSize:            tmpl.Config.BlurSize,
		MorphCloseSize:      tmpl.Config.MorphCloseSize,
		MinAnchorConfidence: float32(tmpl.Config.MinAnchorConfidence),
		AdaptiveBlockSize:   tmpl.Config.AdaptiveBlockSize,
		AdaptiveC:           float32(tmpl.Config.AdaptiveC),
	}

	anchors := make([][]gocv.Mat, len(tmpl.Pages))
	for i := range tmpl.Pages {
		anchors[i] = make([]gocv.Mat, len(tmpl.Pages[i].Anchors))
		for j := range len(tmpl.Pages[i].Anchors) {

			anchorBase64 := tmpl.Pages[i].Anchors[j].Image
			tmpl.Pages[i].Anchors[j].Image = ""

			buf, err := base64.StdEncoding.DecodeString(anchorBase64)
			if err != nil {
				return "", ErrDecodingImage
			}

			mat, err := gocv.IMDecode(buf, gocv.IMReadGrayScale)
			if err != nil {
				return "", ErrDecodingImage
			}
			defer mat.Close()

			err = scanner.Binarize(&mat, &mat, binarizeConf)
			if err != nil {
				return "", ErrDecodingImage
			}

			anchors[i][j] = mat
		}
	}

	//
	// Now that we've prepared the matrices and the template, we can begin
	// storing them.
	//

	templateId := uuid.New().String()
	jsonStr, err := json.Marshal(tmpl)
	if err != nil {
		return "", ErrDatabaseWrite
	}
	err = s.DB.CreatePreprocessingTemplate(
		context.Background(),
		database.CreatePreprocessingTemplateParams{
			ID:   templateId,
			Json: string(jsonStr),
		},
	)
	if err != nil {
		return "", ErrDatabaseWrite
	}

	//
	// Below, we implement a function that cleans up all resources that have
	// been saved thus far in the operation. That way if there is a failure
	// partway through we won't leave behind any mess in the persistent
	// storage.
	//

	savedAnchors := make([]string, 0)
	cleanupFailure := func() {
		for _, key := range savedAnchors {
			s.Mats.Delete(key)
		}
		s.DB.DeleteAnchorsForTemplate(context.Background(), templateId)
		s.DB.DeletePreprocessingTemplate(context.Background(), templateId)
	}

	//
	// Now, we save the anchors.
	//

	for i := range anchors {
		for j, anchor := range anchors[i] {

			id := uuid.New().String() + ".m4t"
			if err := s.Mats.Set(id, anchor); err != nil {
				cleanupFailure()
				return "", ErrFileStorageWrite
			}
			savedAnchors = append(savedAnchors, id)

			err := s.DB.CreateAnchor(
				context.Background(),
				database.CreateAnchorParams{
					ID:          id,
					TemplateID:  templateId,
					PageIndex:   int64(i),
					AnchorIndex: int64(j),
				},
			)
			if err != nil {
				cleanupFailure()
				return "", ErrDatabaseWrite
			}
		}
	}

	return templateId, nil
}

func (s *local) LoadPreprocessingTemplate(
	id string,
) (*dto.PreprocessTemplate, [][]gocv.Mat, error) {

	record, err := s.DB.GetPreprocessingTemplate(context.Background(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrNotFound
		}
		return nil, nil, ErrDatabaseRead
	}

	tmpl := &dto.PreprocessTemplate{}
	if err := json.Unmarshal([]byte(record.Json), tmpl); err != nil {
		return nil, nil, ErrDeserializing
	}

	expectedAnchors := 0
	for i := range tmpl.Pages {
		expectedAnchors += len(tmpl.Pages[i].Anchors)
	}

	anchorRows, err := s.DB.GetAnchorsForTemplate(
		context.Background(),
		id,
	)
	if err != nil {
		return nil, nil, ErrDatabaseRead
	}
	if len(anchorRows) != expectedAnchors {
		// The preprocessing template is corrupted. We'll delete it and then
		// return [ErrNotFound].
		s.DeletePreprocessingTemplate(id)
		return nil, nil, ErrNotFound
	}

	//
	// The query will sort the anchors by ascending page index and then
	// anchor index. That's the reason the following loop works as intended.
	//

	mats := make([][]gocv.Mat, 0)
	for _, record := range anchorRows {
		if record.AnchorIndex == 0 {
			mats = append(mats, make([]gocv.Mat, 0))
		}

		anchor, err := s.Mats.Get(record.ID)
		if err != nil {
			// The template is corrupted. We'll delete it and then return
			// [ErrNotFound].
			s.DeletePreprocessingTemplate(id)
			return nil, nil, ErrNotFound
		}

		mats[record.PageIndex] = append(mats[record.PageIndex], anchor)
	}

	return tmpl, mats, nil
}

func (s *local) DeletePreprocessingTemplate(id string) {
	s.DB.DeletePreprocessingTemplate(context.Background(), id)
	anchors, _ := s.DB.GetAnchorsForTemplate(context.Background(), id)
	for _, anchor := range anchors {
		s.Mats.Delete(anchor.ID)
	}
	s.DB.DeleteAnchorsForTemplate(context.Background(), id)
}

//
// Scans
//
// Scans used for processing are stored as matrices. Each scan also needs a
// matching human-viewable picture for producing snippets.
//

func (s *local) SaveScan(
	pages []gocv.Mat,
	pagePictures []gocv.Mat,
	templateId string,
) (string, error) {
	if len(pages) != len(pagePictures) {
		panic("resources: SaveScan called with len(pages) != len(pagePictures)")
	}

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
	// Below, we implement a function that cleans up all resources that have
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
	// Now, we iterate over the pages in the scan and store the resources.
	//

	for i := range pages {

		pageId := uuid.New().String() + ".m4t"
		if err := s.Mats.Set(pageId, pages[i]); err != nil {
			cleanupFailure()
			return "", ErrFileStorageWrite
		}
		savedMats = append(savedMats, pageId)

		pictureId := uuid.New().String() + fstore.ImgFileExt
		pictureBuf, err := gocv.IMEncode(fstore.OpenCVImgExt, pagePictures[i])
		if err != nil {
			cleanupFailure()
			return "", ErrEncodingImage
		}
		defer pictureBuf.Close()

		err = s.Images.SetBytes(pictureId, pictureBuf.GetBytes())
		if err != nil {
			cleanupFailure()
			return "", ErrFileStorageWrite
		}
		savedImages = append(savedImages, pictureId)

		err = s.DB.CreateScanPage(
			context.Background(),
			database.CreateScanPageParams{
				ID:         pageId,
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

func (s *local) LoadScan(scanId string) ([]gocv.Mat, error) {
	records, err := s.DB.GetPagesForScan(context.Background(), scanId)
	if err != nil {
		return nil, ErrDatabaseRead
	}
	if len(records) == 0 {
		return nil, ErrNotFound
	}

	mats := make([]gocv.Mat, len(records))
	for i, record := range records {
		mat, err := s.Mats.Get(record.ID)
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
		s.Mats.Delete(page.ID)
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

func (s *local) LoadScanPicture(
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
