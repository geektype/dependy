FROM golang:1.23.1 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod/ --mount=type=cache,target=/root/.cache/go-build \
    make bin

FROM scratch AS binary
COPY --from=build /src/bin /
