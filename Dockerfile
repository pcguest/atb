# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS web-builder
WORKDIR /src/web
ENV NEXT_TELEMETRY_DISABLED=1

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.7-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS go-builder
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

ARG TARGETOS
ARG TARGETARCH
# Required for OCI label accuracy. Publish workflows pass the release git tag.
ARG ATB_VERSION
RUN printf '%s\n' "${ATB_VERSION}" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || \
  (echo "ATB_VERSION must be a non-empty release version (e.g. --build-arg ATB_VERSION=v1.15.3)" >&2; exit 1)
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -ldflags='-s -w' -o /out/atb ./cmd/atb

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS runtime
WORKDIR /app
ARG ATB_VERSION
LABEL org.opencontainers.image.version="${ATB_VERSION}"

COPY --from=go-builder /out/atb /app/atb
COPY --from=web-builder /src/web/out /app/web/out
COPY LICENSE THIRD_PARTY_NOTICES /licenses/atb/

EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/atb"]
