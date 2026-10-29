// Package config reads the emulator parameters from the command line.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

const (
	vfsFlag      = "vfs"
	scriptFlag   = "script"
	debugPrefix  = "[debug]"
	notSetMarker = "(not set)"
)

// ErrUnexpectedArgument is returned by Parse when a positional argument follows the flags.
var ErrUnexpectedArgument = errors.New("unexpected argument")

// Options holds the emulator parameters given on the command line.
type Options struct {
	// VFSPath is the physical location of the virtual file system.
	VFSPath string
	// ScriptPath is the startup script run before the interactive session.
	ScriptPath string
}

// Parse reads Options from args, the command-line arguments without the program name.
// Flags may be written as -name value, -name=value or with a double dash; a flag left
// out keeps an empty value. Errors and the usage text go to errOut under the program
// name. It returns flag.ErrHelp for -h and -help, and ErrUnexpectedArgument for
// positional arguments.
func Parse(name string, args []string, errOut io.Writer) (Options, error) {
	var opts Options
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(errOut)
	flags.StringVar(&opts.VFSPath, vfsFlag, "", "`path` to the physical location of the VFS")
	flags.StringVar(&opts.ScriptPath, scriptFlag, "", "`path` to the startup script")
	flags.Usage = func() { usage(flags) }
	if err := flags.Parse(args); err != nil {
		return Options{}, err
	}
	if flags.NArg() > 0 {
		err := fmt.Errorf("%w: %s", ErrUnexpectedArgument, flags.Arg(0))
		fmt.Fprintln(errOut, err)
		flags.Usage()
		return Options{}, err
	}
	return opts, nil
}

// usage prints the synopsis of the program followed by the description of every flag.
func usage(flags *flag.FlagSet) {
	fmt.Fprintf(flags.Output(), "Usage: %s [-%s path] [-%s path]\n", flags.Name(), vfsFlag, scriptFlag)
	flags.PrintDefaults()
}

// WriteDebug prints every parameter with its value to w, one per line, so each launch
// shows exactly how the emulator was configured. Values are quoted to make spaces
// visible; parameters left out are marked as not set.
func (o Options) WriteDebug(w io.Writer) {
	fmt.Fprintf(w, "%s %s: %s\n", debugPrefix, vfsFlag, describe(o.VFSPath))
	fmt.Fprintf(w, "%s %s: %s\n", debugPrefix, scriptFlag, describe(o.ScriptPath))
}

// describe returns value quoted, or a not-set marker when value is empty.
func describe(value string) string {
	if value == "" {
		return notSetMarker
	}
	return strconv.Quote(value)
}
