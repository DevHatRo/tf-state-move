#!/usr/bin/env bash

# Build script for tf-state-move (CI-compatible)
# Cross-compiles for the same platforms as defined in CI workflow

set -e

# Get the package name from go.mod
package_name="tf-state-move"

# Define platforms matching CI workflow
platforms=(
    "linux/amd64"
    "windows/amd64"
    "darwin/amd64"
)

# Create build directory
build_dir="build"
mkdir -p "$build_dir"

echo "Building $package_name for CI platforms..."
echo "Build directory: $build_dir"
echo ""

# Build for each platform
for platform in "${platforms[@]}"; do
    # Split platform into OS and architecture
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    # Create output filename (matching CI artifact names)
    output_name="$package_name-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name+='.exe'
    fi
    
    output_path="$build_dir/$output_name"
    
    echo "Building for $GOOS/$GOARCH..."
    
    # Build the executable (matching CI build command)
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
echo "These artifacts match your CI workflow configuration." 
