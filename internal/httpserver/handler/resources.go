package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"io"

	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/database/sqlc"
	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/httpserver/dto"

	"github.com/google/uuid"
	"github.com/pierrec/lz4/v4"
	"gocv.io/x/gocv"
)

// A interface that presents all resources that an HTTP handler should have
// access to.
type ServerResources interface {
	// Closes all connections.
	Close() error

	// Saves a marking template and returns the new ID for it.
	SaveMarkingTemplate(tmpl *dto.MarkingTemplate) (string, error)

	// Loads a marking template and returns the new ID for it.
	LoadMarkingTemplate(id string) (*dto.MarkingTemplate, error)

	// Saves a preprocessing template and returns the new ID for it.
	SavePreprocessingTemplate(tmpl *dto.PreprocessingTemplate) (string, error)

	// Loads a preprocessing template and returns the ID for it.
	LoadPreprocessingTemplate(id string) (*dto.PreprocessingTemplate, error)

	// Loads anchor matrices. To get the i-th page's j-th index, you would
	// index [i][j] from the returned slice.
	LoadAnchors(templateId string) ([][]*gocv.Mat, error)

	// Saves the given anchor matrix.
	SaveAnchor(
		anchors *gocv.Mat,
		templateId string,
		pageIdx, anchorIdx int,
	) error

	// Saves a preprocessed scan via a slice of images where each image
	// represents a page. The preprocessing template ID is also included for
	// the sake of debugging. Returns an ID for the scan.
	SaveScan(
		pages []*gocv.Mat,
		colorPages []*gocv.Mat,
		templateId string,
	) (string, error)

	// Loads a preprocessed scan's pages.
	LoadScan(scanId string) ([]*gocv.Mat, error)

	// Loads a page from a scan.
	LoadColorScan(scanId string, pageIdx int) (image.Image, error)
}

//
// Below, we implement the interface.
//

type defaultResources struct {
	DBCnx  io.Closer
	DB     database.Querier
	Images fs.ImageStore
	Mats   fs.MatStore
}

var _ ServerResources = (*defaultResources)(nil)

// An implementation of ServerResources that uses a default SQLite database and
// stores files locally. All data (i.e., the SQLite file and the root directory
// for stored files) will be stored in the specified root.
func NewDefaultResources(rootDir string) (*defaultResources, error) {

	db, cnx, err := database.Connect(rootDir + "/database.sqlite3")
	if err != nil {
		return nil, err
	}

	images, err := fs.NewLocalImageStore(rootDir + "/images")
	if err != nil {
		return nil, err
	}

	mats, err := fs.NewLocalMatStore(rootDir + "/matrices")
	if err != nil {
		return nil, err
	}

	res := &defaultResources{
		DBCnx:  cnx,
		DB:     db,
		Images: images,
		Mats:   mats,
	}
	return res, nil
}

func (s *defaultResources) Close() error {
	return s.DBCnx.Close()
}

//
// Marking Templates
//
// Marking templates are often very large (~150KB), primarily due to their JSON
// format. We can save ~90% of this storage with minimal performance overhead
// by using LZ4 compression.
//

func (s *defaultResources) SaveMarkingTemplate(
	tmpl *dto.MarkingTemplate,
) (string, error) {

	buf := bytes.Buffer{}
	lz4Encoder := lz4.NewWriter(&buf)
	jsonEncoder := json.NewEncoder(lz4Encoder)

	if err := jsonEncoder.Encode(tmpl); err != nil {
		lz4Encoder.Close()
		return "", err
	}

	if err := lz4Encoder.Close(); err != nil {
		return "", err
	}

	id := uuid.New()

	err := s.DB.CreateMarkingTemplate(
		context.Background(),
		sqlc.CreateMarkingTemplateParams{
			ID:   id.String(),
			Json: buf.Bytes(),
		},
	)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (s *defaultResources) LoadMarkingTemplate(
	id string,
) (*dto.MarkingTemplate, error) {

	record, err := s.DB.GetMarkingTemplate(context.Background(), id)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(record.Json)
	lz4Decoder := lz4.NewReader(buf)
	jsonDecoder := json.NewDecoder(lz4Decoder)

	tmpl := &dto.MarkingTemplate{}
	if err := jsonDecoder.Decode(tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

//
// Preprocessing Templates
//
// Unlike marking templates, preprocessing templates are a lot more modest in
// size. We just store the uncompressed JSON.
//

func (s *defaultResources) SavePreprocessingTemplate(
	tmpl *dto.PreprocessingTemplate,
) (string, error) {

	id := uuid.New()
	jsonStr, err := json.Marshal(tmpl)
	if err != nil {
		return "", err
	}

	err = s.DB.CreatePreprocessingTemplate(
		context.Background(),
		sqlc.CreatePreprocessingTemplateParams{
			ID:   id.String(),
			Json: string(jsonStr),
		},
	)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (s *defaultResources) LoadPreprocessingTemplate(
	id string,
) (*dto.PreprocessingTemplate, error) {

	record, err := s.DB.GetPreprocessingTemplate(context.Background(), id)
	if err != nil {
		return nil, err
	}

	tmpl := &dto.PreprocessingTemplate{}
	if err := json.Unmarshal([]byte(record.Json), tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

//
// Anchors
//
// Anchors are stored as matrices. These methods just straightforwardly call
// the MatStore methods.
//

func (s *defaultResources) SaveAnchor(
	mat *gocv.Mat,
	templateId string,
	pageIdx, anchorIdx int,
) error {

	id := uuid.New().String() + ".m4t"
	if err := s.Mats.Set(id, mat); err != nil {
		return err
	}

	err := s.DB.CreateAnchor(
		context.Background(),
		sqlc.CreateAnchorParams{
			ID:          id,
			TemplateID:  templateId,
			PageIndex:   int64(pageIdx),
			AnchorIndex: int64(anchorIdx),
		},
	)
	if err != nil {
		s.Mats.Delete(id)
		return err
	}

	return nil
}

func (s *defaultResources) LoadAnchors(
	templateId string,
) ([][]*gocv.Mat, error) {

	anchorRecords, err := s.DB.GetAnchorsForTemplate(
		context.Background(),
		templateId,
	)
	if err != nil {
		return nil, err
	}

	mats := make([][]*gocv.Mat, 0, 2)
	for _, record := range anchorRecords {

		// The query will sort the anchors by ascending page index and then
		// anchor index. That's the reason this loop works as intended.

		anchor, err := s.Mats.Get(record.ID)
		if err != nil {
			return nil, err
		}

		if record.AnchorIndex == 0 {
			mats = append(mats, make([]*gocv.Mat, 0, 4))
		}
		mats[record.PageIndex] = append(mats[record.PageIndex], anchor)

	}

	return mats, nil
}

//
// Scans
//
// Scans used for processing are stored as matrices. Each scan also needs a
// matching colored image for producing snippets.
//

func (s *defaultResources) SaveScan(
	pages []*gocv.Mat,
	colorPages []*gocv.Mat,
	templateId string,
) (string, error) {

	scanId := uuid.New().String()
	err := s.DB.CreateScan(context.Background(), sqlc.CreateScanParams{
		ID:                      scanId,
		PreprocessingTemplateID: templateId,
	})
	if err != nil {
		return "", err
	}

	for i := range pages {

		pageId := uuid.New().String() + ".m4t"
		if err := s.Mats.Set(pageId, pages[i]); err != nil {
			return "", err
		}

		colorPageId := uuid.New().String() + fs.ImgFileExt
		colorPageBuf, err := gocv.IMEncode(fs.OpenCVImgExt, *colorPages[i])
		if err != nil {
			return "", err
		}

		err = s.Images.SetBytes(colorPageId, colorPageBuf.GetBytes())
		if err != nil {
			return "", err
		}

		err = s.DB.CreateScanPage(
			context.Background(),
			sqlc.CreateScanPageParams{
				ID:            pageId,
				ColorImageKey: colorPageId,
				PageIndex:     int64(i),
				ScanID:        scanId,
			},
		)
		if err != nil {
			return "", err
		}
	}

	return scanId, nil
}

func (s *defaultResources) LoadScan(scanId string) ([]*gocv.Mat, error) {
	records, err := s.DB.GetScanPages(context.Background(), scanId)
	if err != nil {
		return nil, err
	}

	mats := make([]*gocv.Mat, len(records))
	for i, record := range records {
		mat, err := s.Mats.Get(record.ID)
		if err != nil {
			return nil, err
		}
		mats[i] = mat
	}

	return mats, nil
}

func (s *defaultResources) LoadColorScan(
	scanId string,
	pageIdx int,
) (image.Image, error) {

	page, err := s.DB.GetScanPage(
		context.Background(),
		sqlc.GetScanPageParams{
			ScanID:    scanId,
			PageIndex: int64(pageIdx),
		},
	)
	if err != nil {
		return nil, err
	}

	img, err := s.Images.Get(page.ColorImageKey)
	if err != nil {
		return nil, err
	}

	return img, nil
}
