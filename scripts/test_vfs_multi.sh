#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Multi-file VFS'
vshell -vfs vfs/multi.b64 -script scripts/startup/tree.vsh

section 'Multi-file VFS, explored interactively'
vshell_typing 'tree docs
tree ./docs/../logo.png
tree data.bin/x
' -vfs vfs/multi.b64
