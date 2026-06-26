//go:build integration

package marker

import (
	"os"
	"path/filepath"
	"testing"

	"gocv.io/x/gocv"
)

// TestMarkIntegration evaluates a real preprocessed exam scan against the
// mark template. Test data must be present in the testdata/ directory:
//
//	testdata/preprocessed.png  — binary image produced by `omr preprocess`
//	testdata/mark.json         — mark template
func TestMarkIntegration(t *testing.T) {
	const testdataDir = "testdata"

	preprocessedPath := filepath.Join(testdataDir, "preprocessed.png")
	if _, err := os.Stat(preprocessedPath); err != nil {
		t.Skip("testdata not present, skipping integration test")
	}

	img := gocv.IMRead(preprocessedPath, gocv.IMReadGrayScale)
	defer img.Close()
	if img.Empty() {
		t.Fatal("could not read preprocessed image")
	}

	markTmplFile, err := os.Open(filepath.Join(testdataDir, "mark.json"))
	if err != nil {
		t.Fatalf("open mark template: %v", err)
	}
	defer markTmplFile.Close()

	markTmpl, err := LoadTemplate(markTmplFile)
	if err != nil {
		t.Fatalf("load mark template: %v", err)
	}

	result, err := Evaluate(img, markTmpl)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if len(result.Answers) != 8 {
		t.Fatalf("expected 8 answers, got %d", len(result.Answers))
	}

	// Index answers by question ID for lookup below.
	byID := make(map[string]Answer, len(result.Answers))
	for _, ans := range result.Answers {
		byID[ans.QuestionID] = ans
	}

	// Verify the sheet is not read as entirely blank.
	answered := 0
	for _, ans := range result.Answers {
		if len(ans.Selected) > 0 {
			answered++
		}
	}
	if answered == 0 {
		t.Error("all questions read as blank — mark template may be misaligned")
	}

	// Spot-check known answers from the test sheet.
	checks := []struct {
		question string
		selected []string
		flagged  bool
	}{
		// Q2 was specified in the JSON as multi-select, so it won't get flagged
		{question: "Q1", selected: []string{"B"}, flagged: false},
		{question: "Q2", selected: []string{"C", "E"}, flagged: false},
		{question: "Q3", selected: []string{"B"}, flagged: false},
		{question: "Q4", selected: []string{"C"}, flagged: false},
		{question: "Q5", selected: []string{"B", "D"}, flagged: true},
		{question: "Q6", selected: []string{}, flagged: true},
		{question: "Q7", selected: []string{}, flagged: true},
		{question: "Q8", selected: []string{"B"}, flagged: false},
	}

	for _, c := range checks {
		ans, ok := byID[c.question]
		if !ok {
			t.Errorf("%s: not found in results", c.question)
			continue
		}
		if !stringSlicesEqual(ans.Selected, c.selected) {
			t.Errorf("%s: selected = %v, want %v",
				c.question, ans.Selected, c.selected)
		}
		if ans.Flag != c.flagged {
			t.Errorf("%s: flag = %v, want %v",
				c.question, ans.Flag, c.flagged)
		}
	}
}
