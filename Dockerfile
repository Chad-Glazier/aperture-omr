# The official GoCV image
FROM ghcr.io/hybridgroup/opencv:4.13.0

ENV GOPATH=/go

COPY . /go/src/ubco-team15/omr

WORKDIR /go/src/ubco-team15/omr
RUN go build -o /build/omr .

CMD ["/build/omr", "serve"]
