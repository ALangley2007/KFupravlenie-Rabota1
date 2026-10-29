#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Unknown flag (status 2)'
vshell -vfs vfs/minimal.b64 -verbose

section 'Flag without a value (status 2)'
vshell -vfs vfs/minimal.b64 -script

section 'Unexpected positional argument (status 2)'
vshell -vfs vfs/minimal.b64 -script scripts/startup/hello.vsh extra

section 'Missing startup script (status 127)'
vshell -vfs vfs/minimal.b64 -script scripts/startup/missing.vsh

section 'Startup script is a directory (status 126)'
vshell -vfs vfs/minimal.b64 -script scripts/startup
