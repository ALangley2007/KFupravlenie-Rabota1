#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Comments, quoting and stub commands'
vshell -vfs vfs -script scripts/startup/demo.vsh

section 'Errors are reported with the line number and the script goes on'
vshell -vfs vfs -script scripts/startup/errors.vsh

section 'exit in the script ends the session with its status'
vshell -vfs vfs -script scripts/startup/exit.vsh

section 'The interactive session goes on after the script'
vshell_typing 'cd typed "by user"
exit 4
' -vfs vfs -script scripts/startup/hello.vsh
