# syntax=docker/dockerfile:1.7

FROM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS web-builder
WORKDIR /src/web
ENV NEXT_TELEMETRY_DISABLED=1

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go-builder
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
# Required for OCI label accuracy. Publish workflows pass the release git tag.
ARG ATB_VERSION
RUN test -n "${ATB_VERSION}" || (echo "ATB_VERSION build-arg is required (e.g. --build-arg ATB_VERSION=v1.16.0)" >&2; exit 1)
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -ldflags='-s -w' -o /out/atb ./cmd/atb

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS runtime
WORKDIR /app
ARG ATB_VERSION
LABEL org.opencontainers.image.version="${ATB_VERSION}"

COPY --from=go-builder /out/atb /app/atb
COPY --from=web-builder /src/web/out /app/web/out

EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/atb"]
