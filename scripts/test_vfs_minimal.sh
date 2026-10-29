#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Minimal VFS from base64 text'
vshell -vfs vfs/minimal.b64 -script scripts/startup/tree.vsh

section 'Minimal VFS, explored interactively'
vshell_typing 'tree
tree readme.txt
tree readme.txt/
' -vfs vfs/minimal.b64

section 'Base64 text on a single line'
tr -d '\n' < vfs/minimal.b64 > "$TMP/minimal-oneline.b64"
vshell -vfs "$TMP/minimal-oneline.b64" -script scripts/startup/tree.vsh

section 'Raw ZIP archive'
base64 -d < vfs/minimal.b64 > "$TMP/minimal.zip"
vshell -vfs "$TMP/minimal.zip" -script scripts/startup/tree.vsh

section 'No VFS given: an empty one'
vshell -script scripts/startup/tree.vsh
