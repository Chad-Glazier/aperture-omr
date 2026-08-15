package dto

import (
	"errors"
	"fmt"
)

//
// In this file we implement some types for query parameters, to be used with
// [ParseQuery]. Unlike for the request bodies, some of these types are not
// specific to endpoints and are meant to be more reusable.
//

//
// Utility Types
//

type IdQuery struct {
	Id string
}

func (i IdQuery) Validate() error { 
	return nil 
}

type LimitQuery struct {
	Limit uint
}

func (l LimitQuery) Validate() error {
	if l.Limit == 0 {
		return fmt.Errorf("limit parameter must be positive")
	}
	return nil
}

//
// Endpoint-Specific Types
//

type GetSnippetQuery struct {
	Template string
	Scan     string
	Question string
}

func (g GetSnippetQuery) Validate() error { return nil }


type GetImageQuery struct {
	Scan string
	Page uint
}

func (g GetImageQuery) Validate() error { return nil }

type PostScanPdfQuery struct {
	PreprocessingTemplate string
	Dpi                   uint   `default:"300"`
}

func (p PostScanPdfQuery) Validate() error {
	if p.Dpi == 0 {
		return errors.New("dpi must be positive")
	}
	return nil
}
