package handler

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"sync"
	"time"

	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/marker"

	"gocv.io/x/gocv"
)

func PostMarkingJob(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Parse the request.
		//

		defer r.Body.Close()
		jsonBuf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(
				w,
				"error reading body: "+err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		markingJob, err := dto.ParseMarkingJobRequest(jsonBuf)
		if err != nil {
			http.Error(
				w,
				"error parsing body: "+err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		//
		// Fetch the required resources.
		//

		tmpl, err := s.LoadMarkingTemplate(markingJob.TemplateId)
		if err != nil {
			http.Error(
				w,
				"error retrieving template: " + err.Error(),
				http.StatusNotFound,
			)
			return
		}

		scans := make([]scan, len(markingJob.ScanIds))
		for i, scanId := range markingJob.ScanIds {
			pages, err := s.LoadScan(scanId)
			if err != nil {
				http.Error(
					w,
					"error loading scan " + scanId + ": " + err.Error(),
					http.StatusNotFound,
				)
				return
			}
			if len(pages) != len(tmpl.Pages) {
				http.Error(
					w,
					fmt.Sprintf(
						"page count %d for scan %s does not match "+ 
						"page count %d of template %s",
						len(pages), scanId,
						len(tmpl.Pages), markingJob.TemplateId,
					),
					http.StatusBadRequest,
				)
			}
			scans[i].id = scanId
			scans[i].pages = pages
		}

		totalQuestions := 0
		for _, page := range tmpl.Pages {
			totalQuestions += len(page.Questions)
		}

		//
		// Run the job.
		//

		results := dto.MarkingResult{}
		results.PagesMarked = len(tmpl.Pages) * len(scans)
		results.TemplateId = markingJob.TemplateId
		results.StartTime = int(time.Now().Unix())
		results.Scans = make([]dto.Scan, len(markingJob.ScanIds))
		for i := range results.Scans {
			results.Scans[i].Marks = make([]dto.Mark, totalQuestions)
		}

		wg := sync.WaitGroup{}		
		for i, scan := range scans {
			wg.Go(func() {
				marks, err := markScan(tmpl, &scan)
				if err != nil {
					results.Errors = append(
						results.Errors, 
						"error in scan " + scan.id + ": " + err.Error(),
					)
					return
				}
				for j, mark := range marks {
					results.Scans[i].Marks[j].Flagged = mark.flagged
					results.Scans[i].Marks[j].Selected = mark.selected
					results.Scans[i].Marks[j].QuestionId = mark.questionId
				} 
				results.Scans[i].ScanId = scan.id
			})
		}
		wg.Wait()

		results.EndTime = int(time.Now().Unix())

		sendJson(w, results)
	}
}

//
// The marking function is defined below.
//

type scan struct {
	id string
	pages []image.Image
}

type marks []struct {
	questionId string
	flagged    bool
	selected   []string
}

func markScan(tmpl *dto.MarkingTemplate, scan *scan) (marks, error) {

	//
	// Translate the template.
	//

	template := marker.Template{
		Config: marker.Config{
			FillThreshold: &tmpl.Config.FillThreshold,
			BubbleInset: &tmpl.Config.BubbleInset,
			FlagThreshold: &tmpl.Config.FlagThreshold,
		},
		Pages: make([]marker.Page, len(tmpl.Pages)),
	}

	for i, p := range tmpl.Pages {
		// yes, I know how it looks
		template.Pages[i].Questions = make([]marker.Question, len(p.Questions))
		for j, q := range p.Questions {
			template.Pages[i].Questions[j].ID = q.ID
			template.Pages[i].Questions[j].Type = q.Type
			template.Pages[i].Questions[j].BubbleWidth = q.BubbleWidth
			template.Pages[i].Questions[j].BubbleHeight = q.BubbleHeight
			template.Pages[i].Questions[j].Options = make([]marker.Bubble, len(q.Options))
			for k, o := range q.Options {
				template.Pages[i].Questions[j].Options[k].Label = o.Label
				template.Pages[i].Questions[j].Options[k].X = o.X
				template.Pages[i].Questions[j].Options[k].Y = o.Y
			}
		}
	}

	// 
	// Translate the images.
	//

	mats := make([]gocv.Mat, len(scan.pages))
	for i, img := range scan.pages {
		buf := bytes.Buffer{}
		if err := fs.EncodeImg(&buf, img); err != nil {
			return nil, err
		}
		mat, err := gocv.IMDecode(buf.Bytes(), gocv.IMReadGrayScale)
		if err != nil {
			return nil, err
		}
		mats[i] = mat
	}

	//
	// Do the marking.
	//

	result, err := marker.Evaluate(mats, &template)
	if err != nil {
		return nil, err
	}

	marks := make(marks, len(result.Answers))
	for i, answer := range result.Answers {
		marks[i].flagged = answer.Flag
		marks[i].questionId = answer.QuestionID
		marks[i].selected = answer.Selected
	}

	return marks, nil
}
