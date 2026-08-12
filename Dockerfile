# Build the collector, then ship it on a tiny static base.
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/geopilot ./cmd/collector

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/geopilot /geopilot
EXPOSE 8080
ENTRYPOINT ["/geopilot"]
