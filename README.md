
<br />
<div width="100%" align="center">
    <img width="160px" alt="aperture logo" src="./pages/img/logo.gif" />
</div>
<br />
<br />

Aperture OMR is a high-performance [Optical Mark Recognition](https://en.wikipedia.org/wiki/Optical_mark_recognition) system which analyzes scanned bubble sheet exams to produce marks. Aperture exposes its functionality through a REST API served over HTTP and is meant to be integrated with other apps.

## Benchmarks

[Placeholder.]

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

## Environment Variables

The following variables can be used to configure the OMR:

```conf
OMR_GLOBAL_KEY="some secret"
OMR_ADMIN_KEY="some super secret"
```

The `OMR_GLOBAL_KEY` is a key that locks all endpoints for the API. Upon receiving a request, the OMR will look for an `OMR-API-Key` header and check if it matches this variable. Leaving the variable un-set will mean that anyone with the IP address (and port number) can access the OMR.

The `OMR_ADMIN_KEY` is used to restrict specific endpoints (e.g., the endpoint to delete all exams older than a certain date). The OMR looks for this key in the `OMR-Admin-Key` header of incoming requests. If you don't set this variable, it will fall back to a default (which will be logged upon server startup).

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

## Acknowledgements

This service was originally developed as a part of a capstone project at UBC. During the development, [Kaden Harris](https://github.com/KadenHarris) implemented the original `internal/scanner` and `internal/marker` packages. The rest of my group also indirectly contributed by reviewing my PRs and doing manual testing. Thanks gang.
