package dto

import (
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/Chad-Glazier/aperture-omr/internal/marker"
	"github.com/Chad-Glazier/aperture-omr/internal/scanner"
	"gocv.io/x/gocv"
)

//
// This file defines incoming request bodies. Each of the structs implements
// [Validator], so that they can be parsed with [ParseJsonBody].
//

//
// ScanDeleteRequest
//

type ScanDeleteRequest []string

func (s ScanDeleteRequest) Validate() error { return nil }

//
// MarkingJobRequest
//

type MarkingJobRequest struct {
	TemplateId string   `json:"template"`
	ScanIds    []string `json:"scans"`
}

func (m *MarkingJobRequest) Validate() error {

	switch {
	case m.TemplateId == "":
		return errors.New("template field must be provided")
	case m.ScanIds == nil:
		return errors.New("scans must be provided")
	case len(m.ScanIds) == 0:
		return errors.New("scans must not be empty")
	}

	return nil
}

//
// MarkingTemplate
//

type MarkingTemplate struct {
	Config MarkingConfig `json:"config"`
	Pages  []MarkingPage `json:"pages"`
}

type MarkingConfig struct {
	FillThreshold float64 `json:"fillThreshold"`
	// SelectionThreshold, when set, is used in place of FillThreshold for
	// the final gap-based selection cutoff only; see
	// marker.Config.SelectionThreshold for why that needs to be a separate
	// knob from FillThreshold's other (alignment-fallback) role. This one
	// must stay a pointer, unlike FillThreshold/FlagThreshold above: unset
	// has to mean "fall back to FillThreshold", not silently become 0,
	// which would disable gap-based selection for any question that
	// doesn't set it.
	SelectionThreshold *float64 `json:"selectionThreshold,omitempty"`
	BubbleInset        float64  `json:"bubbleInset"`
	FlagThreshold      float64  `json:"flagThreshold"`
	SearchRadius       int      `json:"searchRadius"`
	// SearchGroupSize/GroupSearchRadius split a wide row into
	// independently-aligned groups instead of one shared-offset search for
	// the whole row; see marker.Config.SearchGroupSize for the full
	// rationale. Both 0 (default) keeps SearchRadius's original behavior.
	SearchGroupSize   int `json:"searchGroupSize"`
	GroupSearchRadius int `json:"groupSearchRadius"`
}

type MarkingPage struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	ID           string           `json:"id"`
	BubbleWidth  int              `json:"bubbleWidth"`
	BubbleHeight int              `json:"bubbleHeight"`
	Type         string           `json:"type,omitempty"`
	Options      []QuestionOption `json:"options"`
}

type QuestionOption struct {
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	// BaselineBias corrects for this option's own printed glyph reading a
	// different natural (unmarked) ink level than its question's other
	// options; see marker.Bubble.BaselineBias for the full rationale. 0
	// (default) leaves detection exactly as before.
	BaselineBias float64 `json:"baselineBias"`
}

func (t *MarkingTemplate) Validate() error {
	if err := t.Config.validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if len(t.Pages) == 0 {
		return fmt.Errorf("pages must contain at least one page")
	}

	questionIDs := make(map[string]struct{})

	for i, page := range t.Pages {
		if err := page.validate(questionIDs); err != nil {
			return fmt.Errorf("pages[%d]: %w", i, err)
		}
	}

	return nil
}

func (c *MarkingConfig) validate() error {
	if c.FillThreshold < 0 || c.FillThreshold > 1 {
		return fmt.Errorf("fillThreshold must be between 0 and 1")
	}

	if c.SelectionThreshold != nil && (*c.SelectionThreshold < 0 || *c.SelectionThreshold > 1) {
		return fmt.Errorf("selectionThreshold must be between 0 and 1")
	}

	if c.BubbleInset < 0 || c.BubbleInset > 1 {
		return fmt.Errorf("bubbleInset must be between 0 and 1")
	}

	if c.FlagThreshold < 0 || c.FlagThreshold > 1 {
		return fmt.Errorf("flagThreshold must be between 0 and 1")
	}

	if c.SearchRadius < 0 {
		return fmt.Errorf("searchRadius must be nonnegative")
	}

	if c.SearchGroupSize < 0 {
		return fmt.Errorf("searchGroupSize must be nonnegative")
	}

	if c.GroupSearchRadius < 0 {
		return fmt.Errorf("groupSearchRadius must be nonnegative")
	}

	return nil
}

func (p *MarkingPage) validate(ids map[string]struct{}) error {
	if len(p.Questions) == 0 {
		return fmt.Errorf("questions must contain at least one question")
	}

	for i, q := range p.Questions {
		if _, exists := ids[q.ID]; exists {
			return fmt.Errorf("duplicate question id %q", q.ID)
		}

		ids[q.ID] = struct{}{}

		if err := q.validate(); err != nil {
			return fmt.Errorf("questions[%d]: %w", i, err)
		}
	}

	return nil
}

func (q *Question) validate() error {

	switch {
	case q.BubbleWidth <= 0:
		return fmt.Errorf("bubbleWidth must be positive")
	case q.BubbleHeight <= 0:
		return fmt.Errorf("bubbleHeight must be positive")
	case q.Type != "" && q.Type != "single" && q.Type != "multi":
		return fmt.Errorf(
			"type must be \"multi\", \"single\", " +
				"or omitted (defaults to \"single\")",
		)
	case len(q.Options) == 0:
		return fmt.Errorf("question must contain at least one option")
	}

	labels := make(map[string]struct{})

	for i, option := range q.Options {
		if _, exists := labels[option.Label]; exists {
			return fmt.Errorf("duplicate option label %q", option.Label)
		}

		labels[option.Label] = struct{}{}

		if err := option.validate(); err != nil {
			return fmt.Errorf("options[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *QuestionOption) validate() error {
	if strings.TrimSpace(o.Label) == "" {
		return errors.New("label cannot be empty")
	}

	if o.X < 0 {
		return errors.New("x must be non-negative")
	}

	if o.Y < 0 {
		return errors.New("y must be non-negative")
	}

	return nil
}

// Converts a MarkingTemplate into a [marker.Template].
func AdaptMarkerTemplate(tmpl *MarkingTemplate) *marker.Template {

	template := marker.Template{
		Config: marker.Config{
			FillThreshold:      &tmpl.Config.FillThreshold,
			SelectionThreshold: tmpl.Config.SelectionThreshold,
			BubbleInset:        &tmpl.Config.BubbleInset,
			FlagThreshold:      &tmpl.Config.FlagThreshold,
			SearchRadius:       &tmpl.Config.SearchRadius,
			SearchGroupSize:    &tmpl.Config.SearchGroupSize,
			GroupSearchRadius:  &tmpl.Config.GroupSearchRadius,
		},
		Pages: make([]marker.Page, len(tmpl.Pages)),
	}

	for i, p := range tmpl.Pages {

		template.Pages[i].Questions = make(
			[]marker.Question,
			len(p.Questions),
		)
		for j, q := range p.Questions {

			template.Pages[i].Questions[j].ID = q.ID
			template.Pages[i].Questions[j].Type = q.Type
			template.Pages[i].Questions[j].BubbleWidth = q.BubbleWidth
			template.Pages[i].Questions[j].BubbleHeight = q.BubbleHeight
			template.Pages[i].Questions[j].Options = make(
				[]marker.Bubble, len(q.Options),
			)

			for k, o := range q.Options {
				template.Pages[i].Questions[j].Options[k].Label = o.Label
				template.Pages[i].Questions[j].Options[k].X = o.X
				template.Pages[i].Questions[j].Options[k].Y = o.Y
				template.Pages[i].Questions[j].Options[k].BaselineBias = o.BaselineBias
			}

		}
	}

	return &template
}


//
// PreprocessingTemplate
//

type PreprocessingTemplate struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Config    struct {
		BlurSize            int     `json:"blurSize"`
		MorphCloseSize      int     `json:"morphCloseSize"`
		MinAnchorConfidence float64 `json:"minAnchorConfidence"`
		AdaptiveBlockSize   int     `json:"adaptiveBlockSize"`
		AdaptiveC           float64 `json:"adaptiveC"`
	} `json:"config"`
	Pages []struct {
		Anchors []struct {
			Image string `json:"image"` // base64-encoded
			Center struct {
				X int `json:"x"`
				Y int `json:"y"`
			} `json:"center"`
			Roi struct {
				Min struct {
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"min"`
				Max struct {
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"max"`
			} `json:"roi"`
		} `json:"anchors"`
	} `json:"pages"`
}

func (p *PreprocessingTemplate) Validate() error {
	switch {
	case p.Width <= 0:
		return fmt.Errorf("width must be positive")
	case p.Height <= 0:
		return fmt.Errorf("height must be positive")
	case p.Config.BlurSize <= 0:
		return fmt.Errorf("blurSize must be nonnegative")
	case p.Config.BlurSize%2 == 0:
		return fmt.Errorf("blurSize must be odd")
	case p.Config.MorphCloseSize < 0:
		return fmt.Errorf("morphCloseSize must be nonnegative")
	case p.Config.MinAnchorConfidence <= 0.0,
		p.Config.MinAnchorConfidence >= 1.0:
		return fmt.Errorf("minAnchorConfidence must be in (0.0, 1.0)")
	case len(p.Pages) == 0:
		return fmt.Errorf("at least one page must be included")
	case p.Config.AdaptiveBlockSize == 0:
		return fmt.Errorf("adaptiveBlockSize is required")
	case p.Config.AdaptiveBlockSize%2 == 0:
		return fmt.Errorf("adaptiveBlockSize must be odd")
	case p.Config.AdaptiveBlockSize <= 0:
		return fmt.Errorf("adaptiveBlockSize must be positive")
	}

	const minAnchors = 3
	for _, page := range p.Pages {
		if len(page.Anchors) < minAnchors {
			return fmt.Errorf(
				"each page must have at least %d anchors",
				minAnchors,
			)
		}
		for i := range page.Anchors {
			if page.Anchors[i].Image == "" {
				return fmt.Errorf(
					"base64-encoded images must be set for each anchor",
				)
			}
			err := inBounds(
				p.Width, p.Height, 
				page.Anchors[0].Center.X, page.Anchors[0].Center.Y,
			)
			if err != nil {
				return err
			}
			err = inBounds(
				p.Width, p.Height,
				page.Anchors[0].Roi.Min.X, page.Anchors[0].Roi.Min.Y,
			)
			if err != nil {
				return err
			}
			err = inBounds(
				p.Width, p.Height,
				page.Anchors[0].Roi.Max.X, page.Anchors[0].Roi.Max.Y,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func inBounds(width, height, x, y int) error {
	switch {
	case x < 0:
		return fmt.Errorf("x-coordinates must be nonnegative")
	case y < 0:
		return fmt.Errorf("y-coordinates must be nonnegative")
	case x > width:
		return fmt.Errorf("x-coordinates must be less than the width")
	case y > height:
		return fmt.Errorf("y-coordinates must be less than the height")
	}
	return nil
}

// Converts a PreprocessingTemplate into a [scanner.Template], populating it 
// with the given anchors. An error will be returned if the number of anchors 
// does not match the number of anchors expected by the template.
func AdaptScannerTemplate(
	tmpl *PreprocessingTemplate,
	anchors [][]gocv.Mat,
) (*scanner.Template, error) {

	nPages := len(tmpl.Pages)
	if len(anchors) != nPages {
		return nil, fmt.Errorf(
			"preprocessing template page count (%d) doesn't match anchors "+
				"given (%d)",
			nPages, len(anchors),
		)
	}

	for i := range nPages {
		if len(tmpl.Pages[i].Anchors) != len(anchors[i]) {
			return nil, fmt.Errorf(
				"preprocessing template anchor count for page %d (%d) "+
					"doesn't match anchors given (%d)",
				i, len(tmpl.Pages[i].Anchors), len(anchors[i]),
			)
		}
	}

	scannerTmpl := scanner.Template{
		Width:     tmpl.Width,
		Height:    tmpl.Height,
		Config: scanner.Config{
			BlurSize:            tmpl.Config.BlurSize,
			MorphCloseSize:      tmpl.Config.MorphCloseSize,
			MinAnchorConfidence: float32(tmpl.Config.MinAnchorConfidence),
			AdaptiveBlockSize:   tmpl.Config.AdaptiveBlockSize,
			AdaptiveC:           float32(tmpl.Config.AdaptiveC),
		},
		Pages: make([]scanner.ScanPage, nPages),
	}

	for pageIdx := range nPages {
		nAnchors := len(tmpl.Pages[pageIdx].Anchors)
		scannerTmpl.Pages[pageIdx].Anchors = make(
			[]scanner.Anchor,
			nAnchors,
		)
		for anchorIdx := range nAnchors {
			scannerTmpl.Pages[pageIdx].Anchors[anchorIdx] = scanner.Anchor{
				Image: anchors[pageIdx][anchorIdx],
				ROI: image.Rectangle{
					Min: image.Point{
						X: tmpl.Pages[pageIdx].Anchors[anchorIdx].Roi.Min.X,
						Y: tmpl.Pages[pageIdx].Anchors[anchorIdx].Roi.Min.Y,
					},
					Max: image.Point{
						X: tmpl.Pages[pageIdx].Anchors[anchorIdx].Roi.Max.X,
						Y: tmpl.Pages[pageIdx].Anchors[anchorIdx].Roi.Max.Y,
					},
				},
				Center: image.Point{
					X: tmpl.Pages[pageIdx].Anchors[anchorIdx].Center.X,
					Y: tmpl.Pages[pageIdx].Anchors[anchorIdx].Center.Y,
				},
			}
		}
	}

	return &scannerTmpl, nil
}
