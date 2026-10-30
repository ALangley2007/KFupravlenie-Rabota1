package vfs

import (
	"errors"
	"strings"
	"time"
)

// ErrBusy is returned for an attempt to move the root directory.
var ErrBusy = errors.New("device or resource busy")

// ErrInvalidMove is returned for an attempt to move a directory into itself or below it.
var ErrInvalidMove = errors.New("invalid argument")

// ErrIsDir is returned for an attempt to replace a directory with a file.
var ErrIsDir = errors.New("is a directory")

// ErrNotEmpty is returned for an attempt to replace a directory that still has entries.
var ErrNotEmpty = errors.New("directory not empty")

// Rename moves node into the directory dir under name, in memory only, following the
// rules of rename(2): an existing file of that name is replaced by a file, an existing
// empty directory by a directory, and renaming a node onto itself does nothing. It fails
// with ErrBusy for the root, ErrInvalidMove when dir is node or lies below it, ErrIsDir
// or ErrNotDir when the kinds of node and the replaced entry differ, ErrNotEmpty for a
// directory with entries, and ErrBadName for a name that is not a single path element.
// The directories that lose and gain the entry are marked as modified now.
func (f *FS) Rename(node, dir *Node, name string) error {
	if err := checkRename(node, dir, name); err != nil {
		return err
	}
	existing := dir.children[name]
	if existing == node {
		return nil
	}
	if err := checkReplace(node, existing); err != nil {
		return err
	}
	now := time.Now()
	delete(node.parent.children, node.name)
	node.parent.modTime = now
	node.name, node.parent = name, dir
	dir.children[name] = node
	dir.modTime = now
	return nil
}

// checkRename reports why node cannot be moved into dir under name at all, whatever
// dir already holds.
func checkRename(node, dir *Node, name string) error {
	if name == "" || name == currentDir || name == parentDir || strings.Contains(name, separator) {
		return ErrBadName
	}
	if node.parent == nil {
		return ErrBusy
	}
	if !dir.IsDir() {
		return ErrNotDir
	}
	for ancestor := dir; ancestor != nil; ancestor = ancestor.parent {
		if ancestor == node {
			return ErrInvalidMove
		}
	}
	return nil
}

// checkReplace reports why node cannot take the place of existing, the entry it would
// replace, or nil when there is no such entry or the replacement is allowed.
func checkReplace(node, existing *Node) error {
	switch {
	case existing == nil:
		return nil
	case existing.IsDir() && !node.IsDir():
		return ErrIsDir
	case !existing.IsDir() && node.IsDir():
		return ErrNotDir
	case existing.IsDir() && len(existing.children) > 0:
		return ErrNotEmpty
	default:
		return nil
	}
}
