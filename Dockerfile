# syntax=docker/dockerfile:1

# builder for admin

FROM node:20-bookworm-slim AS web
WORKDIR /repo

COPY web/admin/package*.json web/admin/
RUN --mount=type=cache,target=/root/.npm \
    cd web/admin && (npm ci || npm install)
COPY web/admin web/admin
RUN cd web/admin && npm run build

# 2: build go bin
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

# from stage 1 we take the built admin assets
COPY --from=web /repo/src/serve/admin/dist ./src/serve/admin/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /aplsonic .


# build bento4 mp4decrypt for GAMDL
FROM debian:bookworm-slim AS bento4
ENV DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG BENTO4_VERSION=1.6.0-641
RUN --mount=type=cache,target=/bento4-build,id=bento4-build-${TARGETARCH} \
    set -eux; \
    apt-get update; \
    mkdir -p /out; \
    if [ "$TARGETARCH" = "amd64" ]; then \
        apt-get install -y --no-install-recommends ca-certificates curl unzip; \
        sdkver="$(echo "$BENTO4_VERSION" | tr '.' '-')"; \
        curl -fsSL -o /tmp/bento4.zip \
            "https://www.bok.net/Bento4/binaries/Bento4-SDK-${sdkver}.x86_64-unknown-linux.zip"; \
        unzip -j /tmp/bento4.zip '*/bin/mp4decrypt' -d /out; \
    else \
        apt-get install -y --no-install-recommends ca-certificates git cmake g++ make; \
        git clone --depth 1 --branch "v${BENTO4_VERSION}" \
            https://github.com/axiomatic-systems/Bento4.git /bento4; \
        cmake -S /bento4 -B /bento4-build -DCMAKE_BUILD_TYPE=Release; \
        cmake --build /bento4-build --target mp4decrypt -j"$(nproc)"; \
        cp /bento4-build/mp4decrypt /out/mp4decrypt; \
    fi; \
    chmod +x /out/mp4decrypt; \
    rm -rf /var/lib/apt/lists/*

# actual runtime
FROM debian:bookworm-slim AS runtime
ENV DEBIAN_FRONTEND=noninteractive

# needed:
# chromium: go-rod (browser.go)
# ffmpeg + mp4decrypt (bento4)
# python3 + uv (gamdl)
RUN apt-get update && apt-get install -y --no-install-recommends \
        chromium \
        ffmpeg \
        python3 \
        ca-certificates \
        curl \
        fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

COPY --from=bento4 /out/mp4decrypt /usr/local/bin/mp4decrypt

# uv + gamdl
ENV UV_INSTALL_DIR=/usr/local/bin \
    UV_TOOL_DIR=/opt/uv/tools \
    UV_TOOL_BIN_DIR=/usr/local/bin \
    UV_PYTHON_INSTALL_DIR=/opt/uv/python
RUN curl -LsSf https://astral.sh/uv/install.sh | sh \
    && uv tool install gamdl \
    && rm -rf /root/.cache

# chromium (wrap it so we can acc use it)
RUN printf '#!/bin/sh\nexec /usr/bin/chromium --no-sandbox --disable-dev-shm-usage --disable-gpu "$@"\n' \
        > /usr/local/bin/chromium-wrapper \
    && chmod +x /usr/local/bin/chromium-wrapper \
    && ln -sf /usr/local/bin/chromium-wrapper /usr/local/bin/chromium

# fuck u rod
ENV ROD_BROWSER_BIN=/usr/local/bin/chromium

WORKDIR /app
COPY --from=build /aplsonic /app/aplsonic

# create user for dis
RUN useradd --create-home --uid 1000 aplsonic \
    && mkdir -p /data \
    && chown -R aplsonic:aplsonic /app /opt/uv /data
USER aplsonic

# music + album-art cache; named volumes inherit /data's ownership so uid 1000 can write
VOLUME ["/data"]

# default ports, user can export more later
EXPOSE 4533 4534

ENTRYPOINT ["/app/aplsonic"]
CMD ["serve"]

# for AIO package :sigh:
FROM runtime AS aio
USER root
ENV DEBIAN_FRONTEND=noninteractive \
    APLSONIC_DATA=/data

RUN apt-get update && apt-get install -y --no-install-recommends \
        mariadb-server \
    && rm -rf /var/lib/apt/lists/*

# entrypoint, config
COPY config/configuration.aio.yml /app/configuration.yml
COPY docker/aio-entrypoint.sh /usr/local/bin/aio-entrypoint.sh
RUN chmod +x /usr/local/bin/aio-entrypoint.sh

# data for all of it
VOLUME ["/data"]
EXPOSE 4533 4534

ENTRYPOINT ["/usr/local/bin/aio-entrypoint.sh"]
CMD ["serve"]