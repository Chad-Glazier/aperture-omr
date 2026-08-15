package dto

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
)

//
// This file implements two important "parsers": one to parse a JSON body to
// a struct, and another to parse the query parameters to a struct.
//

type Validator interface {
	// This should return an error if some part of the struct was invalid. The
	// error message should be meaningful and safe to send to the client.
	Validate() error
}

// Parses a request body as JSON, decoding it with gzip and/or deflate as
// instructed by the Content-Encoding header. This function will also close the
// given request body.
//
// The second return value will be false if there was an error. In that case,
// this function will have already sent an appropriate response to the client.
// Possible response statuses include 400 Bad Request, 415 Unsupported Media
// Type, and 413 Request Entity Too Large.
func ParseJsonBody[T Validator](
	w http.ResponseWriter,
	r *http.Request,
	maxSize uint64,
) (T, bool) {

	var v T

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w,
			"expected Content-Type header to be application/json",
			http.StatusUnsupportedMediaType,
		)
		return v, false
	}

	body, err := decode(r)
	if err != nil {
		switch err {
		case errMalformedContent:
			http.Error(w,
				"request body content was not encoded correctly",
				http.StatusBadRequest,
			)
			return v, false
		case errUnsupportedContentEncoding:
			http.Error(w,
				"request Content-Encoding contains an unsupported "+
					"compression format "+
					"(supported formats are gzip and deflate)",
				http.StatusUnsupportedMediaType,
			)
			return v, false
		default:
			panic("dto: decode returned an unexpected error")
		}
	}
	defer body.Close()

	var buf bytes.Buffer

	written, err := io.Copy(&buf, io.LimitReader(body, int64(maxSize+1)))

	if written > int64(maxSize) {
		http.Error(w,
			"maximum content length "+formatMemorySize(maxSize)+" exceeded",
			http.StatusRequestEntityTooLarge,
		)
		return v, false
	}

	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		http.Error(w,
			"malformed JSON in request body",
			http.StatusBadRequest,
		)
		return v, false
	}

	if err := v.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return v, false
	}

	return v, true
}

func formatMemorySize(bytes uint64) string {
	switch {
	case bytes>>30 >= 1:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes>>20 >= 1:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes>>10 >= 1:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Returns the request body except that, if the Content-Encoding header is set
// on the request, the returned reader will be the decoded version.
//
// If an error is returned, it will be [errUnsupportedContentEncoding] or
// [errMalformedContent]. In either of these cases, the request body will be
// closed.
func decode(r *http.Request) (io.ReadCloser, error) {

	if r.Header.Get("Content-Encoding") == "" {
		return r.Body, nil
	}

	rc := r.Body
	encs := strings.Split(r.Header.Get("Content-Encoding"), ", ")
	for len(encs) > 0 {

		switch encs[len(encs)-1] {
		case "gzip":
			newRc, err := gzip.NewReader(rc)
			if err != nil {
				rc.Close()
				return nil, errMalformedContent
			}
			rc = &wrappingReadCloser{
				wrapped: rc,
				rc:      newRc,
			}
		case "deflate":
			rc = &wrappingReadCloser{
				wrapped: rc,
				rc:      flate.NewReader(rc),
			}
		default:
			rc.Close()
			return nil, errUnsupportedContentEncoding
		}

		encs = encs[:len(encs)-1]
	}

	return rc, nil
}

var (
	errUnsupportedContentEncoding = errors.New("unsupported Content-Encoding")
	errMalformedContent           = errors.New("encoded content is malformed")
)

type wrappingReadCloser struct {
	wrapped io.ReadCloser
	rc      io.ReadCloser
}

func (w *wrappingReadCloser) Read(p []byte) (int, error) {
	return w.rc.Read(p)
}

func (w *wrappingReadCloser) Close() error {
	err1 := w.rc.Close()
	err2 := w.wrapped.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

// Parses the query parameters of a request and uses them to populate a struct
// of the given type. For each exported field on the struct, a matching query
// parameter will be checked, except that the query parameter is expected to
// begin with a lowercase letter. For example, the struct field "Foo" will be
// assigned the value of the query parameter "foo". In order to make a value 
// optional, give it a struct tag "default" which is set to its default value.
//
// Currently supported field types include string, bool, all unsigned integer 
// types, and all signed integer types. The function calls [strconv.ParseBool]
// and thereby uses the same rules for converting strings to bools.
//
// The second return value will be false if there was an error. In that case,
// this function will have already sent an appropriate response to the client.
// The only possible response status is 400 Bad Request.
func ParseQuery[T Validator](
	w http.ResponseWriter,
	r *http.Request,
) (T, bool) {

	var v T

	fields := reflect.VisibleFields(reflect.TypeOf(v))

	for _, f := range fields {
		if !f.IsExported() {
			continue
		}

		key := strings.ToLower(string([]rune(f.Name)[0]))
		if len([]rune(f.Name)) > 1 {
			key += string([]rune(f.Name)[1:])
		}

		strVal := r.URL.Query().Get(key)
		defaultStrVal, hasDefault := f.Tag.Lookup("default")

		if strVal == "" && !hasDefault {
			http.Error(w,
				"query parameter "+key+" required",
				http.StatusBadRequest,
			)
			return v, false
		}
		if strVal == "" {
			strVal = defaultStrVal
		}

		switch f.Type.Kind() {
		case reflect.String:
			reflect.ValueOf(&v).Elem().
				FieldByName(f.Name).
				SetString(strVal)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, 
		     reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				http.Error(w,
					"query parameter "+key+" must be a nonnegative integer",
					http.StatusBadRequest,
				)
				return v, false
			}
			reflect.ValueOf(&v).Elem().
				FieldByName(f.Name).
				SetUint(val)

		case reflect.Int, reflect.Int8,	reflect.Int16, 
		     reflect.Int32, reflect.Int64:
			val, err := strconv.Atoi(strVal)
			if err != nil {
				http.Error(w,
					"query parameter "+key+" must be an integer",
					http.StatusBadRequest,
				)
				return v, false
			}
			reflect.ValueOf(&v).Elem().
				FieldByName(f.Name).
				SetInt(int64(val))

		case reflect.Bool:
			val, err := strconv.ParseBool(strVal)
			if err != nil {
				http.Error(w,
					"query parameter "+key+" must be Boolean",
					http.StatusBadRequest,
				)
				return v, false
			}
			reflect.ValueOf(&v).Elem().
				FieldByName(f.Name).
				SetBool(val)
				
		default:
			panic(
				"dto: type parameter passed to ParseQuery includes "+
				"unsupported field type " + f.Name,
			)
		}
	}
	
	if err := v.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return v, false
	}

	return v, true
}

// Returns the bytes for a request body except it is decoded with gzip and/or
// deflate as instructed by the Content-Encoding header. The Content-Type 
// header is checked and the content is sniffed to verify the correct file 
// type. If you do not need to check the content type, pass an empty string. 
// This function closes the request body.
//
// Since this function maintains the bytes in memory, it should be used only
// when the request body is not excessively large. Consider using 
// [ParseBodyFile] in those cases. 
//
// The second return value will be false if there was an error. In that case,
// this function will have already sent an appropriate response to the client
// and the request body will be closed. Possible response statuses include 
// 400 Bad Request, 415 Unsupported Media Type, 413 Request Entity Too Large,
// and 500 Internal Server Error.
func ParseBodyBytes(
	w http.ResponseWriter, 
	r *http.Request, 
	contentType string,
	maxSize uint64,
) (io.ReadSeekCloser, bool) {
		
	checkContentType := contentType != ""

	if checkContentType && r.Header.Get("Content-Type") != "application/json" {
		http.Error(w,
			"expected Content-Type header to " + contentType,
			http.StatusUnsupportedMediaType,
		)
		r.Body.Close()
		return nil, false
	}

	body, err := decode(r)
	if err != nil {
		switch err {
		case errMalformedContent:
			http.Error(w,
				"request body content was not encoded correctly",
				http.StatusBadRequest,
			)
			return nil, false
		case errUnsupportedContentEncoding:
			http.Error(w,
				"request Content-Encoding contains an unsupported "+
					"compression format "+
					"(supported formats are gzip and deflate)",
				http.StatusUnsupportedMediaType,
			)
			return nil, false
		default:
			panic("dto: decode returned an unexpected error")
		}
	}
	body = http.MaxBytesReader(w, body, int64(maxSize))
	defer body.Close()

	buf, err := io.ReadAll(body)
	if err != nil {
		m := &http.MaxBytesError{}
		if ok := errors.As(err, &m); ok {
			http.Error(w,
				"request body exceeds " + formatMemorySize(maxSize),
				http.StatusRequestEntityTooLarge,
			)
			return nil, false
		}
		http.Error(w,
			"unexpected error while reading request body",
			http.StatusInternalServerError,
		)
		return nil, false
	}

	if checkContentType && http.DetectContentType(buf) != contentType {
		http.Error(w,
			"request body does not match the given Content-Type",
			http.StatusUnsupportedMediaType,
		)
		return nil, false
	}

	return &rsc{ b: buf }, true
}

// A simple [io.ReadSeekCloser] implementation.
type rsc struct {
	idx int64
	b   []byte
}

var _ io.ReadSeekCloser = (*rsc)(nil)

func (r *rsc) Read(p []byte) (int, error) {
	n := min(len(p), len(r.b) - int(r.idx))
	if copied := copy(p, r.b[r.idx:n]); copied != n {
		panic("copy returned fewer bytes than expected")
	}
	r.idx += int64(n)
	return n, nil
}

func (r *rsc) Seek(offset int64, whence int) (int64, error) {

	var newIdx int64

	switch whence {
	case io.SeekStart:
		newIdx = offset
	case io.SeekEnd:
		newIdx = int64(len(r.b)) + offset
	case io.SeekCurrent:
		newIdx = r.idx + offset
	default:
		panic("dto: Seek was given an invalid whence argument")
	}

	if newIdx < 0 {
		return 0, errors.New("dto: Seek call sought a negative byte index")
	}

	r.idx = newIdx
	return newIdx, nil
}

func (r *rsc) Close() error {
	r.idx = 0
	r.b = nil
	return nil
}

// Equivalent to [ParseBodyBytes], except it uses a backing temporary file 
// instead of an in-memory byte buffer. It's essential that the Close method of
// the result be called.
func ParseBodyFile(
	w http.ResponseWriter, 
	r *http.Request, 
	contentType string,
	maxSize uint64,
) (io.ReadSeekCloser, bool) {

	checkContentType := contentType != ""

	if checkContentType && r.Header.Get("Content-Type") != "application/json" {
		http.Error(w,
			"expected Content-Type header to " + contentType,
			http.StatusUnsupportedMediaType,
		)
		r.Body.Close()
		return nil, false
	}

	body, err := decode(r)
	if err != nil {
		switch err {
		case errMalformedContent:
			http.Error(w,
				"request body content was not encoded correctly",
				http.StatusBadRequest,
			)
			return nil, false
		case errUnsupportedContentEncoding:
			http.Error(w,
				"request Content-Encoding contains an unsupported "+
					"compression format "+
					"(supported formats are gzip and deflate)",
				http.StatusUnsupportedMediaType,
			)
			return nil, false
		default:
			panic("dto: decode returned an unexpected error")
		}
	}
	body = http.MaxBytesReader(w, body, int64(maxSize))
	defer body.Close()

	f, err := os.CreateTemp("", "omr_temp_*")
	if err != nil {
		http.Error(w,
			"unexpected error making system call",
			http.StatusInternalServerError,
		)
		return nil, false
	}

	if _, err := io.Copy(f, body); err != nil {
		m := &http.MaxBytesError{}
		if ok := errors.As(err, &m); ok {
			http.Error(w,
				"request body exceeds " + formatMemorySize(maxSize),
				http.StatusRequestEntityTooLarge,
			)
			return nil, false
		}
		http.Error(w,
			"unexpected error while reading request body",
			http.StatusInternalServerError,
		)
		return nil, false		
	}
	
	head := make([]byte, 512)
	f.Seek(0, io.SeekStart)
	f.Read(head)
	if checkContentType && http.DetectContentType(head) != contentType {
		http.Error(w,
			"request body does not match the given Content-Type",
			http.StatusUnsupportedMediaType,
		)
		return nil, false
	}

	f.Seek(0, io.SeekStart)
	return &rscFile{ File: f }, true
}

// A simple [io.ReadSeekCloser] implementation that uses a temporary file.
// Importantly, calling the Close method will also remove the file.
type rscFile struct {
	*os.File
}

func (r *rscFile) Close() error {
	r.File.Close()
	os.Remove(r.File.Name())
	return nil
}
