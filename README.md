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
- [`cmd`](./cmd) contains "main" packages that serve as entrypoints to the program.
- [`api`](./api) includes the specification(s) for the HTTP server. It does not contain the actual server code, just the specification.
- [`internal`](./internal) contains packages that are used internally. This will be most of the project.
  - [`db`](./internal/data/) contains all database interactions. Most of the queries are generated with sqlc (I explain this more [here](./internal/db/sqlc/README.md)).
  - [`http`](./internal/http/) contains the handler functions and middleware that make up the HTTP API for the service. We are just using the standard [`net/http`](https://pkg.go.dev/net/http) library since the API should be relatively simple.
- [`config`](./config) wraps configuration data.
  - Conventionally, this folder would simply include `.env` files and stuff. However, in this project we define it as an actual Go package that is responsible for exposing configuration variables and ensuring that they're all set, as opposed to littering the codebase with `os.Getenv()` calls and error checks. This is *not* conventional, but it seems nice for now.

For more info about these directory naming standards, refer to [this document](https://github.com/golang-standards/project-layout). This is not an "official" project setup, but it is a popular one.
