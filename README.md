# launchy

A fast terminal application launcher for Linux.

## Build

All Go dependencies are committed in `vendor/`, so builds do not need to fetch
modules:

```sh
./scripts/build.sh
```

The binary is written to `build/launchy`.

## Install

Install for the current user:

```sh
./scripts/install.sh
```

This installs `launchy` to `~/.local/bin`. To install elsewhere, set `PREFIX`:

```sh
PREFIX=/usr/local ./scripts/install.sh
```

The installer also creates `~/.config/launchy/config.json` with the sample
palette, without replacing an existing config. `XDG_CONFIG_HOME` is respected.

## Run

```sh
launchy list
```

`list` shows selectable desktop entries from `/usr/share/applications`,
`~/.local/share/applications`, and `~/Desktop`. Selected applications are
started in a new session detached from the terminal. Type in the bottom filter
field to narrow the list by application name. Up/Down moves through the list
and wraps around; Enter launches the selected application.

Use the **Settings** tab (F2) to view the config and press Enter to edit it with
`$EDITOR` (defaults to `vi`). Palette colors accept W3C color names, `#RRGGBB`,
or `default` to use the terminal color.
