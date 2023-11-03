#!/bin/bash
set -ue

# Install packages
apt update
apt dist-upgrade -y
# TODO consider including libressl instead, however this would require building it from source so it may not be a great fit here.
apt install -y git clang lld valgrind gnulib cmake ninja-build python-is-python3 clang libzstd1 zlib1g-dev libzstd-dev po4a doxygen flex libelf-dev bc cpio libssl-dev openssl zstd kmod fontforge perl libfont-ttf-perl meson


