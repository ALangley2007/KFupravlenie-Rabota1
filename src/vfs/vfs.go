// Package vfs implements the virtual file system of the emulator: a tree of
// directories and files held entirely in memory.
package vfs

import (
	"errors"
	"sort"
	"strings"
)

// Root is the path of the root directory.
const Root = "/"

const (
	separator = "/"
	current   = "."
	parent    = ".."
)

// ErrNotExist is returned when a path names nothing in the file system.
var ErrNotExist = errors.New("no such file or directory")

// ErrNotDir is returned when a path goes through a file as if it were a directory.
var ErrNotDir = errors.New("not a directory")

// Node is a directory or a file of the virtual file system.
type Node struct {
	name     string
	data     []byte
	children map[string]*Node
}

// FS is a virtual file system: a tree of nodes under the root directory.
type FS struct {
	root *Node
}

// New returns an empty file system that holds only the root directory.
func New() *FS {
	return &FS{root: newDir(Root)}
}

// newDir returns an empty directory called name.
func newDir(name string) *Node {
	return &Node{name: name, children: map[string]*Node{}}
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

// Lookup returns the node at name, a slash-separated path followed from the root,
// so a leading slash is optional. Like a UNIX kernel, it resolves "." and ".." one
// element at a time, keeps ".." at the root, and fails with ErrNotDir when any element
// but the last one, or a last one followed by a slash, is a file. A missing element
// gives ErrNotExist.
func (f *FS) Lookup(name string) (*Node, error) {
	path := []*Node{f.root}
	for _, part := range strings.Split(name, separator) {
		node := path[len(path)-1]
		if !node.IsDir() {
			return nil, ErrNotDir
		}
		switch part {
		case "", current:
		case parent:
			path = path[:max(1, len(path)-1)]
		default:
			child, ok := node.children[part]
			if !ok {
				return nil, ErrNotExist
			}
			path = append(path, child)
		}
	}
	return path[len(path)-1], nil
}
