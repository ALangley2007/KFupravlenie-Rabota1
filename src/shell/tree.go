package shell

import (
	"fmt"
	"io"
	"strings"

	"vshell/src/vfs"
)

const (
	treeDefaultDir = "."
	treeBranch     = "├── "
	treeLastBranch = "└── "
	treeIndent     = "│   "
	treeLastIndent = "    "
	singularCount  = 1
)

// treeCounts tallies the directories and files a tree listing shows.
type treeCounts struct {
	dirs  int
	files int
}

// treeCommand implements the tree [path] utility: it prints path, the current directory
// by default, and everything below it as an indented tree sorted by name, followed by
// the number of directories and files shown, like the UNIX tree command. A file is
// shown on its own. A missing path or one going through a file is an error.
func treeCommand(s *Shell, args []string) error {
	if len(args) > maxTreeArgs {
		return ErrTooManyArguments
	}
	target := treeDefaultDir
	if len(args) > 0 {
		target = args[0]
	}
	node, err := s.fsys.Lookup(s.resolve(target))
	if err != nil {
		return fmt.Errorf("%s: %w", target, err)
	}
	var listing strings.Builder
	var counts treeCounts
	listing.WriteString(target + "\n")
	if node.IsDir() {
		writeTree(&listing, node, "", &counts)
	} else {
		counts.files++
	}
	fmt.Fprintf(&listing, "\n%s, %s\n",
		plural(counts.dirs, "directory", "directories"), plural(counts.files, "file", "files"))
	_, err = io.WriteString(s.out, listing.String())
	return err
}

// writeTree writes one line per entry below dir, each starting with prefix and a branch,
// descends into subdirectories with a deeper prefix, and adds every entry to counts.
func writeTree(w *strings.Builder, dir *vfs.Node, prefix string, counts *treeCounts) {
	children := dir.Children()
	for i, child := range children {
		branch, indent := treeBranch, treeIndent
		if i == len(children)-1 {
			branch, indent = treeLastBranch, treeLastIndent
		}
		w.WriteString(prefix + branch + child.Name() + "\n")
		if !child.IsDir() {
			counts.files++
			continue
		}
		counts.dirs++
		writeTree(w, child, prefix+indent, counts)
	}
}

// plural returns count followed by the singular noun for one and the plural noun otherwise.
func plural(count int, singular, many string) string {
	if count == singularCount {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, many)
}
