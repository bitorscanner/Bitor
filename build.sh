#!/bin/bash

# Simple Build Script for Bitor
# This script builds the frontend and backend with version injection

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Parse command line arguments
CUSTOM_VERSION=""

for arg in "$@"; do
    case $arg in
        --version=*)
            CUSTOM_VERSION="${arg#*=}"
            shift
            ;;
        *)
            print_status "Usage: ./build.sh [--version=X.Y.Z]"
            print_status "If no version is specified, it will be auto-generated"
            ;;
    esac
done

# Generate version number
generate_version() {
    if [[ -n "$CUSTOM_VERSION" ]]; then
        VERSION="$CUSTOM_VERSION"
        print_status "Using custom version: $VERSION"
    else
        # Get git tag if available, otherwise generate timestamp-based version
        if git describe --tags --exact-match HEAD 2>/dev/null; then
            VERSION=$(git describe --tags --exact-match HEAD)
        elif git rev-parse --git-dir > /dev/null 2>&1; then
            BRANCH=$(git rev-parse --abbrev-ref HEAD)
            COMMIT=$(git rev-parse --short HEAD)
            TIMESTAMP=$(date +%Y%m%d.%H%M%S)
            if [[ "$BRANCH" == "main" ]] || [[ "$BRANCH" == "master" ]]; then
                VERSION="v1.0.0-${TIMESTAMP}-${COMMIT}"
            else
                VERSION="v1.0.0-${BRANCH}-${TIMESTAMP}-${COMMIT}"
            fi
        else
            # No git repository, use timestamp
            TIMESTAMP=$(date +%Y%m%d.%H%M%S)
            VERSION="v1.0.0-${TIMESTAMP}"
        fi
        print_status "Generated version: $VERSION"
    fi
}

# Install pnpm if needed
install_pnpm() {
    if ! command -v pnpm &> /dev/null; then
        print_status "Installing pnpm..."
        npm install -g pnpm
    fi
}

# Build frontend
build_frontend() {
    print_status "Building frontend..."
    
    cd frontend
    
    # Install dependencies
    pnpm install
    
    # Build frontend with version
    PUBLIC_VERSION="$VERSION" pnpm run build
    
    cd ..
    
    print_success "Frontend built successfully"
}

# Build backend
build_backend() {
    print_status "Building backend..."
    
    cd backend
    
    # Tidy go modules
    go mod tidy
    
    # Build with version injection
    go build -ldflags "-X 'main.Version=$VERSION' -X 'bitor/version.Version=$VERSION' -X 'bitor/setup.Version=$VERSION'" -o bitor
    
    # Make binary executable
    chmod +x bitor
    
    cd ..
    
    print_success "Backend built successfully with version $VERSION"
}

# Main build process
main() {
    print_status "Starting Bitor build process..."
    
    # Generate version
    generate_version
    
    # Install pnpm if needed
    install_pnpm
    
    # Build components
    build_frontend
    build_backend
    
    print_success "Build completed successfully!"
    print_status "Binary location: backend/bitor"
    print_status "Version: $VERSION"
    print_status ""
    print_status "To run locally: cd backend && ./bitor serve"
    print_status "To install as system service: sudo ./build-and-install.sh"
}

# Run main function
main "$@"