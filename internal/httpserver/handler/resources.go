package handler

import (

)

//
// The ServerResources interface defines things that should be shared between
// requests. E.g., access to data/file stores.
//

type ServerResources interface {
	// Saves a marking template and returns the new ID for it or an error if
	// the operation failed.
	SaveMarkingTemplate(tmpl *MarkingTemplate) (string, error)
}
