#!/bin/sh

. "$(dirname "$0")/lib.sh"

check_unchanged() {
	if [ "$(cksum < "$1")" = "$2" ]; then
		echo "[$1 on disk: unchanged]"
	else
		echo "[$1 on disk: CHANGED]"
		exit 1
	fi
}

section 'Every mode of mv on the deep VFS'
sum=$(cksum < vfs/deep.b64)
vshell -vfs vfs/deep.b64 -script scripts/startup/mv.vsh
check_unchanged vfs/deep.b64 "$sum"

section 'Moving files of the multi-file VFS'
sum=$(cksum < vfs/multi.b64)
vshell_typing 'mv -v logo.png data.bin docs
mv -v docs/faq.md FAQ.md
ls -l . docs
' -vfs vfs/multi.b64
check_unchanged vfs/multi.b64 "$sum"

section 'Renaming the only file of the minimal VFS, then starting over from disk'
sum=$(cksum < vfs/minimal.b64)
vshell_typing 'mv -v readme.txt README
ls
' -vfs vfs/minimal.b64
vshell_typing 'ls
' -vfs vfs/minimal.b64
check_unchanged vfs/minimal.b64 "$sum"
