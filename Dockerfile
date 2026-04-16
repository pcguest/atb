# syntax=docker/dockerfile:1.7

FROM node:20-alpine AS web-builder
WORKDIR /src/web
ENV NEXT_TELEMETRY_DISABLED=1

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.24.13-alpine AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY uiembed.go trust_embed.go docs_embed.go ./
COPY docs ./docs
COPY schemas ./schemas
COPY SECURITY.md ./
COPY test/golden/golden_test.go ./test/golden/golden_test.go
COPY sdk/python/tests/test_properties.py ./sdk/python/tests/test_properties.py
COPY --from=web-builder /src/web/out ./web/out

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG ATB_VERSION=v1.4.0
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -ldflags='-s -w' -o /out/atb ./cmd/atb

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
ARG ATB_VERSION=v1.4.0
LABEL org.opencontainers.image.version="${ATB_VERSION}"

COPY --from=go-builder /out/atb /app/atb
COPY --from=web-builder /src/web/out /app/web/out

EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/atb"]
