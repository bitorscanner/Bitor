#!/bin/bash
set -e

# Create necessary directories
mkdir -p docker/data docker/nuclei-templates

# Clean up any existing containers
echo "Cleaning up existing containers..."
docker rm -f orbit-test 2>/dev/null || true

# Build the image
echo "Building local image..."
docker build --target=local -t orbit:local .

# Run the container directly
echo "Running container directly..."
docker run --name orbit-test \
    -e API_ENCRYPTION_KEY=12345678901234567890123456789012 \
    -v "$(pwd)/docker/data:/app/pb_data" \
    -v "$(pwd)/docker/nuclei-templates:/app/nuclei-templates" \
    -p 8090:8090 \
    orbit:local

# If we get here, the container exited
echo "Container exited. Showing logs:"
docker logs orbit-test 