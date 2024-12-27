#!/bin/bash

# Get version info (format: v1.0.0-5-g1234567 or v1.0.0)
GIT_DESC=$(git describe --tags 2>/dev/null || echo "v0.0.0")

# Parse version components
BASE_TAG=$(echo $GIT_DESC | cut -d- -f1)
COMMITS_SINCE=$(echo $GIT_DESC | grep -o '[0-9]*-g[0-9a-f]*$' | cut -d- -f1)

# Build final version string
if [ -n "$COMMITS_SINCE" ]; then
    VERSION="${BASE_TAG}+${COMMITS_SINCE}"
else
    VERSION="${BASE_TAG}"
fi

COMMIT=$(git rev-parse --short HEAD)
BUILDTIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Define repository name
REPO_NAME="dhcgn/go-mqtt-dispatcher"

# Build the Docker image
docker build \
    --build-arg VERSION="${VERSION}" \
    --build-arg COMMIT="${COMMIT}" \
    --build-arg BUILDTIME="${BUILDTIME}" \
    -t ${REPO_NAME}:${VERSION} \
    .

# Also tag as latest
docker tag ${REPO_NAME}:${VERSION} ${REPO_NAME}:latest

echo "Built ${REPO_NAME}:${VERSION} and tagged as latest"

# Push the Docker image to the repository
docker push ${REPO_NAME}:${VERSION}
docker push ${REPO_NAME}:latest

echo "Pushed ${REPO_NAME}:${VERSION} and latest to repository"