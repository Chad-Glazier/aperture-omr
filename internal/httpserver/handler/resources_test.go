package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"math"
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
	DB    database.Querier
	Store fs.Store
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
		DB:    db,
		Store: store,
		root:  root,
	}
	return res, nil
}

func (s *res) Cleanup() {
	os.RemoveAll(s.root)
}

func (s res) SaveMarkingTemplate(
	tmpl *dto.MarkingTemplate,
) (string, error) {
	id := uuid.New()
	jsonStr, err := json.Marshal(tmpl)
	if err != nil {
		return "", err
	}

	err = s.DB.CreateMarkingTemplate(
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

func (s res) LoadMarkingTemplate(
	id string,
) (*dto.MarkingTemplate, error) {

	record, err := s.DB.GetMarkingTemplate(context.Background(), id)
	if err != nil {
		return nil, err
	}

	tmpl := &dto.MarkingTemplate{}
	if err := json.Unmarshal([]byte(record.Json), tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (s res) SavePreprocessingTemplate(
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

func (s res) LoadPreprocessingTemplate(
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

func (s res) SaveAnchor(
	img image.Image, templateId string, pageIdx, anchorIdx int,
) error {

	id := uuid.New().String() + fs.ImgFileExt
	if err := s.Store.PutImg(id, img); err != nil {
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
		s.Store.DeleteImg(id)
		return err
	}

	return nil
}

func (s res) LoadAnchor(
	templateId string, pageIdx, anchorIdx int,
) (image.Image, error) {

	anchor, err := s.DB.GetOneAnchorForTemplate(
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

	img, err := s.Store.GetImg(anchor.ID)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (s res) SaveScan(
	pages []image.Image,
	colorPages []image.Image,
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

		pageId := uuid.New().String() + fs.ImgFileExt
		if err := s.Store.PutImg(pageId, pages[i]); err != nil {
			return "", err
		}

		colorPageId := uuid.New().String() + fs.ImgFileExt
		if err := s.Store.PutImg(colorPageId, colorPages[i]); err != nil {
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

func (s res) LoadScan(scanId string) ([]image.Image, error) {
	records, err := s.DB.GetScanPages(context.Background(), scanId)
	if err != nil {
		return nil, err
	}

	images := make([]image.Image, len(records))
	for i, record := range records {
		img, err := s.Store.GetImg(record.ID)
		if err != nil {
			return nil, err
		}
		images[i] = img
	}

	return images, nil
}

func (s res) LoadColorScan(scanId string) ([]image.Image, error) {
	records, err := s.DB.GetScanPages(context.Background(), scanId)
	if err != nil {
		return nil, err
	}

	images := make([]image.Image, len(records))
	for i, record := range records {
		img, err := s.Store.GetImg(record.ColorImageKey)
		if err != nil {
			return nil, err
		}
		images[i] = img
	}

	return images, nil
}

func (s res) LoadSnippet(
	scanId,
	templateId,
	questionId string,
) (image.Image, error) {

	//
	// Get the database records.
	//

	record, err := s.DB.GetMarkingTemplate(context.Background(), templateId)
	if err != nil {
		return nil, err
	}
	tmpl, err := dto.ParseMarkingTemplate([]byte(record.Json))
	if err != nil {
		return nil, fmt.Errorf("error parsing template from database")
	}

	scanRecords, err := s.DB.GetScanPages(context.Background(), scanId)
	if err != nil {
		return nil, err
	}
	if len(scanRecords) < len(tmpl.Pages) {
		return nil, fmt.Errorf("scan page(s) missing from database")
	}

	//
	// Find the question in the template.
	//

	var targetPageIdx int
	var targetQuestion *dto.Question
	for pageIdx := range tmpl.Pages {
		for _, question := range tmpl.Pages[pageIdx].Questions {
			if question.ID == questionId {
				targetPageIdx = pageIdx
				targetQuestion = &question
				break
			}
		}
	}
	if targetQuestion == nil {
		return nil, fmt.Errorf("question %s not found", questionId)
	}

	//
	// Determine the question's bounds in terms of pixels.
	//

	var (
		minX = math.MaxInt
		minY = math.MaxInt
		maxX = 0
		maxY = 0
	)
	for _, option := range targetQuestion.Options {

		// Note: the X,Y coordinates of an option define the center of it.
		// In order to get its bounds, we need to add/subtract half of the
		// bubble's respective dimension size.

		minX = min(minX, option.X-targetQuestion.BubbleWidth/2)
		minY = min(minY, option.Y-targetQuestion.BubbleHeight/2)
		maxX = max(maxX, option.X+targetQuestion.BubbleWidth/2)
		maxY = max(maxY, option.Y+targetQuestion.BubbleHeight/2)

	}

	const padding = 10
	minX = max(0, minX-padding)
	maxX += padding
	minY = max(0, minY-padding)
	maxY += padding

	//
	// Load the image for the page and build the snippet.
	//

	return s.Store.ImgSnippet(
		// Scan records are already ordered by page index.
		scanRecords[targetPageIdx].ID,
		minX, minY,
		maxY-minY, maxX-minX,
	)

}
