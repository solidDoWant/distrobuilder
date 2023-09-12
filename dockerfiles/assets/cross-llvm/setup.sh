#!/bin/bash
set -ue

# Install packages
apt update
apt dist-upgrade -y
apt install -y git clang lld valgrind gnulib cmake ninja-build python-is-python3 clang libzstd1 zlib1g-dev libzstd-dev
