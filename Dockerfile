FROM golang:1.17-alpine as build
WORKDIR /app

# Restore modules - Start
COPY go.mod ./
COPY go.sum ./
RUN go mod download
# Restore modules - End

COPY *.go ./
COPY internal/ internal/
COPY data/ data/
RUN go build -ldflags="-s -w" -o /goaurrpc

FROM alpine:3.15
WORKDIR /
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "1000" \
    "nonroot"
USER nonroot:nonroot

COPY sample.conf /sample.conf

COPY --from=build /goaurrpc /goaurrpc

ENTRYPOINT ["/goaurrpc", "-c", "/sample.conf"]
