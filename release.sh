#!/bin/bash
# Simple wrapper for version-bump.sh with common shortcuts

case "$1" in
    "patch"|"p")
        ./version-bump.sh -t patch -y
        ;;
    "minor"|"m")
        ./version-bump.sh -t minor -y
        ;;
    "major"|"M")
        ./version-bump.sh -t major -y
        ;;
    *)
        echo "🚀 Quick Release Script"
        echo
        echo "Usage: $0 <command>"
        echo
        echo "Commands:"
        echo "  patch, p    - Patch release (e.g., v0.5.5 → v0.5.6)"
        echo "  minor, m    - Minor release (e.g., v0.5.5 → v0.6.0)" 
        echo "  major, M    - Major release (e.g., v0.5.5 → v1.0.0)"
        echo
        echo "For advanced options, use: ./version-bump.sh --help"
        ;;
esac