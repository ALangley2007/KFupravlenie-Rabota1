// Stage 3 startup script: every command so far, run against the deep VFS
// (vfs/deep.b64, see scripts/test_vfs_deep.sh). The VFS lives in memory only.

// tree lists the current directory by default, here the root.
tree
// Absolute and relative paths, "." and "..", a trailing slash, quoting.
tree /home/guest
tree home/guest/projects
tree ./var/../etc
tree "/home/guest/projects/vshell/src/"
tree /..
// A file is shown on its own.
tree /etc/motd
// An empty directory.
tree /tmp

// Errors: a missing path, a file used as a directory, too many arguments.
tree /nope
tree /etc/motd/
tree /etc/motd/..
tree /etc /var

// ls and cd are still stubs that print their arguments (stage 4 implements them).
ls -l /home/guest
cd "/home/guest/my projects"

// Errors from the earlier stages.
foo --bar
ls "unterminated
exit 1 2

// exit ends the session with the given status.
exit 0
