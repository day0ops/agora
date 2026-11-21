# web and builder run with --platform=$BUILDPLATFORM (native, not emulated)
# since their RUN steps only need to produce platform-independent JS or
# cross-compiled Go output, not execute as the target arch. This also avoids
# a reproducible QEMU/Rosetta crash in esbuild's own Go binary when the vite
# build stage is emulated (e.g. building a linux/amd64 image on Apple
# Silicon). GOARCH is hardcoded rather than read from $TARGETARCH since this
# project only ever deploys to GKE's standard amd64 node pools, and some
# builders (e.g. podman's classic, non-buildx builder) don't reliably
# populate that automatic build arg.
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /web
COPY console/web/package.json console/web/package-lock.json* ./
RUN npm ci
COPY console/web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
ARG VERSION=dev
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY console/ ./console/
COPY --from=web /web/dist ./console/web/dist
WORKDIR /src/console
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o /out/app .

# kubectl-ate is the substrate CLI the backend shells out to for live actor
# state (see console/source.go); it is a statically-linked Go binary itself,
# so it runs fine under distroless static-debian12 below. v1.0.0 matches this
# repo's own pin (agentic-field-kit's addons/kagent/index.js
# KUBECTL_ATE_VERSION), not the older v0.2.0-beta5 the sibling
# fraud-ops-console demo's Dockerfile pins.
FROM --platform=$BUILDPLATFORM alpine:3.20 AS ate-fetch
RUN apk add --no-cache curl
ARG KUBECTL_ATE_VERSION=v1.0.0
RUN curl -fsSL -o /usr/local/bin/kubectl-ate "https://github.com/kagent-dev/substrate/releases/download/${KUBECTL_ATE_VERSION}/kubectl-ate-linux-amd64" \
    && chmod +x /usr/local/bin/kubectl-ate

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/app /app
COPY --from=ate-fetch /usr/local/bin/kubectl-ate /usr/local/bin/kubectl-ate
EXPOSE 8090
ENTRYPOINT ["/app"]
