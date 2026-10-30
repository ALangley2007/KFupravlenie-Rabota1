// Stage 4 startup script: every mode of ls, cd, uname and clear, run against the
// deep VFS (vfs/deep.b64, see scripts/test_commands.sh). Everything stays in memory.

// clear -x clears the screen but keeps the scrollback, so earlier output stays reachable.
clear -x

// uname prints data of the real host: the kernel name by default, any mix of
// -s -n -r -v -m in a fixed order, or everything with -a.
uname
uname -s
uname -n
uname -r
uname -v
uname -m
uname -sm
uname -m -s
uname -a
// uname errors: an unknown option and an operand.
uname -x
uname extra

// ls lists the current directory by default; the session starts in / since this
// VFS has no home directory for the host user.
ls
// Absolute and relative paths, a file, several operands with headers.
ls /home/guest
ls home/guest/projects/vshell
ls /etc/motd
ls /etc /var/log /etc/hostname
// A crowded directory is laid out in columns.
ls /usr/bin
// -a shows hidden entries with "." and ".."; -l is the long format; options
// combine and may follow operands; "--" ends the options.
ls -a /home/guest
ls -l /home/guest
ls -la /home/guest
ls /home/guest -l -a
ls -l /etc/hostname /etc/motd
ls -l -- /var
// Names with blanks need quoting.
ls "/home/guest/my documents" '/home/guest/my documents' /home/guest/my\ documents
// ls errors: missing paths are reported one by one while the rest is still listed,
// a file used as a directory, an unknown option, a long option.
ls /nope /etc /missing
ls /etc/motd/
ls -z
ls --all

// cd moves through the VFS and the prompt follows.
cd /home/guest
cd projects/vshell/src
cd ..
cd ../../..
cd ./guest/./projects/../../guest
cd "my documents"
ls -l
// "cd -" goes back and prints where it went.
cd -
cd -
// ".." stops at the root.
cd /../../..
// Plain cd and ~ go to /home/<user>, which this VFS does not have.
cd
cd ~
cd ~/projects
// cd errors: a missing directory, a file, too many arguments, an option.
cd /nope
cd /etc/motd
cd /etc /var
cd -L
// A failed cd keeps the current directory.
ls

// tree, the stage 3 utility, sees the same VFS and hides dotfiles.
tree /home/guest

exit
