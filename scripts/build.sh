#!/usr/bin/env bash
set -e

mkdir -p ./bin

# Build all lib and scripts /*/main.go files
for dir in ./lib/* ./scripts/*; do
    if [ -d "$dir" ] && [ -f "$dir/main.go" ]; then
        folder_name=$(basename "$dir")
        output="./bin/niv-$folder_name"
        echo "Building $dir/main.go -> $output"
        go build -o "$output" "$dir/main.go"
    fi
done

# Build ./main.go
if [ -f ./main.go ]; then
    echo "Building ./main.go -> ./bin/niv"
    go build -o ./bin/niv ./main.go
fi

echo "Build completed."
