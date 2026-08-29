# awapp — build a headless image for testing/CI.
# Run headless:  docker run --rm awapp -list-config
# Run in a TTY:  docker run --rm -it awapp -city "Fortaleza,BR"
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags 2>/dev/null || echo v1.0.5)" -o /out/awapp .

FROM alpine:3.20
RUN apk add --no-cache tzdata
COPY --from=build /out/awapp /usr/local/bin/awapp
ENTRYPOINT ["awapp"]
CMD ["-list-config"]
