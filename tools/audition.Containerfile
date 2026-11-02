ARG GO_VERSION
FROM docker.io/library/golang:${GO_VERSION} AS go

FROM docker.io/library/ubuntu:24.04

# keep the native dependencies aligned with sessionmixer's ALSA and GLFW bindings.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates build-essential pkg-config \
    libgtk-3-dev libasound2-dev libgl1-mesa-dev \
    libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}" \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GOAMD64=v1 \
    GOTOOLCHAIN=local GOWORK=off \
    GOPATH=/cache/modules GOCACHE=/cache/build
