#!/bin/bash
# Symphony Multi-Platform Build Script

set -e

# Set UTF-8 encoding for proper emoji display
export LC_ALL="${LC_ALL:-en_US.UTF-8}"
export LANG="${LANG:-en_US.UTF-8}"

# Ensure terminal supports UTF-8
if [ -z "$LC_ALL" ] && [ -z "$LANG" ]; then
    export LC_ALL=C.UTF-8
    export LANG=C.UTF-8
fi

VERSION=${1:-"1.0.0"}
OUTPUT_DIR="dist"

echo "🎵 Building Symphony v${VERSION}"
echo ""

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf ${OUTPUT_DIR}
mkdir -p ${OUTPUT_DIR}

# Build CSS first
echo "🎨 Building Tailwind CSS..."
npm run build:css

# Build for each platform
echo ""
echo "📦 Building binaries..."

# Windows AMD64
echo "  ⚙️  Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ${OUTPUT_DIR}/symphony-windows-amd64.exe .

# macOS AMD64
echo "  ⚙️  Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o ${OUTPUT_DIR}/symphony-darwin-amd64 .

# macOS ARM64 (Apple Silicon)
echo "  ⚙️  Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o ${OUTPUT_DIR}/symphony-darwin-arm64 .

# Linux AMD64
echo "  ⚙️  Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ${OUTPUT_DIR}/symphony-linux-amd64 .

# Linux ARM64
echo "  ⚙️  Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o ${OUTPUT_DIR}/symphony-linux-arm64 .

echo ""
echo "✅ Build complete!"
echo ""
echo "📂 Output directory: ${OUTPUT_DIR}/"
ls -lh ${OUTPUT_DIR}/
