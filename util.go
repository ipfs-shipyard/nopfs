package nopfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// GetDenylistFiles returns a list of ".deny" files found in
// $XDG_CONFIG_HOME/ipfs/denylists and /etc/ipfs/denylists.  The files are
// sortered by their names in their respective folders.
func GetDenylistFiles() ([]string, error) {
	var denylistFiles []string

	// First, look for denylists in $XDG_CONFIG_HOME/ipfs/denylists
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = os.Getenv("HOME") + "/.config"
	}
	ipfsDenylistPath := filepath.Join(xdgConfigHome, "ipfs", "denylists")
	err := filepath.WalkDir(ipfsDenylistPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".deny" {
			denylistFiles = append(denylistFiles, path)
		}
		return nil
	})
	if !os.IsNotExist(err) && err != nil {
		return nil, fmt.Errorf("error walking %s: %w", ipfsDenylistPath, err)
	}

	// Then, look for denylists in /etc/ipfs/denylists
	etcDenylistPath := "/etc/ipfs/denylists"
	err = filepath.WalkDir(etcDenylistPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".deny" {
			denylistFiles = append(denylistFiles, path)
		}
		return nil
	})
	if !os.IsNotExist(err) && err != nil {
		return nil, fmt.Errorf("error walking %s: %w", etcDenylistPath, err)
	}

	return denylistFiles, nil
}

// cutPrefix imported from go1.20
func cutPrefix(s, prefix string) (after string, found bool) {
	if !strings.HasPrefix(s, prefix) {
		return s, false
	}
	return s[len(prefix):], true
}
