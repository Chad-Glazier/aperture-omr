package omr

import (
	"sync/atomic"
)

type PageSet interface {
	Pages() []Mat
	Error() error
	Metadata() map[string]string
}

type result struct {
	pages []Mat
	err   error
	meta  map[string]string
}

func (r result) Pages() []Mat {
	return r.pages
}

func (r result) Error() error {
	return r.err
}

func (r result) Metadata() map[string]string {
	return r.meta
}

// Executes a given callback function for each received page set. The output
// channel will just pass down the same page set unless the callback returned
// an error. In that case, the incoming page set will be closed and the error
// will be propagated through the output.
//
// The parallelism argument determines how many concurrent instances of the
// callback can run at once. If set to zero, it will default to 1.
func ForEach(
	in <-chan PageSet,
	parallelism uint,
	fn func([]Mat) error,
) <-chan PageSet {
	parallelism = max(1, parallelism)

	var (
		out     = make(chan PageSet, parallelism)
		threads atomic.Int32
	)
	for range parallelism {
		go func() {
			threads.Add(1)
			defer func() {
				if threads.Add(-1) == 0 {
					close(out)
				}
			}()

			for pageSet := range in {
				
				if err := pageSet.Error(); err != nil {
					out <- pageSet
					continue
				}

				pages := pageSet.Pages()
				err := fn(pages)
				if err != nil {
					CloseAll(pages)
					out <- result{
						err:  err,
						meta: pageSet.Metadata(),
					}
					return
				}

				out <- result{
					pages: pages,
					meta:  pageSet.Metadata(),
				}
			}
		}()
	}

	return out
}

// Preprocesses page sets as they are received from the given channel, sending
// the results through the output channel. Importantly, all received matrices
// will be closed by this function. Each input slice must have exactly as many
// pages as are expected by the template. It is also assumed that all matrices
// are roughly the same size.
//
// The parallelism argument determines how many concurrent preprocessing
// operations will be executed at a time. If set to zero, it defaults to 1.
//
// If a given page set has a non-nil error, this stream will propagate the
// error to its respective output. It will not stop the stream.
//
// If an error is returned, it will be [ErrCouldNotCalibrate] or [ErrOpenCV].
func PreprocessStream(
	template PreprocessingTemplate,
	parallelism uint,
	pageStream <-chan PageSet,
) (<-chan PageSet, error) {
	if parallelism == 0 {
		parallelism = 1
	}

	//
	// We need to read the first set in order to scale the preprocessing
	// template for the operation.
	//

	first := <-pageStream
	firstPages := first.Pages()
	defer CloseAll(firstPages)

	if len(firstPages) == 0 {
		return nil, ErrCouldNotCalibrate
	}

	template, err := ScalePreprocessingTemplate(
		FitMethodContain,
		template,
		firstPages[0].Width(), firstPages[0].Height(),
	)
	if err != nil {
		return nil, ErrOpenCV
	}
	defer template.Close()

	firstResult := preprocessSet(template, first)

	//
	// Now, we can start up the threads and return the channel.
	//

	var (
		out     = make(chan PageSet, parallelism)
		threads atomic.Int32
	)
	for range parallelism {
		go func() {
			threads.Add(1)
			defer func() {
				if threads.Add(-1) == 0 {
					close(out)
				}
			}()

			for pageSet := range pageStream {
				out <- preprocessSet(template, pageSet)
			}
		}()
	}

	out <- firstResult
	return out, nil
}

func preprocessSet(template PreprocessingTemplate, set PageSet) PageSet {
	if err := set.Error(); err != nil {
		return set
	}

	pages := set.Pages()
	defer CloseAll(pages)

	preprocessed, err := Preprocess(template, pages)
	return result{
		pages: preprocessed,
		err:   err,
		meta:  set.Metadata(),
	}
}

// Marks page sets as they are received from the given channel, sending the
// results through the output channel. Importantly, all received matrices
// will be closed by this function. Each input slice must have exactly as many
// pages as are expected by the template, the matrices must be [MatTypeGray],
// and they must have a similar aspect ratio to the template.
//
// The parallelism argument determines the buffer size of the returned channel,
// not the number of threads. Marking is too fast to justify the overhead of
// real parallelism.
//
// If a given page set has a non-nil error, this stream will propagate the
// error to its respective output. It will not stop the stream.
func MarkStream(
	template MarkTemplate,
	parallelism uint,
	pageStream <-chan PageSet,
) <-chan MarkStreamResult {

	out := make(chan MarkStreamResult, parallelism)

	for pageSet := range pageStream {
		if err := pageSet.Error(); err != nil {
			out <- MarkStreamResult{
				Error:    err,
				Metadata: pageSet.Metadata(),
			}
			continue
		}

		pages := pageSet.Pages()
		defer CloseAll(pages)

		marks, err := Mark(template, pages)
		if err != nil {
			out <- MarkStreamResult{
				Error:    err,
				Metadata: pageSet.Metadata(),
			}
			continue
		}

		out <- MarkStreamResult{
			Marks:    marks,
			Metadata: pageSet.Metadata(),
		}
	}

	return out
}

type MarkStreamResult struct {
	Metadata map[string]string
	Marks    MarkResult
	Error    error
}
