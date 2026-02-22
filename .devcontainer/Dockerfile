FROM mcr.microsoft.com/devcontainers/base:ubuntu

# System dependencies: postgres client + Chromium libs for Playwright
# libasound2t64 is the Ubuntu 24.04+ name; older versions use libasound2
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    libglib2.0-0 libnss3 libnspr4 libdbus-1-3 libatk1.0-0 libatk-bridge2.0-0 \
    libcups2 libdrm2 libxkbcommon0 libatspi2.0-0 libxcomposite1 libxdamage1 \
    libxfixes3 libxrandr2 libgbm1 libpango-1.0-0 libcairo2 \
    && (apt-get install -y --no-install-recommends libasound2t64 2>/dev/null \
        || apt-get install -y --no-install-recommends libasound2) \
    && rm -rf /var/lib/apt/lists/*

# Go — detect architecture at build time
ENV GO_VERSION=1.24.2
RUN ARCH="$(dpkg --print-architecture)" \
    && case "${ARCH}" in \
         amd64) GOARCH="amd64" ;; \
         arm64) GOARCH="arm64" ;; \
         armhf) GOARCH="armv6l" ;; \
         *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
       esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:/home/vscode/go/bin:${PATH}" \
    GOPATH="/home/vscode/go"

# Go tools (as vscode user so binaries land in ~/go/bin)
USER vscode
RUN go install github.com/air-verse/air@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
USER root

# Node.js + Playwright CLI + Chromium browser
ENV NODE_VERSION=20
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g @playwright/cli@latest @anthropic-ai/claude-code@latest
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/playwright-browsers
RUN mkdir -p $PLAYWRIGHT_BROWSERS_PATH \
    && playwright-cli install chromium \
    && chmod -R o+rx $PLAYWRIGHT_BROWSERS_PATH

# Allow vscode user to update globally-installed npm packages (e.g. claude-code)
RUN NPM_GLOBAL="$(npm root -g)" \
    && chown -R vscode:vscode "$NPM_GLOBAL" \
    && chown -R vscode:vscode "$(dirname "$(which claude)")"
