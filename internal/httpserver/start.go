package httpserver

import (
	"context"
	"encoding/json"
	"image"
	"log/slog"
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
	db    database.Querier
	store fs.Store
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
		db:    db,
		store: store,
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

func (s *ServerResources) LoadMarkingTemplate(
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

func (s *ServerResources) SavePreprocessingTemplate(
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

func (s *ServerResources) LoadPreprocessingTemplate(
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

func (s *ServerResources) SaveAnchor(
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

func (s *ServerResources) LoadAnchor(
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

func (s *ServerResources) SaveScan(
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

func (s *ServerResources) LoadScan(scanId string) ([]image.Image, error) {
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
