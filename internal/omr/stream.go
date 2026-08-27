package omr

import (
	"context"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

type PageSet interface {
	Pages() []Mat
	Error() error
}

type result struct {
	pages []Mat
	err   error
}

func (r result) Pages() []Mat {
	return r.pages
}

func (r result) Error() error {
	return r.err
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
		sem = semaphore.NewWeighted(int64(parallelism))
		out = make(chan PageSet, parallelism)
	)

	go func() {
		for pageSet := range in {

			if err := pageSet.Error(); err != nil {
				out <- result{
					pages: nil,
					err:   err,
				}
				continue
			}

			pages := pageSet.Pages()

			// This method only errs if (a) we attempt to acquire a greater
			// weight than was assigned to the semaphore, or (b) the context
			// errs. Neither of those conditions are possible here.
			sem.Acquire(context.Background(), 1)
			go func() {
				defer sem.Release(1)

				err := fn(pages)
				if err != nil {
					CloseAll(pages)
					out <- result{
						pages: nil,
						err:   err,
					}
					return
				}

				out <- result{
					pages: pages,
					err:   nil,
				}	
			}()
		}
		
		sem.Acquire(context.Background(), int64(parallelism))
		close(out)
	}()

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

			for pageSet := range pageStream {
				out <- preprocessSet(template, pageSet)
			}

			if threads.Add(-1) == 0 {
				close(out)
			}
		}()
	}

	out <- firstResult
	return out, nil
}

func preprocessSet(template PreprocessingTemplate, set PageSet) PageSet {
	if err := set.Error(); err != nil {
		return result{
			pages: nil,
			err:   err,
		}
	}

	pages := set.Pages()
	defer CloseAll(pages)

	preprocessed, err := Preprocess(template, pages)
	return result{
		pages: preprocessed,
		err:   err,
	}
}
