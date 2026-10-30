package shell

import (
	"fmt"
	"os"
	"os/user"
	"strings"
)

const (
	homeSymbol          = "~"
	pathSeparator       = "/"
	userPromptSign      = "$"
	rootPromptSign      = "#"
	rootUID             = "0"
	rootHome            = "/root"
	homesDir            = "/home/"
	fallbackUser        = "user"
	fallbackHost        = "localhost"
	hostDomainSeparator = "."
	userDomainSeparator = `\`
)

// HostInfo describes the user and the host machine the session belongs to.
type HostInfo struct {
	User  string
	Group string
	Host  string
	// Home is the user's home directory inside the VFS; empty selects the conventional
	// one, see DefaultHome.
	Home string
	Root bool
}

// DetectHost reads the current user, their primary group and the host name from the real
// OS. Values that cannot be read fall back to "user" and "localhost", and the group to the
// user name; the host name is cut at the first dot and a Windows DOMAIN\user name is
// reduced to the user part. Home is left empty, since the session lives in the VFS.
func DetectHost() HostInfo {
	info := HostInfo{User: fallbackUser, Host: fallbackHost}
	if current, err := user.Current(); err == nil {
		info.User = shortUserName(current.Username)
		info.Root = current.Uid == rootUID
		if group, err := user.LookupGroupId(current.Gid); err == nil {
			info.Group = group.Name
		}
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		info.Host = shortHostName(host)
	}
	if info.Group == "" {
		info.Group = info.User
	}
	return info
}

// DefaultHome returns the conventional home directory of the user on a UNIX system:
// /root for the superuser and /home/<user> for everyone else.
func (h HostInfo) DefaultHome() string {
	if h.Root {
		return rootHome
	}
	return homesDir + h.User
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
