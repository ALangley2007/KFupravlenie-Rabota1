// Package vfs implements the virtual file system of the emulator: a tree of
// directories and files held entirely in memory.
package vfs

import (
	"errors"
	"io/fs"
	"slices"
	"sort"
	"strings"
	"time"
)

// Root is the path of the root directory.
const Root = "/"

const (
	separator  = "/"
	currentDir = "."
	parentDir  = ".."
)

// DefaultDirMode is the mode of directories an archive does not describe: the root and
// directories implied by file paths.
const DefaultDirMode = fs.ModeDir | 0o755

// ErrNotExist is returned when a path names nothing in the file system.
var ErrNotExist = errors.New("no such file or directory")

// ErrNotDir is returned when a path goes through a file as if it were a directory.
var ErrNotDir = errors.New("not a directory")

// Node is a directory or a file of the virtual file system.
type Node struct {
	name     string
	mode     fs.FileMode
	modTime  time.Time
	data     []byte
	parent   *Node
	children map[string]*Node
}

// FS is a virtual file system: a tree of nodes under the root directory.
type FS struct {
	root *Node
}

// New returns an empty file system that holds only the root directory, modified now.
func New() *FS {
	return &FS{root: newDir(Root, nil, time.Now())}
}

// newDir returns an empty directory called name inside parent, with the default mode.
func newDir(name string, parent *Node, modTime time.Time) *Node {
	return &Node{name: name, mode: DefaultDirMode, modTime: modTime, parent: parent, children: map[string]*Node{}}
}

// Root returns the root directory.
func (f *FS) Root() *Node {
	return f.root
}

// Name returns the last element of the node's path, or "/" for the root.
func (n *Node) Name() string {
	return n.name
}

// IsDir reports whether the node is a directory.
func (n *Node) IsDir() bool {
	return n.children != nil
}

// Size returns the length of a file's content in bytes, or 0 for a directory.
func (n *Node) Size() int64 {
	return int64(len(n.data))
}

// Mode returns the type and permission bits of the node, e.g. drwxr-xr-x or -rw-r--r--.
func (n *Node) Mode() fs.FileMode {
	return n.mode
}

// ModTime returns the time the node was last modified.
func (n *Node) ModTime() time.Time {
	return n.modTime
}

// Parent returns the directory that holds the node; the root is its own parent.
func (n *Node) Parent() *Node {
	if n.parent == nil {
		return n
	}
	return n.parent
}

// Path returns the absolute path of the node, such as "/home/guest", or "/" for the root.
func (n *Node) Path() string {
	var parts []string
	for node := n; node.parent != nil; node = node.parent {
		parts = append(parts, node.name)
	}
	slices.Reverse(parts)
	return Root + strings.Join(parts, separator)
}

// Links returns the number of hard links to the node as a UNIX file system counts them:
// one for a file, and for a directory two (its name and ".") plus one per subdirectory,
// whose ".." points back to it.
func (n *Node) Links() int {
	if !n.IsDir() {
		return 1
	}
	links := 2
	for _, child := range n.children {
		if child.IsDir() {
			links++
		}
	}
	return links
}

// Children returns the entries of a directory sorted by name, or nil for a file.
func (n *Node) Children() []*Node {
	if !n.IsDir() {
		return nil
	}
	children := make([]*Node, 0, len(n.children))
	for _, child := range n.children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool { return children[i].name < children[j].name })
	return children
}

// Lookup returns the node at name followed from the root; see LookupFrom.
func (f *FS) Lookup(name string) (*Node, error) {
	return f.LookupFrom(f.root, name)
}

// LookupFrom returns the node at the slash-separated path name, taken relative to dir,
// or to the root when name starts with a slash. Like a UNIX kernel, it resolves "." and
// ".." one element at a time, keeps ".." at the root, and fails with ErrNotDir when any
// element but the last one, or a last one followed by a slash, is a file. A missing
// element gives ErrNotExist. An empty name stays in dir.
func (f *FS) LookupFrom(dir *Node, name string) (*Node, error) {
	node := dir
	if strings.HasPrefix(name, separator) {
		node = f.root
	}
	for _, part := range strings.Split(name, separator) {
		if !node.IsDir() {
			return nil, ErrNotDir
		}
		switch part {
		case "", currentDir:
		case parentDir:
			node = node.Parent()
		default:
			child, ok := node.children[part]
			if !ok {
				return nil, ErrNotExist
			}
			node = child
		}
	}
	return node, nil
}
