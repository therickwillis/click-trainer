FROM mcr.microsoft.com/devcontainers/base:ubuntu

# System dependencies: postgres client + Chromium libs for Playwright
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    libglib2.0-0 libnss3 libnspr4 libdbus-1-3 libatk1.0-0 libatk-bridge2.0-0 \
    libcups2 libdrm2 libxkbcommon0 libatspi2.0-0 libxcomposite1 libxdamage1 \
    libxfixes3 libxrandr2 libgbm1 libpango-1.0-0 libcairo2 libasound2t64 \
    && rm -rf /var/lib/apt/lists/*

# Go
ENV GO_VERSION=1.24.2
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz
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
RUN npm install -g @playwright/cli@latest
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/playwright-browsers
RUN mkdir -p $PLAYWRIGHT_BROWSERS_PATH \
    && playwright-cli install chromium \
    && chmod -R o+rx $PLAYWRIGHT_BROWSERS_PATH
