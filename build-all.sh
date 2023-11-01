#!/bin/bash
set -e

ROOTFS_PATH="/tmp/root-filesystem"

tarball_path() {
    echo "/tmp/package/$1.tar.gz"
}

build() {
    go run . build "$@"
}

package() {
    BUILDER_NAME="$1"
    TARBALL_PATH=$(tarball_path "$BUILDER_NAME")
    go run . package tarball --build-output-path "/tmp/output/$BUILDER_NAME" --output-file-path "$TARBALL_PATH"
}

install() {
    TARBALL_PATH=$(tarball_path "$1")
    go run . install tarball -s "$TARBALL_PATH" -D "$ROOTFS_PATH"
}

build_standard_builder() {
    build "$@" --output-directory-path "/tmp/output/$BUILDER_NAME" --toolchain-directory-path "/tmp/output/cross-llvm" --root-fs-directory-path "$ROOTFS_PATH"
}

fhs() {
    BUILDER_NAME="root-filesystem"
    build "$BUILDER_NAME" --output-directory-path "/tmp/output/$BUILDER_NAME"
    package "$BUILDER_NAME"
    install "$BUILDER_NAME"
}

linux_headers() {
    BUILDER_NAME="linux-headers"
    build "$BUILDER_NAME" --output-directory-path "/tmp/output/$BUILDER_NAME"
    package "$BUILDER_NAME"
    install "$BUILDER_NAME"
}

standard_builder() {
    BUILDER_NAME="$1"
    build_standard_builder "$@"
    package "$BUILDER_NAME"
    install "$BUILDER_NAME"
}

rm -rf "$ROOTFS_PATH"
fhs
linux_headers
standard_builder musl-libc
standard_builder zlib-ng
standard_builder xz
standard_builder lz4
standard_builder zstd
standard_builder libressl
standard_builder busybox --config-file-path ./assets/busybox/.config
standard_builder linux-kernel --config-file-path ./assets/linux-kernel/.config
chroot "$ROOTFS_PATH" echo "chroot test"
