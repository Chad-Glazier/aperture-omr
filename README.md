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
docker run --rm -p 3000:3000 omr_server
```

This will log a URL to the documentation on startup. Follow that link to view details about the endpoints.

If you want the container to persist and maintain its stored data (volume), instead run:

```sh
docker run -p 3000:3000 -v omr-data:/app/data omr_server
```

>The Go version set in `go.mod` is fixed to match the version of the GoCV image we rely on. Do not change it.

## Setting Up a Local Environment

In order to run the project outside of Docker, you must first ensure that you have [OpenCV](https://gocv.io/getting-started/) and [Go](https://go.dev/doc/install) installed. If you have those, you should be able to run the program:

```sh
go run .
```

This should print a help message that describes the subcommands for the program. 

If you get an error that mentions missing C/C++ objects, it's likely that GoCV isn't seeing your OpenCV installation. Refer to [their documentation](https://gocv.io/getting-started/) to correct this.

## File Structure

In keeping with Go conventions, the top-level directories are as follows:
- [`cmd`](./cmd) contains the [Cobra](https://github.com/spf13/cobra) commands that serve as entrypoints to the program. The [`rootCmd`](./cmd/root.go) command is used by the top-level [`main.go`](./main.go) program.
- [`api`](./api) includes the specification(s) for the HTTP server. It does not contain the actual server code, just the specification.
- [`internal`](./internal) contains packages that are used internally. This will be most of the project.
  - [`database`](./internal/database/) contains all database interactions. Most of the queries are generated with sqlc (I explain this more [here](./internal/database/sqlc/README.md)).
  - [`httpserver`](./internal/httpserver/) contains the handler functions and middleware that make up the HTTP API for the service. We are just using the standard [`net/http`](https://pkg.go.dev/net/http) library since the API should be relatively simple.
    - [`dto`](./internal/httpserver/dto/) contains the data transfer objects (DTOs) for the server. Any complex object that will be sent or received from the server is put there, along with relevant deserialization and validation functions.
    - [`handler`](./internal/httpserver/handler/) includes the bulk of the HTTP server's logic.
    - [`resources`](./internal/httpserver/resources/) is a package that is solely concerned with implementing the [`ServerResources`](./internal/httpserver/handler/resources.go) interface. The package is very minimal and only exists separately to avoid cyclical dependencies.
  - [`fs`](./internal/fs/) exposes a simple interface for file storage, particularly images. Internally, it currently has two implementations; one wraps the local file system (suitable for testing) and the other wraps an S3 client.
- [`config`](./config) wraps configuration data.
  - Conventionally, this folder would simply include `.env` files and stuff. However in this project we define it as an actual Go package that's responsible for wrapping configuration variables and ensuring that they're all set, as opposed to littering the codebase with `os.Getenv()` calls and error checks. This also allows us to set environment variables based on the runtime mode (development, production, or test) and validate them when the program starts, instead of postponing potential runtime errors.
  - We expect environment variables to be set outside of the application (e.g., by a Docker configuration or in the command line).

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
- [AWS's SDK](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2) and some of its subpackages are used to manage an S3 client. It's only used in the [fs](./internal/fs) package

## Development Notes

### Room for Improvement

The following is a list of known improvements that can be made to the system.
- The current storage system relies on converting images to/from `image.Image` objects repeatedly for the convenience of a common interface. In multiple areas, this is likely unnecessary and instead we can just use `io.Writer`/`io.Reader` to minimize the number of times we load the full image into memory.
  - On a related note, the processing pipeline should probably universally use JPEGs to save disk space and (potentially) speed. We can use magic bytes to confirm that an incoming file is a JPEG and then, rather than decoding the whole image, just write it straight to storage. This method might be tricky with S3 storage but would be trivial to implement for the local filesystem.
- The database layer currently only uses SQLite3 as an interim approach. Given that we are using sqlc for code generation, it would be very straightforward to implement a version for Postgres.
  - More generally, the storage should be slightly refactored so that it's easier to swap between using external services vs a single container. The configuration should be exposed via the command line or environment variables.
- The API spec assumes the domain and port are `localhost:3000`. However, we could rewrite it as a template and have those values populated at runtime.
- The service, in its current form, manages its own database and file storage. If this is the approach we want to stick with, we should include endpoints to manage that data (e.g., delete old data, compress unused images periodically, etc). It would also be feasible to make a small web UI to expose that functionality.
- Anchor images are stored as properly encoded image files right now, but we only really need them as `gocv.Mat` objects. There's certainly a more direct way to serialize that data than the current approach of converting them to/from image files.
- Logs are currently printed by directly calling `slog` functions that write to stdout. We should change this so that the logger is on the `ServerResources` struct instead. This way we could conveniently also log errors to a database, a log file, or whatever else.
- The whole `config` package needs to be refactored or removed entirely. I don't know why I thought it would be a good idea.
- The tests in `httpserver/handlers` cover the normal path, but they don't exhaust all of the bad inputs. We could make the test suite a little more comprehensive.
  - Those tests also have a good amount of copy+pasted code, since some handlers depend on others for setup. This can probably be refactored.
