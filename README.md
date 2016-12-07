# gimlet

Based on [Gin](https://github.com/codegangsta/gin).

## Installation

```
go get github.com/arekkas/gimlet
```

## Usage

```
Because go run uses subprocesses, there is no easy way to kill an app that was started by go run. This
is why gimlet builds the program first, and then executes the binary.

Examples:

- gimlet watch --immediate `--this-will-be-passed-down some-argument`
- gimlet watch --path ./src

Usage:
  gimlet watch <command> [flags]

Flags:
  -a, --app-port string       Port for the Go web server. (default "3001")
  -e, --exclude stringSlice   Relative directories to exclude. (default [.git,vendor])
  -i, --immediate             Run the server immediately after it's built instead of on first http request.
      --interval int          Interval for polling in ms. Lower values require more CPU time. (default 200)
      --kill-on-error         Set to true to kill gimlet if an error occurrs during build or run.
  -l, --listen string         Listening address for the proxy server.
      --path string           Path to watch files from. (default ".")
  -p, --port int              Port for the proxy server. (default 3000)
```
