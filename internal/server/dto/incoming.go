package dto

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
)

//
// This file defines incoming request bodies. Each of the structs implements
// [Validator], so that they can be parsed with [ParseJsonBody].
//

var (
	ErrInvalidBase64             = errors.New("invalid base64 encoding")
	ErrUnrecognizedImageEncoding = errors.New("image encoding not recognized")
)

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

func (m MarkingJobRequest) Validate() error {

	switch {
	case m.TemplateId == "":
		return errors.New("template field is required")
	case m.ScanIds == nil:
		return errors.New("scans field is required")
	case len(m.ScanIds) == 0:
		return errors.New("scans must not be empty")
	}

	if slices.Contains(m.ScanIds, "") {
		return errors.New("scan ID strings cannot be empty")
	}

	return nil
}

//
// MarkTemplate
//

type MarkTemplate struct {
	Aspect            float64        `json:"aspectRatio"`
	BubbleRadius      float64        `json:"bubbleRadius"`
	MinimumConfidence float64        `json:"minimumConfidence"`
	Binarization      BinarizeConfig `json:"binarization"`

	Pages []Page `json:"pages"`
}

func (m MarkTemplate) Validate() error {
	switch {
	case m.Aspect <= 0:
		return errors.New("the aspect ratio must be positive (width/height)")
	case m.BubbleRadius <= 0, m.BubbleRadius > 1:
		return errors.New("the bubble radius must be in the interval (0, 1]")
	case len(m.Pages) == 0:
		return errors.New("the page list cannot be empty")
	}

	if err := m.Binarization.Validate("binarization:"); err != nil {
		return err
	}

	for i, p := range m.Pages {
		prefix := fmt.Sprintf("page %d:", i)
		if err := p.Validate(prefix); err != nil {
			return err
		}

		questionIds := make(map[string]bool)
		for _, q := range p.Questions {
			if questionIds[q.Id] {
				return fmt.Errorf("page %d: duplicate question IDs are not allowed", i)
			}
			questionIds[q.Id] = true
		}
	}

	return nil
}

type Page struct {
	Questions []Question `json:"questions"`
}

func (p Page) Validate(prefix string) error {
	switch {
	case len(p.Questions) == 0:
		return errors.New(prefix + " a page's question list cannot be empty")
	}

	for i, q := range p.Questions {
		qPrefix := fmt.Sprintf("question %d:", i)
		if err := q.Validate(prefix + " " + qPrefix); err != nil {
			return err
		}
	}

	return nil
}

type Question struct {
	Id      string   `json:"id"`
	Bubbles []Bubble `json:"bubbles"`
}

func (q Question) Validate(prefix string) error {
	switch {
	case q.Id == "":
		return errors.New(prefix + " a question's id cannot be empty")
	case len(q.Bubbles) == 0:
		return errors.New(prefix + " a question's bubble list cannot be empty")
	}

	bubbleIds := make(map[string]bool)
	for i, b := range q.Bubbles {

		bPrefix := fmt.Sprintf("bubble %d:", i)
		if err := b.Validate(prefix + " " + bPrefix); err != nil {
			return err
		}

		if bubbleIds[b.Id] {
			return errors.New(prefix + " bubble IDs must be unique (per question)")
		}
		bubbleIds[b.Id] = true
	}

	return nil
}

type Bubble struct {
	Id string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

func (b Bubble) Validate(prefix string) error {
	switch {
	case b.Id == "":
		return errors.New(prefix + " a bubble's id cannot be empty")
	}

	return nil
}

type BinarizeConfig struct {
	BlurSize       float64 `json:"blurSize"`
	MorphCloseSize float64 `json:"morphCloseSize"`
	BlockSize      float64 `json:"blockSize"`
	AdaptiveC      float64 `json:"adaptiveC"`
}

func (b BinarizeConfig) Validate(prefix string) error {
	switch {
	case b.BlurSize <= 0, b.BlurSize > 1:
		return errors.New(prefix + " blurSize must be in the interval (0, 1]")
	case b.BlockSize <= 0:
		return errors.New(prefix + " blockSize must be greater than 0")
	case b.MorphCloseSize <= 0, b.MorphCloseSize > 1:
		return errors.New(prefix + " morphCloseSize must be in the interval (0, 1]")
	}

	return nil
}

// Returns the equivalent [omr.MarkTemplate] represented by this struct.
func (m MarkTemplate) Adapt() omr.MarkTemplate {

	var out omr.MarkTemplate

	out.Aspect = m.Aspect
	out.BubbleRadius = m.BubbleRadius
	out.MinimumConfidence = m.MinimumConfidence
	out.Binarization = omr.BinarizeConfig(m.Binarization)

	out.Questions = make([][]omr.Question, len(m.Pages))
	for i, p := range m.Pages {
		out.Questions[i] = make([]omr.Question, len(p.Questions))
		for j, q := range p.Questions {
			out.Questions[i][j].Bubbles = make([]omr.Bubble, len(q.Bubbles))
			for k, b := range q.Bubbles {
				out.Questions[i][j].Bubbles[k].Id = b.Id
				out.Questions[i][j].Bubbles[k].Pos.X = b.X
				out.Questions[i][j].Bubbles[k].Pos.Y = b.Y
			}
		}
	}

	return out
}

//
// PreprocessTemplate
//

type PreprocessTemplate struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`

	MinAnchorConfidence float64            `json:"minAnchorConfidence"`
	AnchorSearchConfig  AnchorSearchConfig `json:"anchorSearchConfig"`

	Anchors [][]Anchor `json:"anchors"`
}

func (p PreprocessTemplate) Validate() error {
	switch {
	case p.Width < 1, p.Height < 1:
		return errors.New("the template dimensions must be positive")
	case p.MinAnchorConfidence >= 1, p.MinAnchorConfidence <= 0:
		return errors.New("the minAnchorConfidence must be in the interval (0, 1)")
	case len(p.Anchors) < 1:
		return errors.New("the template needs anchors for at least one page")
	}

	if err := p.AnchorSearchConfig.Validate("anchorSearchConfig:"); err != nil {
		return err
	}

	for i := range p.Anchors {
		if len(p.Anchors[i]) < 3 {
			return fmt.Errorf("page %d: each page must have at least 3 anchors", i)
		}

		for j, a := range p.Anchors[i] {
			prefix := fmt.Sprintf("anchor %d, %d:", i, j)
			if err := a.Validate(prefix); err != nil {
				return err
			}
		}
	}

	return nil
}

type AnchorSearchConfig struct {
	InitialAngle       float64 `json:"initialAngle"`
	AngleSearchBreadth float64 `json:"angleSearchBreadth"`
	SearchAreaPadding  float64 `json:"searchAreaPadding"`
	MaxQuality         float64 `json:"maxQuality"`
}

func (a AnchorSearchConfig) Validate(prefix string) error {
	switch {
	case a.AngleSearchBreadth < 0, a.AngleSearchBreadth >= math.Pi/2:
		return errors.New(prefix + " angleSearchBreadth must be in the interval [0, \u03c0/2] (in degrees, [0\u00b0, 90\u00b0])")
	case a.SearchAreaPadding < 0, a.AngleSearchBreadth > 1:
		return errors.New(prefix + " searchAreaPadding must be in the interval [0, 1]")
	case a.MaxQuality < 0, a.MaxQuality > 1:
		return errors.New(prefix + " maxQuality must be in the interval [0, 1]")
	}

	return nil
}

type Anchor struct {
	Image string  `json:"image"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

func (a Anchor) Validate(prefix string) error {
	switch {
	case a.Image == "":
		return errors.New(prefix + " an anchor's image field cannot be empty")
	case a.X < 0, a.X > 1, a.Y < 0, a.Y > 1:
		return errors.New(prefix + " an anchor's coordinates should be in the interval [0, 1]")
	}

	return nil
}

// Returns the equivalent [omr.PreprocessTemplate] represented by this struct.
//
// If an error is returned, it will be PLACEHOLDER.
func (p PreprocessTemplate) Adapt() (omr.PreprocessTemplate, error) {

	var out omr.PreprocessTemplate

	out.AnchorSearchConfig = omr.FindAnchorConfig(p.AnchorSearchConfig)
	out.Height = p.Height
	out.Width = p.Width
	out.MinAnchorConfidence = p.MinAnchorConfidence

	out.Anchors = make([][]omr.Anchor, len(p.Anchors))
	for i := range p.Anchors {

		out.Anchors[i] = make([]omr.Anchor, len(p.Anchors[i]))
		for j, a := range p.Anchors[i] {

			mat, err := omr.DecodeBase64(a.Image)
			switch err {
			case nil:
			case omr.ErrBase64Decoding:
				return omr.PreprocessTemplate{}, ErrInvalidBase64
			default:
				return omr.PreprocessTemplate{}, ErrUnrecognizedImageEncoding
			}

			out.Anchors[i][j].Pos.X = a.X
			out.Anchors[i][j].Pos.Y = a.Y
			out.Anchors[i][j].Mat = mat
		}
	}

	return out, nil
}
