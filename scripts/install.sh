#!/bin/sh

set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
prefix=${PREFIX:-"$HOME/.local"}
destdir=${DESTDIR:-}
config_home=${XDG_CONFIG_HOME:-"$HOME/.config"}

"$root_dir/scripts/build.sh"
install -Dm755 "$root_dir/build/launchy" "${destdir}${prefix}/bin/launchy"

config_file="${destdir}${config_home}/launchy/config.json"
if [ ! -e "$config_file" ] && [ ! -L "$config_file" ]; then
    install -Dm644 "$root_dir/config.example.json" "$config_file"
fi
