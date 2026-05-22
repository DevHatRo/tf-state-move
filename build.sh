#!/usr/bin/env bash

# Build script for tf-state-move
# Cross-compiles for multiple platforms

set -e

# Get the package name from go.mod
package_name="tf-state-move"

# Define supported platforms
platforms=(
    "linux/amd64"
    "linux/386"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
    "windows/arm64"
)

# Create build directory
build_dir="build"
mkdir -p "$build_dir"

echo "Building $package_name for multiple platforms..."
echo "Build directory: $build_dir"
echo ""

# Build for each platform
for platform in "${platforms[@]}"; do
    # Split platform into OS and architecture
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    # Create output filename
    output_name="$package_name-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name+='.exe'
    fi
    
    output_path="$build_dir/$output_name"
    
    echo "Building for $GOOS/$GOARCH..."
    
    # Build the executable
    env GOOS=$GOOS GOARCH=$GOARCH go build -v -o "$output_path" .
    
    # Check if build was successful
    if [ $? -eq 0 ]; then
        echo "✓ Successfully built: $output_name"
        
        # Show file size
        if command -v du >/dev/null 2>&1; then
            size=$(du -h "$output_path" | cut -f1)
            echo "  Size: $size"
        fi
    else
        echo "✗ Failed to build: $output_name"
        exit 1
    fi
    
    echo ""
done

echo "Build completed successfully!"
echo "All artifacts are available in the '$build_dir' directory:"
echo ""

# List all built artifacts
if command -v ls >/dev/null 2>&1; then
    ls -la "$build_dir/"
fi

echo ""
echo "To create a release, you can zip the artifacts:"
echo "cd $build_dir && zip -r ../${package_name}-release.zip ." 
