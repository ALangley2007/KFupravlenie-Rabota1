#!/bin/sh

. "$(dirname "$0")/lib.sh"

section 'Every mode of ls, cd, uname and clear'
vshell -vfs vfs/deep.b64 -script scripts/startup/commands.vsh

section 'A VFS with the home directory of the host user'
user=$(id -un)
mkdir -p "$TMP/home/home/$user/docs"
echo 'Notes in the home directory.' > "$TMP/home/home/$user/notes.txt"
echo 'A document.' > "$TMP/home/home/$user/docs/plan.txt"
sh scripts/mkvfs.sh "$TMP/home" "$TMP/home.b64"
vshell -vfs "$TMP/home.b64" -script scripts/startup/home.vsh

section 'clear prints ANSI escape sequences (made visible with cat -v)'
vshell_typing 'clear
clear -x
clear -z
clear now
' -vfs vfs/minimal.b64 2>&1 | cat -v
