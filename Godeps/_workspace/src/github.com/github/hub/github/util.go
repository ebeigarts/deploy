package github

import (
	"github.com/remind101/deploy/Godeps/_workspace/src/github.com/github/hub/git"
	"github.com/remind101/deploy/Godeps/_workspace/src/github.com/mattn/go-isatty"
)

func IsHttpsProtocol() bool {
	httpProcotol, _ := git.Config("hub.protocol")
	if httpProcotol == "https" {
		return true
	}

	httpClone, _ := git.Config("--bool hub.http-clone")
	if httpClone == "true" {
		return true
	}

	return false
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd)
}
