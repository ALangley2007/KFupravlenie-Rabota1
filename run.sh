#!/bin/sh
ROOT=$(cd "$(dirname "$0")" && pwd) || exit 1
(cd "$ROOT" && go build -o bin/vshell ./src) || exit 1
exec "$ROOT/bin/vshell" "$@"
