package dto

import (
	"encoding/json"
	"fmt"
)

// Parses and validates a preprocessing template from JSON text.
func ParsePreprocessingTemplate(
	jsonBuf []byte,
) (*PreprocessingTemplate, error) {
	v := &PreprocessingTemplate{}
	if err := json.Unmarshal(jsonBuf, v); err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return v, nil
}

type PreprocessingTemplate struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Config struct {
		BlurSize            int     `json:"blurSize"`
		MorphCloseSize      int     `json:"morphCloseSize"`
		MinAnchorConfidence float64 `json:"minAnchorConfidence"`
		AdaptiveBlockSize   int     `json:"adaptiveBlockSize"`
		AdaptiveC           float64 `json:"adaptiveC"`
	} `json:"config"`
	Pages []struct {
		Anchors []struct {
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
		for _, anchor := range page.Anchors {
			err := inBounds(p.Width, p.Height, anchor.Center.X, anchor.Center.Y)
			if err != nil {
				return err
			}
			err = inBounds(
				p.Width, p.Height,
				anchor.Roi.Min.X, anchor.Roi.Min.Y,
			)
			if err != nil {
				return err
			}
			err = inBounds(
				p.Width, p.Height,
				anchor.Roi.Max.X, anchor.Roi.Max.Y,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//
// General helper functions
//

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
