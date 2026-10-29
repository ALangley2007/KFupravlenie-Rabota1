#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Deep VFS'
vshell -vfs vfs/deep.b64 -script scripts/startup/tree.vsh

section 'Deep VFS with the stage 3 startup script'
vshell -vfs vfs/deep.b64 -script scripts/startup/vfs.vsh
