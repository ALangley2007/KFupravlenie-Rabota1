// Run against a VFS that holds the host user's home directory (scripts/test_commands.sh
// builds one), so the session starts there and the prompt shows "~".
ls
cd docs
cd ..
cd /
cd
cd ~/docs
ls ~
ls "~"  // a quoted tilde is not expanded
cd -
exit
