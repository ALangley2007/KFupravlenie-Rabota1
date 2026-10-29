// Startup script demo: comments, quoting and the stub commands.
// Comments use Go syntax: "//" starts a comment that runs to the end of the line.

ls
ls -l "my dir" 'single quoted' escaped\ space
cd "/usr/local/my folder"  // a trailing comment is ignored too
cd a"b c"d "" 'it"s'

// "//" opens a comment only at the start of a word, so these stay arguments:
ls a//b "// double" '// single' \//escaped
