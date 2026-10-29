#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Missing VFS file'
vshell -vfs vfs/missing.b64 -script scripts/startup/tree.vsh

section 'Directory instead of a VFS file'
vshell -vfs vfs -script scripts/startup/tree.vsh

section 'Empty file'
: > "$TMP/empty.b64"
vshell -vfs "$TMP/empty.b64" -script scripts/startup/tree.vsh

section 'Text that is not base64'
echo 'This is not an archive.' > "$TMP/text.b64"
vshell -vfs "$TMP/text.b64" -script scripts/startup/tree.vsh

section 'Base64 of something that is not a ZIP archive'
echo 'plain text' | base64 > "$TMP/plain.b64"
vshell -vfs "$TMP/plain.b64" -script scripts/startup/tree.vsh

section 'Truncated ZIP archive'
base64 -d < vfs/deep.b64 | head -c 200 > "$TMP/truncated.zip"
vshell -vfs "$TMP/truncated.zip" -script scripts/startup/tree.vsh
