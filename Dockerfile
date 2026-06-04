# The official GoCV image
FROM ghcr.io/hybridgroup/opencv:4.13.0

ENV GOPATH=/go

COPY . /go/src/ubc/team15

WORKDIR /go/src/ubc/team15
RUN go build -o /build/version ./cmd/version/

CMD ["/build/version"]
