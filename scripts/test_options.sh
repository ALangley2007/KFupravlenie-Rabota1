#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'No parameters'
vshell

section 'VFS path only'
vshell -vfs vfs

section 'Startup script only'
vshell -script scripts/startup/hello.vsh

section 'Both parameters'
vshell -vfs vfs -script scripts/startup/hello.vsh

section 'Both parameters in the opposite order'
vshell -script scripts/startup/hello.vsh -vfs vfs

section 'Double-dash and name=value spellings'
vshell --vfs=vfs --script=scripts/startup/hello.vsh

section 'Paths with spaces and an absolute path'
vshell -vfs "$HOME/my vfs" -script "$ROOT/scripts/startup/hello.vsh"

section 'Help'
vshell -h
