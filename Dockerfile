FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO/SQLite
RUN apk add --no-cache build-base sqlite-dev gcc musl-dev

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Enable CGO with system sqlite3 (fast dynamic link, avoids compiling 8MB C source)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -tags libsqlite3 -ldflags="-w -s" -o whatsmiau main.go

FROM alpine:latest

RUN apk update && apk add --no-cache ffmpeg mailcap sqlite-libs

WORKDIR /app

COPY --from=builder /app/whatsmiau /app/whatsmiau

RUN mkdir /app/data && chmod 777 -R /app/data

EXPOSE 8080 8081

ENTRYPOINT ["./whatsmiau"]

