package dto

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//
// Helper functions
//

func gzipBody(t *testing.T, s []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)

	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func deflateBody(t *testing.T, s []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, 1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

type requestBody struct {
	Name string `json:"name"`
}

func (b requestBody) Validate() error {
	if b.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

//
// Tests
//

func TestParseJsonBody(t *testing.T) {

	tests := []struct {
		name        string
		body        []byte
		encoding    string
		maxSize     uint64
		want        requestBody
		wantOK      bool
		wantStatus  int
	}{
		{
			name:       "json",
			body:       []byte(`{"name":"Alice"}`),
			maxSize:    1<<10,
			want:       requestBody{Name: "Alice"},
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "gzip",
			body:       gzipBody(t, []byte(`{"name":"Alice"}`)),
			encoding:   "gzip",
			maxSize:    1<<10,
			want:       requestBody{Name: "Alice"},
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "deflate",
			body:       deflateBody(t, []byte(`{"name":"Alice"}`)),
			encoding:   "deflate",
			maxSize:    1<<10,
			want:       requestBody{Name: "Alice"},
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed json",
			body:       []byte(`{"name":`),
			maxSize:    1<<10,
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported encoding",
			body:       []byte(`{"name":"Alice"}`),
			encoding:   "br",
			maxSize:    1<<10,
			wantOK:     false,
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:       "body too large",
			body:       []byte(`{"name":"Alice"}`),
			maxSize:    4,
			wantOK:     false,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(
				http.MethodPost,
				"/",
				bytes.NewReader(tt.body),
			)
			r.Header.Set("Content-Type", "application/json")

			if tt.encoding != "" {
				r.Header.Set("Content-Encoding", tt.encoding)
			}

			w := httptest.NewRecorder()

			got, ok := ParseJsonBody[requestBody](w, r, tt.maxSize)

			if ok != tt.wantOK {
				t.Fatalf("ParseBody() ok = %v, want %v", ok, tt.wantOK)
			}

			if ok && got != tt.want {
				t.Errorf("ParseBody() = %+v, want %+v", got, tt.want)
			}

			if !ok && w.Code != tt.wantStatus {
				t.Errorf(
					"ParseBody() status = %d, want %d; body = %q",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}

func TestParseBodyValidationError(t *testing.T) {

	r := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(`{"name":""}`),
	)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	_, ok := ParseJsonBody[requestBody](w, r, 1<<10)

	if ok {
		t.Fatal("ParseBody() ok = true, want false")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("ParseBody() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		encoding    string
		want        string
		wantErr     error
	}{
		{
			name: "no encoding",
			body: []byte("original"),
			want: "original",
		},
		{
			name:     "gzip",
			body:     gzipBody(t, []byte("original")),
			encoding: "gzip",
			want:     "original",
		},
		{
			name:     "deflate",
			body:     deflateBody(t, []byte("original")),
			encoding: "deflate",
			want:     "original",
		},
		{
			name:     "unsupported encoding",
			body:     []byte("original"),
			encoding: "compress",
			wantErr:  errUnsupportedContentEncoding,
		},
		{
			name:     "malformed gzip",
			body:     []byte("not gzip data"),
			encoding: "gzip",
			wantErr:  errMalformedContent,
		},
		{
			name:     "multiple compressions",
			body:     gzipBody(t, deflateBody(t, []byte("original"))),
			encoding: "deflate, gzip",
			want:     "original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
				Body:   io.NopCloser(bytes.NewReader(tt.body)),
			}

			if tt.encoding != "" {
				req.Header.Set("Content-Encoding", tt.encoding)
			}

			got, err := decode(req)

			if err != tt.wantErr {
				t.Fatalf("decode() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			defer got.Close()

			data, err := io.ReadAll(got)
			if err != nil {
				t.Fatalf("reading decoded body: %v", err)
			}

			if string(data) != tt.want {
				t.Errorf("decoded body = %q, want %q", data, tt.want)
			}
		})
	}
}

type query struct {
	Name   string
	Count  int
	Limit  uint
	Offset int64
	Size   uint32 `default:"10"`
}

func (q query) Validate() error { return nil }

func TestParseQuery(t *testing.T) {

	tests := []struct {
		name       string
		url        string
		want       query
		wantOK     bool
		wantStatus int
	}{
		{
			name: "all fields",
			url:  "/?name=hello&count=-5&limit=10&offset=-20&size=50",
			want: query{
				Name:   "hello",
				Count:  -5,
				Limit:  10,
				Offset: -20,
				Size:   50,
			},
			wantOK: true,
		},
		{
			name: "default value",
			url:  "/?name=hello&count=5&limit=10&offset=20",
			want: query{
				Name:   "hello",
				Count:  5,
				Limit:  10,
				Offset: 20,
				Size:   10,
			},
			wantOK: true,
		},
		{
			name:       "missing required parameter",
			url:        "/?count=5&limit=10&offset=20",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid signed integer",
			url:        "/?name=hello&count=abc&limit=10&offset=20",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid unsigned integer",
			url:        "/?name=hello&count=5&limit=abc&offset=20",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative unsigned integer",
			url:        "/?name=hello&count=5&limit=-1&offset=20",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "integer overflow",
			url:        "/?name=hello&count=999999999999999999999999&limit=10&offset=20",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			got, ok := ParseQuery[query](w, r)

			if ok != tt.wantOK {
				t.Fatalf("ParseQuery() ok = %v, want %v", ok, tt.wantOK)
			}

			if ok && got != tt.want {
				t.Errorf("ParseQuery() = %+v, want %+v", got, tt.want)
			}

			if !ok && w.Code != tt.wantStatus {
				t.Errorf(
					"ParseQuery() status = %d, want %d; body = %q",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}

type queryWithAllTypes struct {
	Int    int
	Int8   int8
	Int16  int16
	Int32  int32
	Int64  int64
	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
	Bool   bool
}

func (q queryWithAllTypes) Validate() error { return nil }

func TestParseQueryIntegerTypes(t *testing.T) {
	r := httptest.NewRequest(
		http.MethodGet,
		"/?int=-1"+
			"&int8=-8"+
			"&int16=-16"+
			"&int32=-32"+
			"&int64=-64"+
			"&uint=1"+
			"&uint8=8"+
			"&uint16=16"+
			"&uint32=32"+
			"&uint64=64"+
			"&bool=true",
		nil,
	)

	w := httptest.NewRecorder()

	got, ok := ParseQuery[queryWithAllTypes](w, r)

	if !ok {
		t.Fatalf("ParseQuery() ok = false, status = %d, body = %q",
			w.Code, w.Body.String())
	}

	want := queryWithAllTypes{
		Int:    -1,
		Int8:   -8,
		Int16:  -16,
		Int32:  -32,
		Int64:  -64,
		Uint:   1,
		Uint8:  8,
		Uint16: 16,
		Uint32: 32,
		Uint64: 64,
		Bool:   true,
	}

	if got != want {
		t.Errorf("ParseQuery() = %+v, want %+v", got, want)
	}
}
