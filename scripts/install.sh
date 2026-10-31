#!/bin/sh

set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
prefix=${PREFIX:-"$HOME/.local"}
destdir=${DESTDIR:-}

"$root_dir/scripts/build.sh"
install -Dm755 "$root_dir/build/launchy" "${destdir}${prefix}/bin/launchy"
