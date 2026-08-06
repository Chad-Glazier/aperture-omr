
<div width="100%" align="center">
    <img width="160px" alt="aperture logo" src="./pages/img/logo.gif" />
</div>

<!-- # Aperture OMR -->

[description placeholder]

## Setup

The docker image can be built from the top-level dockerfile. 

```sh
# from the `app/omr` directory...
docker build --tag omr_server .
```

Since we use OpenCV as a dependency the program takes a while to build. Refer to the [this section](#setting-up-a-local-environment) if you need a faster build for development. 

Once the image is built, you can run the app via docker:

```sh
docker run --rm -p 3000:3000 --name omr omr_server
```

This will log a URL to the documentation on startup. Follow that link to view details about the endpoints.

If you want the container to persist and maintain its stored data (volume), instead run:

```sh
docker run -p 3000:3000 -v omr-data:/app/data --name omr omr_server
```

## Testing

If you want to run tests from within a container, run

```sh
docker exec -t <container-name> sh -c "go test ./..."
```

To execute coverage tests, you can run a script from inside the container:

```sh
docker exec -t <container-name> sh -c "./scripts/coverage.sh"
```

## Setting Up a Local Environment

>The Go version set in `go.mod` is fixed to match the version of the GoCV image we rely on. Do not change it.

In order to run the project outside of Docker, you must first ensure that you have [OpenCV](https://gocv.io/getting-started/) and [Go](https://go.dev/doc/install) installed. If you have those, you should be able to run the program:

```sh
go run .
```

This should print a help message that describes the subcommands for the program. 

If you get an error that mentions missing C/C++ objects, it's likely that GoCV isn't seeing your OpenCV installation. Refer to [their documentation](https://gocv.io/getting-started/) to correct this.

## Dependencies

The following is a list of dependencies. You can also refer to the [go.mod](./go.mod) file.

C dependencies:
- [OpenCV](https://opencv.org/) is used for computer vision stuff.

Go packages (excluding the standard library):
- [GoCV](gocv.io/x/gocv) provides Go bindings for OpenCV.
- [SQLite3](modernc.org/sqlite) is included as our local database.
- [rs/cors](https://pkg.go.dev/github.com/rs/cors) is used to configure CORS. It's thinly wrapped in [cors.go](./internal/server/middleware/cors.go). 
- [Cobra](https://cobra.dev/) is used to set up the command-line interface. It's only used in the [cmd](./cmd) package.
- [lz4](https://github.com/pierrec/lz4/v4) is used to compress OpenCV matrices when we save them to persistent storage.
- [pdfcpu](https://github.com/pdfcpu/pdfcpu) is used to preprocess PDFs.
- [fitz](https://github.com/gen2brain/go-fitz) is used to render PDFs.
- [gopsutil](https://github.com/shirou/gopsutil/v4) is used to make OS-agnostic syscalls for monitoring resource usage.
- [Google's UUID package](https://github.com/google/uuid) is used for generating UUIDs.

Developer dependencies (not required for runtime):
- [sqlc](https://sqlc.dev/) is used to generate Go functions from SQL queries (read more [here](./internal/database/sqlc/README.md)).

## Acknowledgements

This service was originally developed as a part of a capstone project at UBC. During the development, [Kaden Harris](https://github.com/KadenHarris) implemented the original `internal/scanner` and `internal/marker` packages. The rest of my group also indirectly contributed by reviewing my PRs and doing manual testing. Thanks gang.
