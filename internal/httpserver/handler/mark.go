package handler

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/marker"

	"gocv.io/x/gocv"
)

type scan struct {
	id     string
	pages  []*gocv.Mat
	closed bool
}

// Idempotently closes the scan's pages.
func (s *scan) close() {
	if !s.closed {
		return
	}

	for i := range s.pages {
		s.pages[i].Close()
	}
	s.closed = true
}

func PostMarkingJob(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Parse the request.
		//

		defer debug.FreeOSMemory()
		defer r.Body.Close()

		jsonBuf, err := io.ReadAll(r.Body)
		if err != nil {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"error reading body: "+err.Error(),
			)
			return
		}

		markingJob, err := dto.ParseMarkingJobRequest(jsonBuf)
		if err != nil {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"error parsing body: "+err.Error(),
			)
			return
		}

		//
		// Fetch the required resources.
		//

		t, err := s.LoadMarkingTemplate(markingJob.TemplateId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusNotFound,
				dto.ErrTemplateNotFound,
				"error retrieving template: "+err.Error(),
			)
			return
		}
		template := dto.AdaptMarkerTemplate(t)

		//
		// We stream the scans and mark them in parallel. This sets a limit
		// on the number of scans that are kept in memory at a given time.
		//

		errorSent := atomic.Bool{}
		nextIdx := atomic.Uint32{}
		nScans := len(markingJob.ScanIds)

		markingResults := dto.MarkingResult{
			PagesMarked: nScans * len(template.Pages),
			TemplateId:  markingJob.TemplateId,
		}
		markingResults.Scans = make([]dto.Scan, nScans)

		wg := sync.WaitGroup{}
		for range runtime.GOMAXPROCS(0) {
			wg.Go(func() {

				defer func() {
					if r := recover(); r != nil {
						if !errorSent.Swap(true) {
							dto.SendError(
								w,
								http.StatusInternalServerError,
								dto.ErrInternal,
								"unexpected panic during marking",
							)
						}
					}
				}()

				//
				// Each thread runs until the pool of scan IDs is exhausted,
				// or until an error is sent.
				//

				for !errorSent.Load() {

					idx := nextIdx.Add(1) - 1
					if int(idx) >= nScans {
						break
					}
					scanId := markingJob.ScanIds[idx]

					//
					// Load the preprocessed scan's page matrices.
					//

					pages, err := s.LoadScan(scanId)
					if err != nil && !errorSent.Swap(true) {
						dto.SendError(
							w,
							http.StatusNotFound,
							dto.ErrScanNotFound,
							"error loading scan "+scanId+": "+err.Error(),
						)
						break
					}
					scan := scan{id: scanId, pages: pages}
					defer scan.close()

					if len(pages) != len(template.Pages) &&
						!errorSent.Swap(true) {
						dto.SendError(
							w,
							http.StatusBadRequest,
							dto.ErrPageCountMismatch,
							fmt.Sprintf(
								"page count %d for scan %s does not match "+
									"page count %d of template %s",
								len(pages), scanId,
								len(template.Pages), markingJob.TemplateId,
							),
						)
						break
					}

					//
					// Get the marks.
					//

					marks, err := marker.Evaluate(scan.pages, template)
					if err != nil && !errorSent.Swap(true) {
						dto.SendError(
							w,
							http.StatusUnprocessableEntity,
							dto.ErrMarkingFailed,
							"scan "+scanId+" failed marking",
						)
						break
					}
					scan.close()

					markingResults.Scans[idx].ScanId = scanId
					markingResults.Scans[idx].Marks = make(
						[]dto.Mark,
						len(marks.Answers),
					)
					for i, a := range marks.Answers {
						markingResults.Scans[idx].Marks[i] = *dto.AdaptMark(&a)
					}
				}
			})
		}
		wg.Wait()
		if errorSent.Load() {
			return
		}

		dto.SendCompressedJson(w, r, markingResults)
	}
}
