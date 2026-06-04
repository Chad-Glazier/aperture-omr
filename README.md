# OMR Service

This directory contains the source code for the OMR service.

## Setup

The [Dockerfile](./Dockerfile) can be used to start the OMR service. 

```sh
# from the `app/omr` directory...
docker build --tag omr .
docker run --rm omr:latest
```

Since we use OpenCV as a dependency, the program takes a while to build. Once I figure out a faster local development environment I'll write instructions for the setup.

>Note: The Go version, set in `go.mod` is fixed to match the version of the GoCV image we rely on. Do not change it.

## File Structure

In keeping with Go conventions, the top-level directories are as follows:
- `cmd` contains "main" packages that serve as entrypoints to the program.
- `internal` contains packages that are used internally. This will be most packages.
- `configs` includes configuration files.
- `scripts` includes programs that are not directly used by the main application. This could include build scripts, code generation tools, etc.
- `test` includes integration tests (unit tests should be included with the package that they test on).
- `docs` includes documentation.
- `assets` includes non-code resources, such as images.

For more info about these standards, refer to [this document](https://github.com/golang-standards/project-layout).
