package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log/slog"
	"math"
	"net/http"
	"os"

	"ubco-team15/omr/config"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/database/sqlc"
	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"

	"github.com/google/uuid"
)

func Start() {

	res, err := NewServerResources()
	if err != nil {
		slog.Error("error getting server resources", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	mux.HandleFunc("POST /template/mark", handler.PostMarkingTemplate(res))
	mux.HandleFunc("POST /template/preprocess", handler.PostPreprocessingTemplate(res))
	mux.HandleFunc("POST /scan", handler.PostScan(res))
	mux.HandleFunc("POST /mark", handler.PostMarkingJob(res))
	mux.HandleFunc("GET /snippet", handler.GetSnippet(res))

	httpHandler := middleware.Cors(mux)
	httpHandler = middleware.Recovery(httpHandler)
	httpHandler = middleware.Logger(httpHandler)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: httpHandler,
	}

	slog.Info("starting server at http://" + config.Host + ":" + config.Port)
	server.ListenAndServe()
}

//
// Below, we implement the ServerResources interface. This is how the database
// and file storage are provided to the handler functions.
//

type ServerResources struct {
	DB    database.Querier
	Store fs.Store
}

var _ handler.ServerResources = (*ServerResources)(nil)

func NewServerResources() (*ServerResources, error) {
	// In the future we can consider changing this to ":memory:" during
	// testing in order to avoid a cleanup step. This only works for SQLite
	// though.
	db, err := database.Connect("data/database.sqlite3")
	if err != nil {
		return nil, err
	}

	store := fs.NewLocalStore("data/images")

	res := &ServerResources{
		DB:    db,
		Store: store,
	}
	return res, nil
}

func (s *ServerResources) SaveMarkingTemplate(
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

func (s *ServerResources) LoadMarkingTemplate(
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

func (s *ServerResources) SavePreprocessingTemplate(
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

func (s *ServerResources) LoadPreprocessingTemplate(
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

func (s *ServerResources) SaveAnchor(
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

func (s *ServerResources) LoadAnchor(
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

func (s *ServerResources) SaveScan(
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

func (s *ServerResources) LoadScan(scanId string) ([]image.Image, error) {
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

func (s *ServerResources) LoadColorScan(scanId string) ([]image.Image, error) {
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

func (s *ServerResources) LoadSnippet(
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
	minX -= padding
	maxX += padding
	minY -= padding
	maxY += padding

	//
	// Load the image for the page and build the snippet.
	//

	return s.Store.ImgSnippet(
		// Scan records are already ordered by page index.
		scanRecords[targetPageIdx].ColorImageKey,
		minX, minY,
		maxX-minX, maxY-minY,
	)

}
