# syntax=docker/dockerfile:1

FROM golang:1.22 AS build-env
WORKDIR /build
RUN --mount=type=bind,source=.,target=.,readonly \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o /app .

CMD ["/app"]

