package shell

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vshell/src/vfs"
)

const (
	lsOptions      = "al"
	optionAll      = "a"
	optionLong     = "l"
	hiddenPrefix   = "."
	terminalWidth  = 80
	columnGap      = "  "
	dirDisplaySize = 4096
	blockBytes     = 4096
	blockKiB       = 4
	recentLayout   = "Jan _2 15:04"
	oldLayout      = "Jan _2  2006"
	sizeBase       = 10
	singleOperand  = 1
)

// recentPeriod is how old a time may be to show its clock instead of its year: half of
// an average Gregorian year, as GNU ls uses.
const recentPeriod = 15778476 * time.Second

// lsEntry is one name ls shows and the node it stands for.
type lsEntry struct {
	name string
	node *vfs.Node
}

// listing collects the output of one ls run with its options.
type listing struct {
	all   bool
	long  bool
	owner string
	group string
	now   time.Time
	out   strings.Builder
}

// lsCommand implements ls [-al] [path...] like GNU ls: it lists the contents of each
// directory and the name of each file given, the current directory by default. Files
// come first, then directories, each sorted by name, and directories get a "name:"
// header when there are several operands. -a also shows hidden entries with "." and
// "..", and -l the long format. Paths that cannot be accessed are reported one by one
// and set status 2, as does an invalid option.
func lsCommand(s *Shell, args []string) error {
	letters, operands, err := splitOptions(args, lsOptions)
	if err != nil {
		return &ExitError{Status: StatusMisuse, Err: err}
	}
	if len(operands) == 0 {
		operands = []string{currentDirName}
	}
	files, dirs, ok := s.statOperands(operands)
	ls := &listing{
		all:   strings.Contains(letters, optionAll),
		long:  strings.Contains(letters, optionLong),
		owner: s.host.User,
		group: s.host.Group,
		now:   time.Now(),
	}
	ls.writeEntries(files)
	ls.writeDirs(dirs, len(files) > 0, len(operands) > singleOperand)
	if _, err := io.WriteString(s.out, ls.out.String()); err != nil {
		return err
	}
	if !ok {
		return &ExitError{Status: StatusMisuse, Err: errReported}
	}
	return nil
}

// statOperands looks up every operand, reporting the ones that cannot be accessed, and
// splits the others into files and directories, each sorted by name. ok is false when
// any operand failed.
func (s *Shell) statOperands(operands []string) (files, dirs []lsEntry, ok bool) {
	ok = true
	for _, name := range operands {
		node, err := s.lookup(name)
		switch {
		case err != nil:
			s.warn(fmt.Errorf("ls: cannot access '%s': %w", name, err))
			ok = false
		case node.IsDir():
			dirs = append(dirs, lsEntry{name, node})
		default:
			files = append(files, lsEntry{name, node})
		}
	}
	sortEntries(files)
	sortEntries(dirs)
	return files, dirs, ok
}

// sortEntries orders entries by name, byte by byte, as ls does in the C locale.
func sortEntries(entries []lsEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
}

// writeDirs writes the contents of each directory, separated by blank lines from each
// other and from the file names written before, and headed by the directory name when
// headers is set.
func (ls *listing) writeDirs(dirs []lsEntry, afterFiles, headers bool) {
	for i, dir := range dirs {
		if afterFiles || i > 0 {
			ls.out.WriteString("\n")
		}
		if headers {
			ls.out.WriteString(dir.name + ":\n")
		}
		ls.writeDir(dir.node)
	}
}

// writeDir writes the entries of dir: hidden ones only with -a, which also adds "." and
// ".." first, and in long format after a line with the total number of blocks.
func (ls *listing) writeDir(dir *vfs.Node) {
	var entries []lsEntry
	if ls.all {
		entries = append(entries, lsEntry{currentDirName, dir}, lsEntry{parentDirName, dir.Parent()})
	}
	for _, child := range dir.Children() {
		if ls.all || !strings.HasPrefix(child.Name(), hiddenPrefix) {
			entries = append(entries, lsEntry{child.Name(), child})
		}
	}
	if ls.long {
		fmt.Fprintf(&ls.out, "total %d\n", totalBlocks(entries))
	}
	ls.writeEntries(entries)
}

// writeEntries writes entries in long format with -l, otherwise in columns.
func (ls *listing) writeEntries(entries []lsEntry) {
	if len(entries) == 0 {
		return
	}
	if ls.long {
		ls.writeLong(entries)
		return
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.name
	}
	ls.out.WriteString(formatColumns(names, terminalWidth))
}

// writeLong writes one line per entry with its mode, link count, owner, group, size,
// modification time and name, aligning every column across the lines.
func (ls *listing) writeLong(entries []lsEntry) {
	rows := make([][]string, len(entries))
	var links, owner, group, size int
	for i, entry := range entries {
		rows[i] = ls.longFields(entry)
		links = max(links, utf8.RuneCountInString(rows[i][1]))
		owner = max(owner, utf8.RuneCountInString(rows[i][2]))
		group = max(group, utf8.RuneCountInString(rows[i][3]))
		size = max(size, utf8.RuneCountInString(rows[i][4]))
	}
	for _, row := range rows {
		fmt.Fprintf(&ls.out, "%s %*s %-*s %-*s %*s %s %s\n",
			row[0], links, row[1], owner, row[2], group, row[3], size, row[4], row[5], row[6])
	}
}

// longFields returns the long-format columns for entry. Directories show 4096 bytes,
// the size of one block, as on ext4.
func (ls *listing) longFields(entry lsEntry) []string {
	size := entry.node.Size()
	if entry.node.IsDir() {
		size = dirDisplaySize
	}
	return []string{
		entry.node.Mode().String(),
		strconv.Itoa(entry.node.Links()),
		ls.owner,
		ls.group,
		strconv.FormatInt(size, sizeBase),
		ls.formatTime(entry.node.ModTime()),
		entry.name,
	}
}

// formatTime returns t with its clock when it lies within the last six months and with
// its year when it is older or in the future, as ls does.
func (ls *listing) formatTime(t time.Time) string {
	if t.After(ls.now) || ls.now.Sub(t) > recentPeriod {
		return t.Format(oldLayout)
	}
	return t.Format(recentLayout)
}

// totalBlocks returns the disk usage of entries in KiB, as the "total" line of ls -l shows.
func totalBlocks(entries []lsEntry) int64 {
	var total int64
	for _, entry := range entries {
		total += blocks(entry.node)
	}
	return total
}

// blocks returns the disk usage of node in KiB as if it were stored in 4 KiB blocks:
// one block for a directory and as many as its content needs for a file.
func blocks(node *vfs.Node) int64 {
	if node.IsDir() {
		return blockKiB
	}
	return (node.Size() + blockBytes - 1) / blockBytes * blockKiB
}

// formatColumns lays names out in columns filled from top to bottom, as ls does on
// a terminal: it uses the fewest rows whose columns, two spaces apart, fit in width.
// Names too wide for any layout end up one per line.
func formatColumns(names []string, width int) string {
	rows := 1
	for rows < len(names) && layoutWidth(columnWidths(names, rows)) > width {
		rows++
	}
	widths := columnWidths(names, rows)
	var out strings.Builder
	for row := range rows {
		for col := range widths {
			i := col*rows + row
			if i >= len(names) {
				break
			}
			out.WriteString(names[i])
			if i+rows < len(names) {
				out.WriteString(strings.Repeat(" ", widths[col]-utf8.RuneCountInString(names[i])) + columnGap)
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

// columnWidths returns the width of every column when names fill rows lines per column.
func columnWidths(names []string, rows int) []int {
	widths := make([]int, (len(names)+rows-1)/rows)
	for i, name := range names {
		widths[i/rows] = max(widths[i/rows], utf8.RuneCountInString(name))
	}
	return widths
}

// layoutWidth returns the width of a line holding columns of the given widths.
func layoutWidth(widths []int) int {
	total := len(columnGap) * (len(widths) - 1)
	for _, width := range widths {
		total += width
	}
	return total
}
