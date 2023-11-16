#!/bin/bash
set -ue

# Install packages
sed -i 's/Suites: bookworm bookworm-updates/Suites: bookworm bookworm-updates experimental/' /etc/apt/sources.list.d/debian.sources
apt update
apt dist-upgrade -y
# TODO consider including libressl instead, however this would require building it from source so it may not be a great fit here.
apt install --no-install-recommends -y git clang lld valgrind gnulib cmake ninja-build python-is-python3 clang libzstd1 zlib1g-dev libzstd-dev po4a doxygen flex libelf-dev bc cpio libssl-dev openssl zstd kmod fontforge perl libfont-ttf-perl libtool-bin xsltproc libxml2-utils texlive-latex-base texlive-formats-extra poppler-utils help2man groff
apt install --no-install-recommends -y -t experimental meson    # version 1.3.0 rc1 or newer is needed for --reconfigure and --clearcache support on the setup subcommand


