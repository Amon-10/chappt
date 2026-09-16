ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY cmd/server ./cmd/server
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/chappt-server \
    ./cmd/server

FROM scratch

COPY --from=build --chown=65532:65532 /out/chappt-server /chappt-server

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/chappt-server"]
