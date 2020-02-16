FROM golang:1.13.8-buster AS builder
WORKDIR /src/app

# Build deps
RUN apt-get update; apt-get install -y automake build-essential gobject-introspection gtk-doc-tools libglib2.0-dev libjpeg-dev libwebp-dev libtiff5-dev libexif-dev libgsf-1-dev liblcms2-dev libxml2-dev swig libmagickcore-dev curl
RUN apt-get install -y lsb-release libvips-dev

COPY . /src/app

# Install our dependencies
RUN go mod vendor
RUN make

FROM debian:10
ENTRYPOINT ["/usr/bin/identifier"]

# Install runtime dependencies
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y libvips ca-certificates && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=builder /src/app/bin/identifier /usr/bin/identifier
