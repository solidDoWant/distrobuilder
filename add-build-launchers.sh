#!/bin/bash

function usage() {
    echo "Usage:"
    echo "$0 <build name>"
}

if [ "$#" -ne 1 ]; then
    >&2 usage
    exit 1
fi

BUILD_NAME="$1"

JQ_TEMPLATE=$(cat << EOF
[
    {
        "name": "Launch build $BUILD_NAME",
        "type": "go",
        "request": "launch",
        "mode": "auto",
        "program": "\${workspaceFolder}/main.go",
        "args": [
            "build",
            "$BUILD_NAME",
            "--output-directory-path",
            "/tmp/output/$BUILD_NAME",
            "--toolchain-directory-path",
            "/tmp/output/cross-llvm",
            "--root-fs-directory-path",
            "/tmp/root-filesystem"
        ]
    },
    {
        "name": "Launch package $BUILD_NAME",
        "type": "go",
        "request": "launch",
        "mode": "auto",
        "program": "\${workspaceFolder}/main.go",
        "args": [
            "package",
            "tarball",
            "-c",
            "--build-output-path",
            "/tmp/output/$BUILD_NAME",
            "--output-file-path",
            "/tmp/package/$BUILD_NAME.tar.gz"
        ]
    },
    {
        "name": "Launch install $BUILD_NAME",
        "type": "go",
        "request": "launch",
        "mode": "auto",
        "program": "\${workspaceFolder}/main.go",
        "args": [
            "install",
            "tarball",
            "-c",
            "-s",
            "/tmp/package/$BUILD_NAME.tar.gz",
            "-D",
            "/tmp/root-filesystem"
        ]
    }
]
EOF
)

LAUNCH_FILE=".vscode/launch.json"
TEMP_FILE="/tmp/launch.json"
jq -r --argjson BUILT_TEMPLATE "$JQ_TEMPLATE" '.configurations |= . + $BUILT_TEMPLATE' "$LAUNCH_FILE" > "$TEMP_FILE"
cat "$TEMP_FILE" > "$LAUNCH_FILE"
rm "$TEMP_FILE"