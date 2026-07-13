# The official GoCV image
FROM ghcr.io/hybridgroup/opencv:4.13.0

ENV GOPATH=/go

# 
# Install the ImageMagick dependency.
# 

RUN apt update
RUN apt install -y \
    imagemagick=8:6.9.11.60+dfsg-1.6+deb12u11 \
    libmagickwand-dev=8:6.9.11.60+dfsg-1.6+deb12u11 \
    ghostscript

# Update permissions policy so that ImageMagick is cool
RUN sed -i \
    's/rights="none"/rights="read|write"/g' \
    /etc/ImageMagick-6/policy.xml


#
# Build the app
#

COPY . /app

WORKDIR /app
RUN go build -o /build/omr .

#
# Run the app
#

ENTRYPOINT ["/build/omr"]
CMD ["serve"]
