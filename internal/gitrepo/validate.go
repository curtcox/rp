// Package gitrepo validates that GitRepo resources are independent git
// repositories before they are used by the planner.
package gitrepo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"rp/internal/model"
)

// ValidateGoalGitRepos ensures GitRepo resources referenced by a goal resolve to
// independent git repository roots. This prevents git commands from silently
// targeting a parent repository when the project directory has no local .git.
func ValidateGoalGitRepos(root string, cfg model.Config, goal model.Goal) error {
	seen := map[string]bool{}
	for _, resName := range goal.Given {
		res, ok := cfg.Resources[resName]
		if !ok || res.Type != "GitRepo" || seen[resName] {
			continue
		}
		seen[resName] = true
		path := pathFromURI(root, realizationURI(res))
		if err := ensureRepoRoot(path); err != nil {
			return fmt.Errorf("resource %q at %s: %w (run git init in the project directory)", resName, path, err)
		}
	}
	return nil
}

func ensureRepoRoot(path string) error {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("not a git repository")
	}
	top, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return err
	}
	absPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		absPath, err = filepath.Abs(path)
		if err != nil {
			return err
		}
	}
	if top != absPath {
		return fmt.Errorf("git toplevel is %s, not %s", top, absPath)
	}
	return nil
}

func realizationURI(res model.Resource) string {
	if len(res.Realizations) == 0 {
		return ""
	}
	return res.Realizations[0].URI
}

func pathFromURI(root, uri string) string {
	if uri == "" {
		return root
	}
	if strings.HasPrefix(uri, "file://") {
		p := strings.TrimPrefix(uri, "file://")
		if p == "." {
			return root
		}
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(root, p)
	}
	return uri
}
