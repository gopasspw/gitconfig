//go:build !windows

package gitconfig

import (
	"os"
	"os/exec"
	"path/filepath"
)

// SystemConfig is the location of the (optional) system-wide config defaults file.
var systemConfig = "/etc/gitconfig" // /etc/gopass/config

func init() {
	if gitPath, err := exec.LookPath("git"); err == nil {
		cand := filepath.Join(filepath.Dir(filepath.Dir(gitPath)), "etc", "gitconfig")
		if _, err := os.Stat(cand); err == nil {
			systemConfig = cand
		}
	}
}
