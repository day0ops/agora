# Build from the repo root: docker build -f brand-registry/Dockerfile .
# --platform=$BUILDPLATFORM (native, not emulated) plus a hardcoded GOARCH
# matches console/Dockerfile's own fix for the same issue: this project only
# ever deploys to amd64 node pools, and a native arm64 build host (e.g. Apple
# Silicon) would otherwise cross the target image with its own host arch,
# producing an unbootable "exec format error" binary on the real cluster.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
WORKDIR /src
COPY brand-registry/ ./brand-registry/
WORKDIR /src/brand-registry
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/app /app
EXPOSE 8090
ENTRYPOINT ["/app"]
