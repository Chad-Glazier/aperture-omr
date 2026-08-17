package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/Chad-Glazier/aperture-omr/internal/marker"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"

	"gocv.io/x/gocv"
)

const (
	MaxSizeMarksRequest = 1 << 20 // 1 MB
)

func RequestMarks(s resources.ServerResources) mw.JobHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, j mw.JobResources) {

		//
		// Parse the request.
		//

		defer sys.Tidy()
		defer r.Body.Close()

		body, ok := dto.ParseJsonBody[*dto.MarkingJobRequest](
			w, r, MaxSizeMarksRequest,
		)
		if !ok {
			return
		}

		//
		// Fetch the required resources.
		//

		t, err := s.LoadMarkingTemplate(body.TemplateId)
		if err != nil {
			http.Error(w,
				"template ID not recognized",
				http.StatusNotFound,
			)
			return
		}
		template := dto.AdaptMarkerTemplate(t)

		//
		// We stream the scans and mark them in parallel.
		//

		scansMarked := atomic.Uint32{}
		nextIdx := atomic.Uint32{}
		nScans := len(body.ScanIds)
		j.SetNotes(fmt.Sprintf("marking %d exams", nScans))

		scans := make([]dto.Scan, nScans)
		errs := make([]*dto.MarkingError, nScans)

		wg := sync.WaitGroup{}
		for range max(1, runtime.GOMAXPROCS(0)-1) {
			wg.Go(func() {

				//
				// Each thread runs until the pool of scan IDs is exhausted.
				//

				for {
					idx := nextIdx.Add(1) - 1
					if int(idx) >= nScans {
						break
					}
					scanId := body.ScanIds[idx]

					err := markScan(s, template, scanId, &scans[idx])
					if err != nil {
						errs[idx] = &dto.MarkingError{
							ScanId: scanId, Debug: err.Error(),
						}
					}

					j.SetProgress(float64(scansMarked.Add(1)) / float64(nScans))
				}
			})
		}
		wg.Wait()
		j.SetProgress(1.0)
		j.SetNotes(fmt.Sprintf("marked %d exams", nScans))

		result := dto.NewMarkingResult(
			body.TemplateId,
			len(template.Pages),
			scans,
			errs,
		)

		switch {
		case len(result.Scans) == 0:
			// Every scan in the batch failed to mark.
			w.WriteHeader(http.StatusUnprocessableEntity)
			dto.SendCompressedJson(w, r, result)
		default:
			dto.SendCompressedJson(w, r, result)
		}
	}
}

type scan struct {
	id     string
	pages  []gocv.Mat
	closed bool
}

// Idempotently closes the scan's pages.
func (s *scan) close() {
	if s.closed {
		return
	}

	for i := range s.pages {
		s.pages[i].Close()
	}
	s.closed = true
}

func markScan(
	s resources.ServerResources,
	template *marker.Template,
	scanId string,
	out *dto.Scan,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while marking scan %s: %v", scanId, r)
		}
	}()

	pages, loadErr := s.LoadScan(scanId)
	if loadErr != nil {
		return fmt.Errorf("error loading scan %s: %w", scanId, loadErr)
	}
	sc := scan{id: scanId, pages: pages}
	defer sc.close()

	if len(pages) != len(template.Pages) {
		return fmt.Errorf(
			"page count %d for scan %s does not match page count %d of template",
			len(pages), scanId, len(template.Pages),
		)
	}

	marks, markErr := marker.Evaluate(sc.pages, template)
	if markErr != nil {
		return fmt.Errorf("scan %s failed marking: %w", scanId, markErr)
	}
	sc.close()

	out.ScanId = scanId
	out.Marks = make([]dto.Mark, len(marks.Answers))
	for i, a := range marks.Answers {
		out.Marks[i] = dto.AdaptMark(&a)
	}
	return nil
}
