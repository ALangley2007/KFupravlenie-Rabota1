package shell

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	homeSymbol          = "~"
	pathSeparator       = "/"
	userPromptSign      = "$"
	rootPromptSign      = "#"
	rootUID             = "0"
	fallbackUser        = "user"
	fallbackHost        = "localhost"
	hostDomainSeparator = "."
	userDomainSeparator = `\`
)

// HostInfo describes the host machine data shown in the prompt.
type HostInfo struct {
	User string
	Host string
	Home string
	Root bool
}

// DetectHost reads the current user, home directory and host name from the real OS.
// Values that cannot be read fall back to "user" and "localhost"; the host name is
// cut at the first dot and a Windows DOMAIN\user name is reduced to the user part.
func DetectHost() HostInfo {
	info := HostInfo{User: fallbackUser, Host: fallbackHost}
	if current, err := user.Current(); err == nil {
		info.User = shortUserName(current.Username)
		info.Home = filepath.ToSlash(current.HomeDir)
		info.Root = current.Uid == rootUID
	}
	if home, err := os.UserHomeDir(); err == nil && info.Home == "" {
		info.Home = filepath.ToSlash(home)
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		info.Host = shortHostName(host)
	}
	return info
}

// Prompt returns the prompt for dir in the form "user@host:dir$ ", with "#" for root.
func (h HostInfo) Prompt(dir string) string {
	return fmt.Sprintf("%s@%s:%s%s ", h.User, h.Host, h.DisplayPath(dir), h.promptSign())
}

// DisplayPath returns dir as the prompt shows it: the home directory and paths inside
// it start with "~", other paths are kept unchanged.
func (h HostInfo) DisplayPath(dir string) string {
	if h.Home == "" {
		return dir
	}
	if dir == h.Home {
		return homeSymbol
	}
	prefix := strings.TrimSuffix(h.Home, pathSeparator) + pathSeparator
	if rest, ok := strings.CutPrefix(dir, prefix); ok {
		return homeSymbol + pathSeparator + rest
	}
	return dir
}

// promptSign returns "#" for the superuser and "$" for everyone else.
func (h HostInfo) promptSign() string {
	if h.Root {
		return rootPromptSign
	}
	return userPromptSign
}

// shortHostName returns host without its domain part, e.g. "pc" for "pc.local".
func shortHostName(host string) string {
	name, _, _ := strings.Cut(host, hostDomainSeparator)
	return name
}

// shortUserName returns name without a Windows domain prefix, e.g. "bob" for `CORP\bob`.
func shortUserName(name string) string {
	return name[strings.LastIndex(name, userDomainSeparator)+1:]
}
