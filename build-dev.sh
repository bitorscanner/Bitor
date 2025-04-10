#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Set environment variables
export APP_ENV=production
export BUILD_MODE=""
export CI_COMMIT_TAG=""

# # Determine the current Git branch
# CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# # Set BUILD_MODE and CI_COMMIT_TAG based on the branch
# if [ -z "$(git tag --points-at HEAD)" ]; then
#   if [ "$CURRENT_BRANCH" == "develop" ]; then
#     export BUILD_MODE="snapshot"
#     export CI_COMMIT_TAG="development-$(date +%Y%m%d%H%M%S)"
#   else
#     export CI_COMMIT_TAG=$(git rev-parse --short HEAD)
#   fi
# else
#   export CI_COMMIT_TAG=$(git describe --tags)
# fi

echo "Building Bitor with tag: DEVELOPMENT"

# Install pnpm if it's not installed
if ! command -v pnpm &> /dev/null; then
  echo "pnpm not found. Installing pnpm..."
  npm install -g pnpm
fi

# Install frontend dependencies and build
echo "Installing frontend dependencies..."
pnpm --prefix=./frontend install

echo "Building frontend..."
PUBLIC_VERSION=${CI_COMMIT_TAG} pnpm --prefix=./frontend run build

# Build the backend
echo "Tidying Go modules..."
(cd ./backend && go mod tidy)

# Build backend
echo "Building backend..."
cd backend
go build -o bitor
cd ..

# Run goreleaser
# echo "Running Goreleaser..."
# if [ "$BUILD_MODE" == "snapshot" ]; then
#   curl -sL https://git.io/goreleaser | bash -s -- --snapshot --clean --skip=publish,announce
# else
#   curl -sL https://git.io/goreleaser | bash -s -- release --clean --skip=publish,announce
# fi

echo "Build completed successfully!" 