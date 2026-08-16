package handler

import (
	"image"
	"image/draw"
	"io"
	"math"
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/fs"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/res"
)

//
// Send an scan's page image.
//

func GetImage(s res.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.GetImageQuery](w, r)
		if !ok {
			return
		}

		img, err := s.OpenScanPicture(q.Scan, q.Page)
		if err != nil {
			http.Error(w,
				"scan page not found",
				http.StatusNotFound,
			)
			return
		}
		defer img.Close()

		w.Header().Add("Content-Type", fs.ImgContentType)
		if _, err := io.Copy(w, img); err != nil {
			http.Error(w,
				"error writing image to response",
				http.StatusInternalServerError,
			)
			return
		}
	}
}

//
// Send a scan's question snippet.
//

func GetSnippet(s res.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.GetSnippetQuery](w, r)
		if !ok {
			return
		}

		//
		// Get the template and find the identified question.
		//

		tmpl, err := s.LoadMarkingTemplate(q.Template)
		if err != nil {
			http.Error(w,
				"marking template not found",
				http.StatusNotFound,
			)
			return
		}

		var (
			targetPageIdx  uint
			targetQuestion *dto.Question
		)
		for pageIdx := range tmpl.Pages {
			for _, question := range tmpl.Pages[pageIdx].Questions {
				if question.ID == q.Question {
					targetPageIdx = uint(pageIdx)
					targetQuestion = &question
					break
				}
			}
		}
		if targetQuestion == nil {
			http.Error(w,
				"question not found",
				http.StatusNotFound,
			)
			return
		}

		//
		// Load the scan image.
		//

		scan, err := s.LoadScanPicture(q.Scan, targetPageIdx)
		if err != nil {
			http.Error(w,
				"error retrieving scan image",
				http.StatusNotFound,
			)
			return
		}

		//
		// Determine the question's bounds in terms of pixels, then build the
		// snippet.
		//

		var (
			minX = math.MaxInt
			minY = math.MaxInt
			maxX = 0
			maxY = 0
		)
		for _, option := range targetQuestion.Options {

			// Note: the X,Y coordinates of an option define the center of it's
			// bubble. In order to get its bounds, we need to add/subtract half
			// of the bubble's respective dimension size.
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

		rect := image.Rect(0, 0, maxX-minX, maxY-minY)
		snippet := image.NewRGBA(rect)
		draw.Draw(
			snippet,
			rect,
			scan,
			image.Point{X: minX, Y: minY},
			draw.Src,
		)

		//
		// Send the snippet.
		//

		w.Header().Add("Content-Type", fs.ImgContentType)
		if err := fs.EncodeImg(w, snippet); err != nil {
			http.Error(w,
				"error writing image to response",
				http.StatusInternalServerError,
			)
			return
		}
	}
}
