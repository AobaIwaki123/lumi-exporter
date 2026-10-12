# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -trimpath -ldflags="-w -s" \
    -o /out/lumi-exporter ./cmd/lumi-exporter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/lumi-exporter /lumi-exporter
COPY config/sources.json /config/sources.json

EXPOSE 8081
USER 65532:65532
ENTRYPOINT ["/lumi-exporter", "-serve", "-registry", "/config/sources.json"]
