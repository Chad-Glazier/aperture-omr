# OMR Service

This directory contains the source code for the OMR service.

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

If you want to run tests in a running container (named `omr` in this case), run

```sh
docker exec -t omr sh -c "go test ./..."
```

>The Go version set in `go.mod` is fixed to match the version of the GoCV image we rely on. Do not change it.

## Setting Up a Local Environment

In order to run the project outside of Docker, you must first ensure that you have [OpenCV](https://gocv.io/getting-started/), [ghostscript](https://www.ghostscript.com/), [ImageMagick6](https://legacy.imagemagick.org/#gsc.tab=0), and [Go](https://go.dev/doc/install) installed. If you have those, you should be able to run the program:

```sh
go run .
```

This should print a help message that describes the subcommands for the program. 

If you get an error that mentions missing C/C++ objects, it's likely that GoCV isn't seeing your OpenCV installation. Refer to [their documentation](https://gocv.io/getting-started/) to correct this.

### Setting Up `pkg-config`

Our source code is using CGo now. In [`magick.go`](./internal/pdf/magick.go) you'll see the following directives:

```go
// #cgo pkg-config: MagickWand
// #cgo pkg-config: opencv4
```

In order for the build to work, Go needs to be able to call `pkg-config` with `MagickWand` and `opencv4` packages. If you don't have `pkg-config` installed, you can install it for your OS:
- On Linux, run `sudo apt install pkg-config`
- On MacOS, run `brew install pkgconf`
- On Windows, download a release from [this website](https://sourceforge.net/projects/pkgconfiglite/files/). Extract the archive and then put the `pkg-config.exe` file somewhere where you'll remember it.

Next, ensure that Go can find the `pkg-config` executable. To do this you must set the `PKG_CONFIG` environment variable to the absolute path of the executable. If you're not sure where it is on Linux or MacOS, use `which pkg-config`.

The last step is to ensure that `pkg-config` can see the configuration files for OpenCV and ImageMagick. Normal installations of those packages on Linux will come with `.pc` files out of the box, but on Windows you might need to make your own. I've written mine down below.

`/path/to/stuff/MagickWand.pc`:
```
prefix=E:/Tooling/imagemagick6/ImageMagick-6.9.13-Q16-HDRI
exec_prefix=${prefix}
libdir=${exec_prefix}/lib
includedir=${prefix}/include

Name: MagickWand
Description: ImageMagick 6 image processing library
Version: 6.9.13
Requires:
Libs: -L${libdir} -lCORE_RL_wand_ -lCORE_RL_magick_
Cflags: -I${includedir}
```

`/path/to/stuff/opencv4.pc` (I'm using MingW on Windows):
```
prefix=E:/Tooling/opencv/build
exec_prefix=${prefix}
includedir=${prefix}/include
libdir=${prefix}/x64/mingw/lib

Name: OpenCV
Description: Open Source Computer Vision Library
Version: 4.13.0

Cflags: -I${includedir}
Libs: -L${libdir} \
    -lopencv_core4130 \
    -lopencv_imgproc4130 \
    -lopencv_imgcodecs4130 \
    -lopencv_highgui4130 \
    -lopencv_videoio4130 \
    -lopencv_features2d4130 \
    -lopencv_calib3d4130 \
    -lopencv_objdetect4130 \
    -lopencv_dnn4130
```

You'll need to adjust these files so that they actually point to your installations, but you get the gist. To make sure that `pkg-config` and Go can see these files, make sure you set the `PKG_CONFIG_PATH` variable to the directory where you keep them.

## File Structure

In keeping with Go conventions, the top-level directories are as follows:
- [`cmd`](./cmd) contains the [Cobra](https://github.com/spf13/cobra) commands that serve as entrypoints to the program. The [`rootCmd`](./cmd/root.go) command is used by the top-level [`main.go`](./main.go) program.
- [`api`](./api) includes the specification(s) for the HTTP server. It does not contain the actual server code, just the specification.
- [`internal`](./internal) contains packages that are used internally. This will be most of the project.
  - [`database`](./internal/database/) contains all database interactions. Most of the queries are generated with sqlc (I explain this more [here](./internal/database/sqlc/README.md)).
  - [`httpserver`](./internal/httpserver/) contains the handler functions and middleware that make up the HTTP API for the service. We are just using the standard [`net/http`](https://pkg.go.dev/net/http) library since the API should be relatively simple.
    - [`dto`](./internal/httpserver/dto/) contains the data transfer objects (DTOs) for the server. Any complex object that will be sent or received from the server is put there, along with relevant deserialization and validation functions.
    - [`handler`](./internal/httpserver/handler/) includes the bulk of the HTTP server's logic.
  - [`fs`](./internal/fs/) exposes a simple interface for file storage, particularly images. Internally, it currently has two implementations; one wraps the local file system (suitable for testing) and the other wraps an S3 client.
  - [`pdf`](./internal/pdf) handles PDF rendering.

For more info about the top-level directory naming standards, refer to [this document](https://github.com/golang-standards/project-layout). This is not an "official" project setup, but it is a popular one.

## Dependencies

The following is a list of dependencies. You can also refer to the [go.mod](./go.mod) file.
- [OpenCV](https://opencv.org/) is used for computer vision stuff.
- [GoCV](gocv.io/x/gocv) provides Go bindings for OpenCV.
- [pgx](https://pkg.go.dev/github.com/jackc/pgx/v5) is the Postgres driver we use. It's used in the [database](./internal/database/) layer.
- [sqlite3](modernc.org/sqlite) is also included as an alternative local database.
- [rs/cors](https://pkg.go.dev/github.com/rs/cors) is used to configure CORS. It's thinly wrapped in [cors.go](./internal/httpserver/middleware/cors.go). 
- [Cobra](https://cobra.dev/) is used to set up the command-line interface. It's only used in the [cmd](./cmd) package.
- [sqlc](https://sqlc.dev/) is used to generate Go functions from SQL queries (read more [here](./internal/database/sqlc/README.md)). sqlc is strictly for code generation; it is not a runtime dependency.
- [lz4](https://github.com/pierrec/lz4/v4) is used to compress OpenCV matrices when we save them to persistent storage.
- [ImageMagick6](https://legacy.imagemagick.org/#gsc.tab=0) is used to convert PDFs to images. 
- [ghostscript](https://www.ghostscript.com/) is a required dependency for ImageMagick to render PDFs. 
  - Since we're only using ImageMagick to render PDFs (at the time of writing), we could remove ImageMagick and just use the ghostscript API directly. However, the API is OS-specific and the overhead from ImageMagick is negligible. It's just not worth the effort at this time.

## Development Notes

### Room for Improvement

The following is a list of known improvements that can be made to the system.
- The database layer currently only uses SQLite3 as an interim approach. Given that we are using sqlc for code generation, it would be very straightforward to implement a version for Postgres.
  - More generally, the storage should be slightly refactored so that it's easier to swap between using external services vs a single container. The configuration should be exposed via the command line or environment variables.
- The service, in its current form, manages its own database and file storage. If this is the approach we want to stick with, we should include endpoints to manage that data (e.g., delete old data, compress unused images periodically, etc). It would also be feasible to make a small web UI to expose that functionality.
- **More testing**