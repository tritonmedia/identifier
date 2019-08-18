FROM golang:1.12.8-alpine3.10 AS builder
WORKDIR /src/app
COPY . /src/app

# Build deps
RUN apk add --no-cache git make bash
ENTRYPOINT ["/usr/bin/identifier"]

# Install our dependencies
RUN go mod vendor
RUN make CGO_ENABLED=0

FROM alpine:3.10
RUN apk add --no-cache ca-certificates
COPY --from=builder /src/app/bin/identifier /usr/bin/identifier