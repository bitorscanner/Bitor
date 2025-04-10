#!/bin/sh
set -e

# Check if API_ENCRYPTION_KEY is set
if [ -z "${API_ENCRYPTION_KEY}" ]; then
    echo "Error: API_ENCRYPTION_KEY environment variable is required"
    echo "Please run the container with: -e API_ENCRYPTION_KEY=your_32_character_secret"
    exit 1
fi

# Set Ansible host key checking based on environment or default to false
export ANSIBLE_HOST_KEY_CHECKING=${ANSIBLE_HOST_KEY_CHECKING:-false}

# Start the application
cd /app && exec ./bitor serve --http 0.0.0.0:8090