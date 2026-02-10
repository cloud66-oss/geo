# use BUILDPLATFORM so the build always runs natively (no QEMU emulation)
FROM --platform=$BUILDPLATFORM golang:1.25 AS build

# TARGETOS and TARGETARCH are set automatically by buildx
ARG TARGETOS
ARG TARGETARCH
ARG SHORT_SHA

WORKDIR /app/geo
COPY . .
RUN go mod vendor

# cross-compile natively instead of emulating the target architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X 'github.com/cloud66-oss/geo/utils.Version=${SHORT_SHA}'" -o /app/geo/geo

FROM alpine:3.21
LABEL maintainer="Cloud 66 Engineering <hello@cloud66.com>"

COPY --from=build /app/geo/geo /app/geo
