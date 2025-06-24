# Build stage for local development
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY backend/ .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bitor

# Base stage with common dependencies
FROM alpine:3.19 AS base
WORKDIR /app

# Install required packages
RUN apk add --no-cache \
    python3 \
    py3-pip \
    py3-virtualenv \
    openssh-client \
    git \
    bash \
    curl \
    wget \
    unzip \
    jq

# Set up Python virtual environment and install Ansible
RUN python3 -m venv /opt/venv && \
    . /opt/venv/bin/activate && \
    pip3 install --no-cache-dir \
    ansible==8.5.0 \
    ansible-core==2.15.5 \
    pywinrm \
    requests && \
    ansible-galaxy collection install azure.azcollection community.general && \
    deactivate

# Update PATH to include virtual environment
ENV PATH="/opt/venv/bin:/app:${PATH}"

# Install Terraform
RUN wget -O /tmp/terraform.zip https://releases.hashicorp.com/terraform/1.10.5/terraform_1.10.5_linux_amd64.zip && \
    unzip /tmp/terraform.zip -d /usr/local/bin && \
    rm /tmp/terraform.zip && \
    chmod +x /usr/local/bin/terraform

# Create necessary directories
RUN mkdir -p /app/pb_data && \
    chmod -R 755 /app/pb_data

# Set environment variables for Ansible configuration
ENV ANSIBLE_FORCE_COLOR=true \
    ANSIBLE_ACTION_WARNINGS=false \
    ANSIBLE_STDOUT_CALLBACK=default \
    ANSIBLE_RETRY_FILES_ENABLED=false

# Expose the application port
EXPOSE 8090

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8090/_/ || exit 1

# Copy entrypoint script
COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh

# Local stage - uses locally built binary
FROM base AS local
COPY --from=builder /build/bitor /app/bitor
COPY backend/pb_public /app/pb_public
RUN chmod +x /app/bitor
ENTRYPOINT ["/docker-entrypoint.sh"]

# GitHub release stage
FROM base AS github
ARG TARGETARCH=x86_64
ARG VERSION
# Set a default version if none provided
ENV BITOR_VERSION=${VERSION:-v1.0.9}
RUN echo "Downloading version: ${BITOR_VERSION} for ${TARGETARCH}" && \
    wget -O bitor.tar.gz \
    "https://github.com/bitorscanner/bitor/releases/download/${BITOR_VERSION}/bitor_${BITOR_VERSION#v}_Linux_${TARGETARCH}.tar.gz" && \
    tar xzf bitor.tar.gz && \
    rm bitor.tar.gz && \
    chmod +x bitor

# Final stage - use --target=local for local builds
FROM github AS final
ENTRYPOINT ["/docker-entrypoint.sh"]
