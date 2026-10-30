// Stage 5 startup script: every mode of mv, run against the deep VFS (vfs/deep.b64,
// see scripts/test_mv.sh). All changes stay in memory: the archive on disk is untouched.

cd /home/guest
ls -a
// Rename a file; -v prints each move.
mv notes.txt todo.txt
mv -v todo.txt notes.txt
// Move a file into a directory under its own name, also given with a trailing slash.
mv notes.txt projects
mv -v projects/notes.txt "my documents/"
ls "my documents"
// Rename a directory, then move it with everything below it into another one.
mv -v projects work
mv -v work "my documents"
tree "my documents"
// Relative paths with "." and "..".
cd "my documents/work"
mv -v ../notes.txt .
mv -v notes.txt ../../notes.txt
cd /home/guest
// Several sources into one directory.
mv -v /usr/bin/ls /usr/bin/cat /usr/bin/mv /var/log
ls /usr/bin /var/log
// An existing file is replaced; -n keeps it, and a later -f undoes -n.
mv -n /etc/motd /etc/hostname
ls /etc
mv -nfv /etc/motd /etc/hostname
ls -l /etc
mv notes.txt "my documents/report.txt"
ls -l "my documents"
// A directory replaces only an empty directory: rename the empty .cache to "home",
// then to "tmp", and move it to the root, where /home has entries and /tmp has none.
mv -v .cache home
mv home /
mv -v home tmp
mv -v tmp /
ls -la /
// Moving a parent of the current directory: the prompt follows the new path.
cd "my documents/work/vshell"
mv -v "/home/guest/my documents" /home/guest/docs
cd /home/guest

// Errors: missing operands, sources that do not exist or are used as directories,
// a destination whose parent is missing or is a file.
mv
mv .profile
mv nope /tmp
mv .profile/ /tmp
mv .profile nope/x
mv .profile .profile/x
// A directory cannot go into itself, a file cannot be moved onto itself, and the root,
// "." and ".." cannot be moved at all.
mv docs docs
mv /home /home/guest/docs/work
mv .profile .
mv / /tmp
mv .. /tmp
// A directory cannot replace a file, nor a file a directory.
mv docs /etc/hostname
mv -v /etc/hostname /etc/docs
mv /etc/docs /home/guest
// Several sources need an existing directory as the target.
mv .profile docs /etc/docs
mv .profile docs /nope
// Unknown options.
mv -i .profile x
mv --force .profile x
// A failed source does not stop the others.
mv -v nope .profile /tmp
ls -a /tmp

tree /home
exit
