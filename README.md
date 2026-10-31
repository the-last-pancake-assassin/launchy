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

## Run

```sh
launchy list
```

`list` shows selectable desktop entries from `/usr/share/applications`,
`~/.local/share/applications`, and `~/Desktop`. Selected applications are
started in a new session detached from the terminal. Type in the bottom filter
field to narrow the list by application name. Up/Down moves through the list
and wraps around; Enter launches the selected application.
