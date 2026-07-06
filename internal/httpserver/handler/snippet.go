package handler

// import (
// 	"image"
// 	"net/http"
// )

// func GetSnippet(s ServerResources) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {

// 		//
// 		// Parse the request information.
// 		//

// 		templateId := r.URL.Query().Get("template")
// 		if templateId == "" {
// 			http.Error(
// 				w,
// 				"template query parameter is missing",
// 				http.StatusBadRequest,
// 			)
// 			return
// 		}
// 		scanId := r.URL.Query().Get("scan")
// 		if scanId == "" {
// 			http.Error(
// 				w,
// 				"scan query parameter is missing",
// 				http.StatusBadRequest,
// 			)
// 			return
// 		}
// 		questionId := r.URL.Query().Get("question")
// 		if questionId == "" {
// 			http.Error(
// 				w,
// 				"question query parameter is missing",
// 				http.StatusBadRequest,
// 			)
// 			return
// 		}

// 		//
// 		// Get the database records.
// 		//

// 		record, err := s.LoadMarkingTemplate(templateId)
// 		if err != nil {
// 			return nil, err
// 		}
// 		tmpl, err := dto.ParseMarkingTemplate([]byte(record.Json))
// 		if err != nil {
// 			return nil, fmt.Errorf("error parsing template from database")
// 		}

// 		scanRecords, err := s.DB.GetScanPages(context.Background(), scanId)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if len(scanRecords) != len(tmpl.Pages) {
// 			return nil, fmt.Errorf(
// 				"mismatch between template page count and scan page count",
// 			)
// 		}

// 		//
// 		// Find the question in the template.
// 		//

// 		var targetPageIdx int
// 		var targetQuestion *dto.Question
// 		for pageIdx := range tmpl.Pages {
// 			for _, question := range tmpl.Pages[pageIdx].Questions {
// 				if question.ID == questionId {
// 					targetPageIdx = pageIdx
// 					targetQuestion = &question
// 					break
// 				}
// 			}
// 		}
// 		if targetQuestion == nil {
// 			return nil, fmt.Errorf("question %s not found", questionId)
// 		}

// 		//
// 		// Determine the question's bounds in terms of pixels.
// 		//

// 		var (
// 			minX = math.MaxInt
// 			minY = math.MaxInt
// 			maxX = 0
// 			maxY = 0
// 		)
// 		for _, option := range targetQuestion.Options {

// 			// Note: the X,Y coordinates of an option define the center of it.
// 			// In order to get its bounds, we need to add/subtract half of the
// 			// bubble's respective dimension size.

// 			minX = min(minX, option.X-targetQuestion.BubbleWidth/2)
// 			minY = min(minY, option.Y-targetQuestion.BubbleHeight/2)
// 			maxX = max(maxX, option.X+targetQuestion.BubbleWidth/2)
// 			maxY = max(maxY, option.Y+targetQuestion.BubbleHeight/2)

// 		}

// 		const padding = 10
// 		minX -= padding
// 		maxX += padding
// 		minY -= padding
// 		maxY += padding

// 		//
// 		// Load the image for the page and build the snippet.
// 		//

// 		return s.Store.ImgSnippet(
// 			// Scan records are already ordered by page index.
// 			scanRecords[targetPageIdx].ColorImageKey,
// 			minX, minY,
// 			maxX-minX, maxY-minY,
// 		)

// 		//
// 		// Get the snippet.
// 		//

// 		img, err := s.LoadSnippet(scanId, templateId, questionId)
// 		if err != nil {
// 			http.Error(
// 				w,
// 				"error retrieving image: "+err.Error(),
// 				http.StatusNotFound,
// 			)
// 			return
// 		}

// 		//
// 		// Send the snippet.
// 		//

// 		w.Header().Add("Content-Type", imgType)
// 		if err := encodeImg(w, img); err != nil {
// 			http.Error(
// 				w,
// 				"error writing image to response: "+err.Error(),
// 				http.StatusInternalServerError,
// 			)
// 			return
// 		}
// 	}
// }

// func snip(img image.Image, x0, y0, x1, y1 int) {

// 	// from the resource function




// 	// from the test function

// 	snippetSize := 400
// 	snippet, err := store.ImgSnippet(name, 20, 20, snippetSize, snippetSize)
// 	if err != nil {
// 		t.Error("error creating snippet: " + err.Error())
// 	}

// 	bounds := snippet.Bounds()
// 	if bounds.Dx() != snippetSize {
// 		t.Errorf("expected snippet width of %d, got %d", snippetSize, bounds.Dx())
// 	}
// 	if bounds.Dy() != snippetSize {
// 		t.Errorf("expected snippet height of %d, got %d", snippetSize, bounds.Dy())
// 	}

// 	// From the other function

// 	rect := image.Rect(0, 0, width, height)

// 	cropped := image.NewRGBA(rect)

// 	draw.Draw(
// 		cropped,
// 		rect,
// 		img,
// 		image.Point{X: x, Y: y},
// 		draw.Src,
// 	)
// }
