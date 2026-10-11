package tests

import (
	"testing"

	"vshell/src/shell"
)

var promptCases = []struct {
	name string
	host shell.HostInfo
	dir  string
	want string
}{
	{"home directory", testHost, "/home/alice", "alice@wonderland:~$ "},
	{"inside home", testHost, "/home/alice/work/src", "alice@wonderland:~/work/src$ "},
	{"outside home", testHost, "/etc", "alice@wonderland:/etc$ "},
	{"similar prefix", testHost, "/home/alice2", "alice@wonderland:/home/alice2$ "},
	{"unknown home", shell.HostInfo{User: "bob", Host: "pc"}, "/tmp", "bob@pc:/tmp$ "},
	{"superuser", shell.HostInfo{User: "root", Host: "pc", Home: "/root", Root: true}, "/root", "root@pc:~# "},
}

func TestPrompt(t *testing.T) {
	for _, tc := range promptCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.host.Prompt(tc.dir); got != tc.want {
				t.Errorf("Prompt(%q) = %q, want %q", tc.dir, got, tc.want)
			}
		})
	}
}

func TestDetectHost(t *testing.T) {
	host := shell.DetectHost()
	if host.User == "" {
		t.Error("DetectHost returned an empty user name")
	}
	if host.Host == "" {
		t.Error("DetectHost returned an empty host name")
	}
	if host.WorkingDir() == "" {
		t.Error("WorkingDir returned an empty path")
	}
}
