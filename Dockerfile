# Use a recent Ubuntu base for compatibility
FROM ubuntu:24.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /workspace

# Install system dependencies
# - build-essential: for GCC (needed by CGO)
# - curl/wget: for downloading tools
# - git: for Bazel to fetch git repositories
# - nodejs/npm: for installing Bazelisk (matching your project's existing pattern)
# - python3: often required by Bazel rules
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    nodejs \
    npm \
    python3 \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.25.0
# We install to /usr/local/go, which is the standard location
RUN curl -L -o go1.25.0.linux-amd64.tar.gz https://go.dev/dl/go1.25.0.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz && \
    rm go1.25.0.linux-amd64.tar.gz

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Configure Go environment variables to use a directory inside /workspace
# This ensures that if you mount your local dir to /workspace, the cache persists (if you want it to)
# or is at least easily accessible.
ENV GOPATH="/workspace/.go"
ENV GOCACHE="/workspace/.cache/go-build"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install Bazelisk globally
RUN npm install -g @bazel/bazelisk

# Default command
CMD ["/bin/bash"]
