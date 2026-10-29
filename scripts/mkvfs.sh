#!/bin/sh

if [ $# -ne 2 ] || [ ! -d "$1" ]; then
	echo "usage: $0 source_dir output.b64" >&2
	exit 2
fi
output=$(cd "$(dirname "$2")" && pwd)/$(basename "$2") || exit 1
cd "$1" || exit 1
zip -qrX - . -x '*.DS_Store' | base64 | fold -w 76 > "$output"
