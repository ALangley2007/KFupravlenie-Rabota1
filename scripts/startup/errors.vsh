// Every error is reported with the script name and line number,
// and the script goes on with the next line.
foo --bar
ls "unterminated
ls trailing\
exit 1 2
ls still running
// A non-numeric exit status ends the session with status 2, as in bash.
exit abc
ls never runs
