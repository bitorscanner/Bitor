#!/bin/bash

# Bitor Build and Install Script
# This script builds the frontend and backend, creates a version number, and installs it as a system service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="bitor"
SERVICE_USER="bitor"
INSTALL_DIR="/opt/bitor"
BINARY_NAME="bitor"
DATA_DIR="/var/lib/bitor"
LOG_DIR="/var/log/bitor"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if user has sudo access
check_sudo() {
    if ! sudo -n true 2>/dev/null; then
        print_error "This script requires sudo access for system installation"
        print_status "Please ensure you can run sudo commands"
        print_status "Usage: ./build-and-install.sh [options]"
        print_status "Options:"
        print_status "  --build-only    Only build, don't install"
        print_status "  --version=X.Y.Z Custom version number"
        exit 1
    fi
}

# Parse command line arguments
BUILD_ONLY=false
CUSTOM_VERSION=""

for arg in "$@"; do
    case $arg in
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --version=*)
            CUSTOM_VERSION="${arg#*=}"
            shift
            ;;
        *)
            print_error "Unknown option: $arg"
            exit 1
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

# Install dependencies
install_dependencies() {
    print_status "Installing dependencies..."
    
    # Install pnpm if not present
    if ! command -v pnpm &> /dev/null; then
        print_status "Installing pnpm..."
        npm install -g pnpm
    fi
    
    # Install Go if not present
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first."
        exit 1
    fi
    
    print_success "Dependencies installed"
}

# Build frontend
build_frontend() {
    print_status "Building frontend..."
    
    cd frontend
    
    # Install dependencies
    pnpm install
    
    # Build frontend
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
    go build -ldflags "-X 'main.Version=$VERSION' -X 'bitor/version.Version=$VERSION' -X 'bitor/setup.Version=$VERSION'" -o "$BINARY_NAME"
    
    # Make binary executable
    chmod +x "$BINARY_NAME"
    
    cd ..
    
    print_success "Backend built successfully with version $VERSION"
}

# Create system user
create_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        print_status "Creating system user: $SERVICE_USER"
        sudo useradd --system --shell /bin/false --home-dir "$DATA_DIR" --create-home "$SERVICE_USER"
        print_success "User $SERVICE_USER created"
    else
        print_status "User $SERVICE_USER already exists"
    fi
}

# Create directories
create_directories() {
    print_status "Creating directories..."
    
    # Create install directory
    sudo mkdir -p "$INSTALL_DIR"
    
    # Create data directory
    sudo mkdir -p "$DATA_DIR"
    sudo chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
    sudo chmod 750 "$DATA_DIR"
    
    # Create log directory
    sudo mkdir -p "$LOG_DIR"
    sudo chown "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR"
    sudo chmod 750 "$LOG_DIR"
    
    print_success "Directories created"
}

# Install binary and assets
install_binary() {
    print_status "Installing binary and assets..."
    
    # Copy binary
    sudo cp "backend/$BINARY_NAME" "$INSTALL_DIR/"
    sudo chown root:root "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    
    # Copy frontend build (if exists)
    if [[ -d "frontend/build" ]]; then
        sudo cp -r frontend/build "$INSTALL_DIR/pb_public"
        sudo chown -R root:root "$INSTALL_DIR/pb_public"
    fi
    
    print_success "Binary and assets installed"
}

# Create systemd service
create_service() {
    print_status "Creating systemd service..."
    
    sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null << EOF
[Unit]
Description=Bitor Security Scanner
Documentation=https://github.com/bitorscanner/bitor
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/$BINARY_NAME serve --http=0.0.0.0:8090
WorkingDirectory=$DATA_DIR
Restart=always
RestartSec=5

# Security measures
NoNewPrivileges=true
ProtectHome=true
ProtectSystem=strict
ReadWritePaths=$DATA_DIR $LOG_DIR
ProtectHostname=true
ProtectClock=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=false
RemoveIPC=true
PrivateTmp=true

# Environment
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Environment=HOME=$DATA_DIR

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    sudo systemctl daemon-reload
    
    print_success "Systemd service created"
}

# Enable and start service
start_service() {
    print_status "Enabling and starting service..."
    
    # Stop service if running
    if sudo systemctl is-active --quiet "$SERVICE_NAME"; then
        print_status "Stopping existing service..."
        sudo systemctl stop "$SERVICE_NAME"
    fi
    
    # Enable service
    sudo systemctl enable "$SERVICE_NAME"
    
    # Start service
    sudo systemctl start "$SERVICE_NAME"
    
    # Wait a moment and check status
    sleep 3
    
    if sudo systemctl is-active --quiet "$SERVICE_NAME"; then
        print_success "Service started successfully"
        print_status "Service status:"
        sudo systemctl status "$SERVICE_NAME" --no-pager -l
    else
        print_error "Service failed to start"
        print_status "Service logs:"
        sudo journalctl -u "$SERVICE_NAME" --no-pager -l --lines=20
        exit 1
    fi
}

# Main installation process
main() {
    print_status "Starting Bitor build and installation process..."
    print_status "Version: $VERSION"
    print_status "Build only: $BUILD_ONLY"
    
    # Generate version
    generate_version
    
    # Install dependencies
    install_dependencies
    
    # Build components
    build_frontend
    build_backend
    
    if [[ "$BUILD_ONLY" == "true" ]]; then
        print_success "Build completed! Binary location: backend/$BINARY_NAME"
        print_status "To install as a service, run: ./build-and-install.sh"
        exit 0
    fi
    
    # Check sudo access for installation
    check_sudo
    
    # Install system components
    create_user
    create_directories
    install_binary
    create_service
    start_service
    
    print_success "Bitor installation completed!"
    print_status "Service: $SERVICE_NAME"
    print_status "Binary: $INSTALL_DIR/$BINARY_NAME"
    print_status "Data: $DATA_DIR"
    print_status "Logs: $LOG_DIR"
    print_status ""
    print_status "Useful commands:"
    print_status "  sudo systemctl status $SERVICE_NAME    # Check service status"
    print_status "  sudo systemctl restart $SERVICE_NAME   # Restart service"
    print_status "  sudo systemctl stop $SERVICE_NAME      # Stop service"
    print_status "  sudo journalctl -u $SERVICE_NAME -f    # Follow logs"
    print_status ""
    print_status "Web interface should be available at: http://localhost:8090"
}

# Run main function
main "$@"