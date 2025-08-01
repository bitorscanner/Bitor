# Bitor Build and Installation Instructions

This document describes how to build and install Bitor using the provided scripts.

## Build Scripts

### Quick Build (`build.sh`)

For development or testing, use the simple build script:

```bash
# Build with auto-generated version
./build.sh

# Build with custom version
./build.sh --version=v1.2.3
```

This script will:
- Generate a version number automatically (or use the one you specify)
- Install pnpm if needed
- Build the frontend (SvelteKit)
- Build the backend (Go) with version injection
- Create an executable at `backend/bitor`

### Build and Install (`build-and-install.sh`)

For production deployment with system service:

```bash
# Build only (no installation)
./build-and-install.sh --build-only

# Build and install as system service (uses sudo for system operations)
./build-and-install.sh

# Build and install with custom version
./build-and-install.sh --version=v1.2.3
```

This script will:
- Everything from the build script
- Create a system user (`bitor`)
- Install the binary to `/opt/bitor/`
- Create data directory at `/var/lib/bitor/`
- Create log directory at `/var/log/bitor/`
- Install systemd service file
- Start and enable the service

## Prerequisites

### Required
- Node.js (for pnpm)
- Go (latest version recommended)
- Git (for version generation)

### For System Installation
- Sudo access (user must be able to run `sudo` commands)
- systemd-based Linux distribution

## Version Generation

The scripts automatically generate version numbers based on:

1. **Git tag**: If HEAD points to a git tag, uses that tag
2. **Git branch**: If on main/master, uses `v1.0.0-TIMESTAMP-COMMIT`
3. **Other branches**: Uses `v1.0.0-BRANCH-TIMESTAMP-COMMIT`
4. **No git**: Uses `v1.0.0-TIMESTAMP`

You can override this with `--version=X.Y.Z`

## Service Management

After installation, manage the service with:

```bash
# Check status
sudo systemctl status bitor

# Start/stop/restart
sudo systemctl start bitor
sudo systemctl stop bitor
sudo systemctl restart bitor

# View logs
sudo journalctl -u bitor -f

# Disable auto-start
sudo systemctl disable bitor
```

## File Locations

After installation:

- **Binary**: `/opt/bitor/bitor`
- **Frontend**: `/opt/bitor/pb_public/`
- **Data**: `/var/lib/bitor/`
- **Logs**: `/var/log/bitor/`
- **Service**: `/etc/systemd/system/bitor.service`

## Web Interface

The web interface will be available at:
- **Default**: `http://localhost:8090`
- **Network**: `http://YOUR_IP:8090`

## Security

The systemd service includes several security hardening measures:
- Runs as dedicated user
- Restricted filesystem access
- Memory protection
- No new privileges
- Protected system directories

## Troubleshooting

### Build Issues
- Ensure Node.js and Go are installed
- Check internet connectivity for dependency downloads
- Verify filesystem permissions

### Service Issues
- Check logs: `sudo journalctl -u bitor`
- Verify permissions on data directory
- Ensure port 8090 is available
- Check firewall settings

### Common Solutions
```bash
# Rebuild and reinstall
sudo systemctl stop bitor
./build-and-install.sh

# Reset data (WARNING: destroys all data)
sudo systemctl stop bitor
sudo rm -rf /var/lib/bitor/*
sudo systemctl start bitor

# Manual start for debugging
sudo -u bitor /opt/bitor/bitor serve --http=0.0.0.0:8090
```

## Development

For development, use the simple build script and run manually:

```bash
./build.sh
cd backend
./bitor serve
```

This allows for faster iteration without system installation.