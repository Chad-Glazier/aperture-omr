package handler

import (
	"image"
	"os"
	"testing"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/httpserver/dto"
)

//
// In this file, we implement a mock version of server resources. To use this
// in a test you can allocate the resources with newTestingResources, then
// defer the Cleanup method to remove any temporary files.
//

// An implementation of server resources for testing purposes. Use the Cleanup
// method to remove any temporary files created.
//
// This implementation is fully functional, meaning that you can query its
// database and image stores to verify that data was correctly stored.
type testingResources struct {
	s    defaultResources
	root string
}

var _ ServerResources = (*testingResources)(nil)

func newTestingResources(t *testing.T) (*testingResources, error) {
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

	res := &testingResources{
		s: defaultResources{
			DB:    db,
			Store: store,
		},
		root: root,
	}
	return res, nil
}

func (s *testingResources) Cleanup() {
	os.RemoveAll(s.root)
}

//
// Delegation calls.
//

func (r *testingResources) SaveMarkingTemplate(
	tmpl *dto.MarkingTemplate,
) (string, error) {
	return r.s.SaveMarkingTemplate(tmpl)
}

func (r *testingResources) LoadMarkingTemplate(
	id string,
) (*dto.MarkingTemplate, error) {
	return r.s.LoadMarkingTemplate(id)
}

func (r *testingResources) SavePreprocessingTemplate(
	tmpl *dto.PreprocessingTemplate,
) (string, error) {
	return r.s.SavePreprocessingTemplate(tmpl)
}

func (r *testingResources) LoadPreprocessingTemplate(
	id string,
) (*dto.PreprocessingTemplate, error) {
	return r.s.LoadPreprocessingTemplate(id)
}

func (r *testingResources) SaveAnchor(
	anchor image.Image,
	templateId string,
	pageIdx,
	anchorIdx int,
) error {
	return r.s.SaveAnchor(anchor, templateId, pageIdx, anchorIdx)
}

func (r *testingResources) LoadAnchor(
	templateId string,
	pageIdx,
	anchorIdx int,
) (image.Image, error) {
	return r.s.LoadAnchor(templateId, pageIdx, anchorIdx)
}

func (r *testingResources) SaveScan(
	pages []image.Image,
	colorPages []image.Image,
	templateId string,
) (string, error) {
	return r.s.SaveScan(pages, colorPages, templateId)
}

func (r *testingResources) LoadScan(scanId string) ([]image.Image, error) {
	return r.s.LoadScan(scanId)
}

func (r *testingResources) LoadColorScan(
	scanId string,
) ([]image.Image, error) {
	return r.s.LoadColorScan(scanId)
}

func (r *testingResources) LoadSnippet(
	scanId,
	templateId,
	questionId string,
) (image.Image, error) {
	return r.s.LoadSnippet(scanId, templateId, questionId)
}
