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

type HostInfo struct {
	User string
	Host string
	Home string
	Root bool
}

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

func (h HostInfo) WorkingDir() string {
	if dir, err := os.Getwd(); err == nil {
		return filepath.ToSlash(dir)
	}
	if h.Home != "" {
		return h.Home
	}
	return pathSeparator
}

func (h HostInfo) Prompt(dir string) string {
	return fmt.Sprintf("%s@%s:%s%s ", h.User, h.Host, h.DisplayPath(dir), h.promptSign())
}

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

func (h HostInfo) promptSign() string {
	if h.Root {
		return rootPromptSign
	}
	return userPromptSign
}

func shortHostName(host string) string {
	name, _, _ := strings.Cut(host, hostDomainSeparator)
	return name
}

func shortUserName(name string) string {
	return name[strings.LastIndex(name, userDomainSeparator)+1:]
}
