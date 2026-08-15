package dto

import "fmt"

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
