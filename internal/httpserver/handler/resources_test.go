package handler

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"testing"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/database/sqlc"
	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/httpserver/dto"

	"github.com/google/uuid"
)

//
// In this file, we implement a mock version of server resources. To use this
// in a test you can allocate the resources with NewServerResources, then
// defer the Cleanup method to remove any temporary files.
//

// An implementation of server resources for testing purposes. Use the Cleanup
// method to remove any temporary files created.
//
// This implementation is fully functional, meaning that you can query its
// database and image stores to verify that data was correctly stored.
type res struct {
	db    database.Querier
	store fs.Store
	root  string
}

var _ ServerResources = (*res)(nil)

func NewServerResources(t *testing.T) (*res, error) {
	t.Helper()

	db, err := database.Connect(":memory:")
	if err != nil {
		return nil, err
	}

	root, err := os.MkdirTemp("", "omr_test_fs_*")
	if err != nil {
		return nil, err
	}
	store := fs.NewLocalStore(root)

	res := &res{
		db:    db,
		store: store,
		root:  root,
	}
	return res, nil
}

func (s *res) Cleanup() {
	os.RemoveAll(s.root)
}

func (s *res) SaveMarkingTemplate(
	tmpl *dto.MarkingTemplate,
) (string, error) {
	id := uuid.New()
	jsonStr, err := json.Marshal(tmpl)
	if err != nil {
		return "", err
	}

	err = s.db.CreateMarkingTemplate(
		context.Background(),
		sqlc.CreateMarkingTemplateParams{
			ID:   id.String(),
			Json: string(jsonStr),
		},
	)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (s *res) LoadMarkingTemplate(
	id string,
) (*dto.MarkingTemplate, error) {

	record, err := s.db.GetMarkingTemplate(context.Background(), id)
	if err != nil {
		return nil, err
	}

	tmpl := &dto.MarkingTemplate{}
	if err := json.Unmarshal([]byte(record.Json), tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *res) SavePreprocessingTemplate(
	tmpl *dto.PreprocessingTemplate,
) (string, error) {
	id := uuid.New()
	jsonStr, err := json.Marshal(tmpl)
	if err != nil {
		return "", err
	}

	err = s.db.CreatePreprocessingTemplate(
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

func (s *res) LoadPreprocessingTemplate(
	id string,
) (*dto.PreprocessingTemplate, error) {

	record, err := s.db.GetPreprocessingTemplate(context.Background(), id)
	if err != nil {
		return nil, err
	}

	tmpl := &dto.PreprocessingTemplate{}
	if err := json.Unmarshal([]byte(record.Json), tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s *res) SaveAnchor(
	img image.Image, templateId string, pageIdx, anchorIdx int,
) error {

	id := uuid.New().String() + fs.ImgFileExt
	if err := s.store.PutImg(id, img); err != nil {
		return err
	}

	err := s.db.CreateAnchor(
		context.Background(),
		sqlc.CreateAnchorParams{
			ID:          id,
			TemplateID:  templateId,
			PageIndex:   int64(pageIdx),
			AnchorIndex: int64(anchorIdx),
		},
	)
	if err != nil {
		s.store.DeleteImg(id)
		return err
	}

	return nil
}

func (s *res) LoadAnchor(
	templateId string, pageIdx, anchorIdx int,
) (image.Image, error) {

	anchor, err := s.db.GetOneAnchorForTemplate(
		context.Background(),
		sqlc.GetOneAnchorForTemplateParams{
			TemplateID:  templateId,
			PageIndex:   int64(pageIdx),
			AnchorIndex: int64(anchorIdx),
		},
	)
	if err != nil {
		return nil, err
	}

	img, err := s.store.GetImg(anchor.ID)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (s *res) SaveScan(
	pages []image.Image, templateId string,
) (string, error) {

	scanId := uuid.New().String()
	err := s.db.CreateScan(context.Background(), sqlc.CreateScanParams{
		ID:                      scanId,
		PreprocessingTemplateID: templateId,
	})
	if err != nil {
		return "", err
	}

	for pageIdx, page := range pages {

		pageId := uuid.New().String() + ".png"

		err := s.store.PutImg(pageId, page)
		if err != nil {
			return "", err
		}

		err = s.db.CreateScanPage(
			context.Background(),
			sqlc.CreateScanPageParams{
				ID:        pageId,
				PageIndex: int64(pageIdx),
				ScanID:    scanId,
			},
		)
		if err != nil {
			return "", err
		}
	}

	return scanId, nil
}

func (s *res) LoadScan(scanId string) ([]image.Image, error) {
	records, err := s.db.GetScanPages(context.Background(), scanId)
	if err != nil {
		return nil, err
	}

	images := make([]image.Image, len(records))
	for i, record := range records {
		img, err := s.store.GetImg(record.ID)
		if err != nil {
			return nil, err
		}
		images[i] = img
	}

	return images, nil
}
