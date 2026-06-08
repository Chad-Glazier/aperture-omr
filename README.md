# OMR Service

This directory contains the source code for the OMR service.

## Setup

The [Dockerfile](./Dockerfile) can be used to start the OMR service. 

```sh
# from the `app/omr` directory...
docker build --tag omr .
docker run --rm omr:latest
```

Since we use OpenCV as a dependency, the program takes a while to build. If you want a faster development environment, you can run the project outside of a container but you'll first need to install [OpenCV](https://gocv.io/getting-started/) and the [Go compiler](https://go.dev/doc/install).

>Note: The Go version, set in `go.mod` is fixed to match the version of the GoCV image we rely on. Do not change it.

## File Structure

In keeping with Go conventions, the top-level directories are as follows:
- `cmd` contains "main" packages that serve as entrypoints to the program.
- `internal` contains packages that are used internally. This will be most of the project.
- `config` wraps configuration data.
  - This package is responsible for exposing configuration variables and ensuring that they're all set, as opposed to littering the codebase with `os.Getenv()` calls and error checks. 
- `api` includes the specification(s) for the HTTP server. It does not contain the actual server code, just the specification.

For more info about these standards, refer to [this document](https://github.com/golang-standards/project-layout).

### Package Overview

- [`internal/data`](./internal/data/) contains all database interactions. Most of the queries are generated with sqlc (I explain this more [here](./internal/data/sqlc/README.md)).
- [`internal/http`](./internal/http/) contains the handler functions and middleware that make up the HTTP API for the service.
