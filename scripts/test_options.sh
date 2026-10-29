#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'No parameters'
vshell

section 'VFS path only'
vshell -vfs vfs/minimal.b64

section 'Startup script only'
vshell -script scripts/startup/hello.vsh

section 'Both parameters'
vshell -vfs vfs/minimal.b64 -script scripts/startup/hello.vsh

section 'Both parameters in the opposite order'
vshell -script scripts/startup/hello.vsh -vfs vfs/minimal.b64

section 'Double-dash and name=value spellings'
vshell --vfs=vfs/minimal.b64 --script=scripts/startup/hello.vsh

section 'Paths with spaces and an absolute path'
cp vfs/minimal.b64 "$TMP/my vfs.b64"
vshell -vfs "$TMP/my vfs.b64" -script "$ROOT/scripts/startup/hello.vsh"

section 'Help'
vshell -h
