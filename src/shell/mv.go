package shell

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"vshell/src/vfs"
)

const (
	mvOptions    = "fnv"
	mvForce      = 'f'
	mvNoClobber  = 'n'
	mvVerbose    = "v"
	minMvOperand = 2
)

// ErrMissingOperand is returned by mv without a source or a destination.
var ErrMissingOperand = errors.New("missing file operand")

// mvOptionSet holds the options of one mv run.
type mvOptionSet struct {
	noClobber bool
	verbose   bool
}

// mvMove is one source to move and where it goes.
type mvMove struct {
	source string
	node   *vfs.Node
	dest   string
	dir    *vfs.Node
	name   string
}

// mvCommand implements mv [-fnv] source dest and mv [-fnv] source... dir like GNU mv,
// changing the VFS in memory only. With two operands where the last one is not an
// existing directory, it renames source to dest, replacing a file there; otherwise it
// moves every source into the directory under its own name. -n skips sources whose
// destination exists, -f undoes an earlier -n, and -v prints each move. Every source
// that cannot be moved is reported and the others are still moved; any failure, a
// missing operand or an invalid option sets status 1.
func mvCommand(s *Shell, args []string) error {
	letters, operands, err := splitOptions(args, mvOptions)
	if err != nil {
		return err
	}
	if len(operands) < minMvOperand {
		return missingOperand(operands)
	}
	opts := mvOptionSet{
		noClobber: strings.LastIndexByte(letters, mvNoClobber) > strings.LastIndexByte(letters, mvForce),
		verbose:   strings.Contains(letters, mvVerbose),
	}
	sources, target := operands[:len(operands)-1], operands[len(operands)-1]
	targetDir, err := s.lookup(target)
	intoDir := err == nil && targetDir.IsDir()
	if len(sources) >= minMvOperand && !intoDir {
		return targetError(target, err)
	}
	failed := false
	for _, source := range sources {
		if err := s.moveOne(source, target, targetDir, intoDir, opts); err != nil {
			s.warn(fmt.Errorf("mv: %w", err))
			failed = true
		}
	}
	if failed {
		return errReported
	}
	return nil
}

// missingOperand returns the error GNU mv gives for no operands or a single one.
func missingOperand(operands []string) error {
	if len(operands) == 0 {
		return ErrMissingOperand
	}
	return fmt.Errorf("missing destination file operand after '%s'", operands[0])
}

// targetError returns the error for a target of several sources that is not a
// directory: lookupErr when it could not be found, as GNU mv words them.
func targetError(target string, lookupErr error) error {
	if lookupErr != nil {
		return fmt.Errorf("target '%s': %w", target, lookupErr)
	}
	return fmt.Errorf("target '%s' is not a directory", target)
}

// moveOne moves source to target, or into targetDir when intoDir is set, following opts.
// A source whose destination exists is skipped silently with -n.
func (s *Shell) moveOne(source, target string, targetDir *vfs.Node, intoDir bool, opts mvOptionSet) error {
	move, err := s.planMove(source, target, targetDir, intoDir)
	if err != nil {
		return err
	}
	node := move.node
	existing, _ := s.fsys.LookupFrom(move.dir, move.name)
	switch {
	case existing == node:
		return fmt.Errorf("'%s' and '%s' are the same file", source, move.dest)
	case existing != nil && opts.noClobber:
		return nil
	}
	if err := s.fsys.Rename(node, move.dir, move.name); err != nil {
		return renameError(move, err)
	}
	if opts.verbose {
		_, err = fmt.Fprintf(s.out, "renamed '%s' -> '%s'\n", source, move.dest)
	}
	return err
}

// planMove looks up source and works out the directory, name and displayed path it gets
// when moved to target, or into targetDir under its own name when intoDir is set. The
// root and a source ending in "." or ".." cannot be moved, as on a UNIX system.
func (s *Shell) planMove(source, target string, targetDir *vfs.Node, intoDir bool) (mvMove, error) {
	node, err := s.lookup(source)
	if err != nil {
		return mvMove{}, fmt.Errorf("cannot stat '%s': %w", source, err)
	}
	if node == s.fsys.Root() || isDotName(source) {
		return mvMove{}, fmt.Errorf("cannot move '%s' to '%s': %w", source, target, vfs.ErrBusy)
	}
	move := mvMove{source: source, node: node}
	if !intoDir {
		return move, s.resolveDest(&move, target)
	}
	move.dir, move.name = targetDir, node.Name()
	move.dest = strings.TrimSuffix(target, pathSeparator) + pathSeparator + node.Name()
	return move, nil
}

// resolveDest fills in the directory and the new name for renaming move.node to target,
// whose parent directory must exist. A trailing slash on target requires a directory.
func (s *Shell) resolveDest(move *mvMove, target string) error {
	move.dest = target
	trimmed := strings.TrimRight(target, pathSeparator)
	if trimmed != target && !move.node.IsDir() {
		return fmt.Errorf("cannot move '%s' to '%s': %w", move.source, target, vfs.ErrNotDir)
	}
	parent, name := path.Split(trimmed)
	dir, err := s.lookup(parent)
	if err == nil && !dir.IsDir() {
		err = vfs.ErrNotDir
	}
	if err != nil {
		return fmt.Errorf("cannot move '%s' to '%s': %w", move.source, target, err)
	}
	move.dir, move.name = dir, name
	return nil
}

// isDotName reports whether source names "." or ".." as its last element, which no
// directory entry can be renamed through.
func isDotName(source string) bool {
	base := path.Base(source)
	return base == currentDirName || base == parentDirName
}

// renameError turns the error vfs.Rename gave for move into the message GNU mv prints.
func renameError(move mvMove, err error) error {
	switch {
	case errors.Is(err, vfs.ErrInvalidMove):
		return fmt.Errorf("cannot move '%s' to a subdirectory of itself, '%s'", move.source, move.dest)
	case errors.Is(err, vfs.ErrIsDir):
		return fmt.Errorf("cannot overwrite directory '%s' with non-directory", move.dest)
	case errors.Is(err, vfs.ErrNotDir):
		return fmt.Errorf("cannot overwrite non-directory '%s' with directory '%s'", move.dest, move.source)
	default:
		return fmt.Errorf("cannot move '%s' to '%s': %w", move.source, move.dest, err)
	}
}
