package marker

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"gocv.io/x/gocv"
)

func ptr(f float64) *float64 { return &f }

func assertError(t *testing.T, err error, expectError bool, errContains string) {
	t.Helper()
	if expectError {
		if err == nil {
			t.Fatalf("expected an error containing %q, but got nil", errContains)
		}
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("expected error to contain %q, but got %q",
				errContains, err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("did not expect an error, but got: %v", err)
	}
}

// fillBubble paints a white filled circle into img at the inset region of the
// bubble centered at (cx, cy) with dimensions (bw, bh). The circle radius
// matches what bubbleFillRatio samples, so a filled bubble reads back as ~1.0.
func fillBubble(img *gocv.Mat, cx, cy, bw, bh int, inset float64) {
	r := bw / 2
	if bh < bw {
		r = bh / 2
	}
	gocv.Circle(img,
		image.Pt(cx, cy),
		int(float64(r)*inset),
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
		-1,
	)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoadTemplate(t *testing.T) {
	validJSON := `{
		"config": {},
		"questions": [{
			"id": "Q1",
			"bubbleWidth": 30,
			"bubbleHeight": 30,
			"options": [{"label": "A", "x": 0, "y": 0}]
		}]
	}`

	tests := []struct {
		name        string
		json        string
		expectError bool
		errContains string
	}{
		{name: "Valid template", json: validJSON},
		{name: "Invalid JSON", json: `{bad`, expectError: true, errContains: "parse"},
		{name: "Empty input", json: ``, expectError: true, errContains: "parse"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := LoadTemplate(strings.NewReader(tc.json))
			assertError(t, err, tc.expectError, tc.errContains)
			if !tc.expectError && tmpl == nil {
				t.Error("expected non-nil template")
			}
		})
	}
}

func TestBubbleFillRatio(t *testing.T) {
	const bw, bh = 30, 30

	white := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(255, 0, 0, 0), 100, 100, gocv.MatTypeCV8UC1)
	defer white.Close()

	black := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(0, 0, 0, 0), 100, 100, gocv.MatTypeCV8UC1)
	defer black.Close()

	tests := []struct {
		name    string
		img     gocv.Mat
		bubble  Bubble
		wantMin float64
		wantMax float64
	}{
		{
			name:    "Fully white image gives high fill ratio",
			img:     white,
			bubble:  Bubble{X: 10, Y: 10},
			wantMin: 0.9,
			wantMax: 1.0,
		},
		{
			name:    "Fully black image gives zero fill ratio",
			img:     black,
			bubble:  Bubble{X: 10, Y: 10},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "Bubble entirely outside image gives zero",
			img:     white,
			bubble:  Bubble{X: 200, Y: 200},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "Partially clipped bubble is clamped without panicking",
			img:     white,
			bubble:  Bubble{X: 90, Y: 10},
			wantMin: 0.0,
			wantMax: 1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bubbleFillRatio(tc.img, tc.bubble, bw, bh, 0.75)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("bubbleFillRatio = %.3f, want in [%.3f, %.3f]",
					got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	const (
		imgW, imgH = 400, 400
		bw, bh     = 30, 30
		inset      = 0.75
	)

	singleQ := Question{
		ID: "Q1", Type: "single",
		BubbleWidth: bw, BubbleHeight: bh,
		Options: []Bubble{
			{Label: "A", X: 65, Y: 115},
			{Label: "B", X: 115, Y: 115},
			{Label: "C", X: 165, Y: 115},
		},
	}
	multiQ := Question{
		ID: "Q1", Type: "multi",
		BubbleWidth: bw, BubbleHeight: bh,
		Options: []Bubble{
			{Label: "A", X: 65, Y: 115},
			{Label: "B", X: 115, Y: 115},
		},
	}

	defaultConfig := Config{
		FillThreshold: ptr(0.5),
		BubbleInset:   ptr(inset),
		FlagThreshold: ptr(0.5),
	}

	// Image with only bubble A of singleQ filled.
	imgOneSelected := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(0, 0, 0, 0), imgH, imgW, gocv.MatTypeCV8UC1)
	defer imgOneSelected.Close()
	fillBubble(&imgOneSelected, 65, 115, bw, bh, inset)

	// Image with bubbles A and B of singleQ both filled.
	imgTwoSelected := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(0, 0, 0, 0), imgH, imgW, gocv.MatTypeCV8UC1)
	defer imgTwoSelected.Close()
	fillBubble(&imgTwoSelected, 65, 115, bw, bh, inset)
	fillBubble(&imgTwoSelected, 115, 115, bw, bh, inset)

	// All-black image: no bubbles filled.
	imgBlank := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(0, 0, 0, 0), imgH, imgW, gocv.MatTypeCV8UC1)
	defer imgBlank.Close()

	emptyImg := gocv.NewMat()
	defer emptyImg.Close()

	tests := []struct {
		name         string
		img          gocv.Mat
		tmpl         *Template
		expectError  bool
		errContains  string
		wantSelected [][]string
		wantFlagged  []bool
	}{
		{
			name:        "Empty image returns error",
			img:         emptyImg,
			tmpl:        &Template{Config: defaultConfig, Questions: []Question{singleQ}},
			expectError: true,
			errContains: "empty image",
		},
		{
			name:        "Template with no questions returns error",
			img:         imgOneSelected,
			tmpl:        &Template{Config: defaultConfig, Questions: []Question{}},
			expectError: true,
			errContains: "no questions",
		},
		{
			name:         "Single-select: one bubble filled is not flagged",
			img:          imgOneSelected,
			tmpl:         &Template{Config: defaultConfig, Questions: []Question{singleQ}},
			wantSelected: [][]string{{"A"}},
			wantFlagged:  []bool{false},
		},
		{
			name:         "Single-select: no bubble filled is flagged",
			img:          imgBlank,
			tmpl:         &Template{Config: defaultConfig, Questions: []Question{singleQ}},
			wantSelected: [][]string{nil},
			wantFlagged:  []bool{true},
		},
		{
			name:         "Single-select: two bubbles filled is flagged",
			img:          imgTwoSelected,
			tmpl:         &Template{Config: defaultConfig, Questions: []Question{singleQ}},
			wantSelected: [][]string{{"A", "B"}},
			wantFlagged:  []bool{true},
		},
		{
			name:         "Multi-select: two bubbles filled is not flagged",
			img:          imgTwoSelected,
			tmpl:         &Template{Config: defaultConfig, Questions: []Question{multiQ}},
			wantSelected: [][]string{{"A", "B"}},
			wantFlagged:  []bool{false},
		},
		{
			name: "FlagThreshold above max confidence flags clear answers",
			img:  imgOneSelected,
			tmpl: &Template{
				Config: Config{
					FillThreshold: ptr(0.5),
					BubbleInset:   ptr(inset),
					FlagThreshold: ptr(1.1),
				},
				Questions: []Question{singleQ},
			},
			wantSelected: [][]string{{"A"}},
			wantFlagged:  []bool{true},
		},
		{
			name: "FlagThreshold zero disables confidence-based flagging",
			img:  imgOneSelected,
			tmpl: &Template{
				Config: Config{
					FillThreshold: ptr(0.5),
					BubbleInset:   ptr(inset),
					FlagThreshold: ptr(0.0),
				},
				Questions: []Question{singleQ},
			},
			wantSelected: [][]string{{"A"}},
			wantFlagged:  []bool{false},
		},
		{
			name:         "Nil config fields fall back to defaults",
			img:          imgOneSelected,
			tmpl:         &Template{Config: Config{}, Questions: []Question{singleQ}},
			wantSelected: [][]string{{"A"}},
			wantFlagged:  []bool{false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Evaluate([]gocv.Mat{tc.img}, tc.tmpl)
			assertError(t, err, tc.expectError, tc.errContains)
			if tc.expectError {
				return
			}

			if len(result.Answers) != len(tc.wantSelected) {
				t.Fatalf("expected %d answers, got %d",
					len(tc.wantSelected), len(result.Answers))
			}
			for i, ans := range result.Answers {
				if !stringSlicesEqual(ans.Selected, tc.wantSelected[i]) {
					t.Errorf("answer[%d]: selected = %v, want %v",
						i, ans.Selected, tc.wantSelected[i])
				}
				if ans.Flag != tc.wantFlagged[i] {
					t.Errorf("answer[%d]: flag = %v, want %v",
						i, ans.Flag, tc.wantFlagged[i])
				}
			}
		})
	}
}
