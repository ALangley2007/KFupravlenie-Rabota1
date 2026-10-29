#!/bin/sh

ROOT=$(cd "$(dirname "$0")/.." && pwd) || exit 1
cd "$ROOT" || exit 1
go build -o bin/vshell ./src || exit 1

TMP=$(mktemp -d bin/tmp.XXXXXX) || exit 1
trap 'rm -rf "$TMP"' EXIT

section() {
	printf '\n### %s\n' "$1"
}

show_args() {
	for arg in "$@"; do
		case $arg in
			'' | *[[:space:]]*) printf " '%s'" "$arg" ;;
			*) printf ' %s' "$arg" ;;
		esac
	done
}

vshell_typing() {
	input=$1
	shift
	printf '$ vshell'
	show_args "$@"
	printf '\n'
	printf '%s' "$input" | bin/vshell "$@"
	printf '[exit status: %d]\n' "$?"
}

vshell() {
	vshell_typing '' "$@"
}
