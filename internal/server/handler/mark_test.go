package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

//
func checkCanonicalMarks(t *testing.T, w httptest.ResponseRecorder) {

	buf, err := testData.ReadFile("testdata/marks/canonical.json")
	assert.Assert(t, err == nil)

	var want dto.MarkingResult
	err = json.Unmarshal(buf, &want)
	assert.Assert(t, err == nil)

	var got dto.MarkingResult
	err = json.Unmarshal(w.Body.Bytes(), &got)
	assert.Assert(t, err == nil)

	
}
