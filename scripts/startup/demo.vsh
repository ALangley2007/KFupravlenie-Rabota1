// Startup script demo: comments and quoting, run against the deep VFS.
// Comments use Go syntax: "//" starts a comment that runs to the end of the line.

cd /home/guest
ls
ls "my documents" 'my documents' my\ documents   // three ways to keep a blank
cd "my documents"  // a trailing comment is ignored too
ls -l report.txt
cd ..

// "//" opens a comment only at the start of a word, so these stay arguments:
ls /usr//bin/ls "// not a comment"
