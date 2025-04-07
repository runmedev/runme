FROM golang:1.24-bookworm

LABEL org.opencontainers.image.authors="StatefulHQ <mail@stateful.com>"
LABEL org.opencontainers.image.source="https://github.com/runmedev/runme"
LABEL org.opencontainers.image.ref.name="runme-build-env"
LABEL org.opencontainers.image.title="Runme build environment"
LABEL org.opencontainers.image.description="An image to build and test runme."

ENV HOME=/root
ENV SHELL=/bin/bash

RUN apt-get update && apt-get install -y \
    "bash" \
    "curl" \
    "make" \
    "python3" \
    "ruby-full" \
    "unzip"

# Install babashka
RUN curl -sLO https://raw.githubusercontent.com/babashka/babashka/master/install && \
    chmod +x install && \
    ./install && \
    rm install

# Install rust + rust-script
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y --profile minimal && \
    rm -f "$HOME/.bashrc" && \
    . "$HOME/.cargo/env" && cargo install rust-script

# Install node.js
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# Install deno
ENV DENO_INSTALL="$HOME/.deno"
RUN curl -fsSL https://deno.land/install.sh | sh && \
    cp "$DENO_INSTALL/bin/deno" /usr/local/bin/deno

# Install direnv
RUN curl -fsSL https://direnv.net/install.sh | bash

# Configure workspace
WORKDIR /workspace

# Handle permissions when mounting a host directory to /workspace
RUN git config --global --add safe.directory /workspace

# Populate Go cache
COPY go.sum go.mod /workspace/
RUN go mod download -x

# Set output for the runme binary
ENV BUILD_OUTPUT=/usr/local/bin/runme
# Enable testing with race detector
ENV RACE=false

# Default command can be overridden for different purposes (build vs test)
CMD [ "make", "test" ]
