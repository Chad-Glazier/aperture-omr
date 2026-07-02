/*
This package implements the Data Transfer Objects. That is, the structured data
we expect to send to/from the client. For each of these types, we include
functions for serialization, deserialization, and validation where appropriate.
*/
package dto

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Parses and validates a marking template from JSON text.
func ParseMarkingTemplate(jsonBuf []byte) (*MarkingTemplate, error) {
	v := &MarkingTemplate{}
	if err := json.Unmarshal(jsonBuf, v); err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return v, nil
}

type MarkingTemplate struct {
	Config MarkingConfig `json:"config"`
	Pages  []MarkingPage `json:"pages"`
}

func (t *MarkingTemplate) Validate() error {
	if err := t.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if len(t.Pages) == 0 {
		return fmt.Errorf("pages must contain at least one page")
	}

	questionIDs := make(map[string]struct{})

	for i, page := range t.Pages {
		if err := page.Validate(questionIDs); err != nil {
			return fmt.Errorf("pages[%d]: %w", i, err)
		}
	}

	return nil
}

type MarkingConfig struct {
	FillThreshold float64 `json:"fillThreshold"`
	BubbleInset   float64 `json:"bubbleInset"`
	FlagThreshold float64 `json:"flagThreshold"`
	SearchRadius  int     `json:"searchRadius"`
}

func (c *MarkingConfig) Validate() error {
	if c.FillThreshold < 0 || c.FillThreshold > 1 {
		return fmt.Errorf("fillThreshold must be between 0 and 1")
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

	return nil
}

type MarkingPage struct {
	Questions []Question `json:"questions"`
}

func (p *MarkingPage) Validate(ids map[string]struct{}) error {
	if len(p.Questions) == 0 {
		return fmt.Errorf("questions must contain at least one question")
	}

	for i, q := range p.Questions {
		if _, exists := ids[q.ID]; exists {
			return fmt.Errorf("duplicate question id %q", q.ID)
		}

		ids[q.ID] = struct{}{}

		if err := q.Validate(); err != nil {
			return fmt.Errorf("questions[%d]: %w", i, err)
		}
	}

	return nil
}

type Question struct {
	ID           string           `json:"id"`
	BubbleWidth  int              `json:"bubbleWidth"`
	BubbleHeight int              `json:"bubbleHeight"`
	Type         string           `json:"type,omitempty"`
	Options      []QuestionOption `json:"options"`
}

func (q *Question) Validate() error {

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

		if err := option.Validate(); err != nil {
			return fmt.Errorf("options[%d]: %w", i, err)
		}
	}

	return nil
}

type QuestionOption struct {
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

func (o *QuestionOption) Validate() error {
	if strings.TrimSpace(o.Label) == "" {
		return fmt.Errorf("label cannot be empty")
	}

	if o.X < 0 {
		return fmt.Errorf("x must be non-negative")
	}

	if o.Y < 0 {
		return fmt.Errorf("y must be non-negative")
	}

	return nil
}
