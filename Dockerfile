# The official GoCV image
FROM ghcr.io/hybridgroup/opencv:4.13.0

ENV GOPATH=/go

COPY . /app

WORKDIR /app
RUN go build -o /build/omr .

ENTRYPOINT ["/build/omr"]
CMD ["serve"]