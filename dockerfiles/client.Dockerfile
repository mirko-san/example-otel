# syntax=docker/dockerfile:1

ARG GO_VERSION=1.22.5

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS base

WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

FROM --platform=$BUILDPLATFORM base AS build

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/client ./cmd/client

FROM debian:bookworm-slim AS final

COPY --from=build /bin/client /client

CMD [ "/client" ]
