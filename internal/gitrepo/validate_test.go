package gitrepo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rp/internal/gitrepo"
	"rp/internal/model"
)

func TestValidateGoalGitReposRequiresIndependentRepo(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "project")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", parent, "init").Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	cfg := model.Config{
		Resources: map[string]model.Resource{
			"repo": {
				Type: "GitRepo",
				Realizations: []model.Realization{{
					Kind: "local_path",
					URI:  "file://.",
				}},
			},
		},
	}
	goal := model.Goal{Given: map[string]string{"repo": "repo"}}
	err := gitrepo.ValidateGoalGitRepos(child, cfg, goal)
	if err == nil {
		t.Fatal("expected error when project uses parent git repository")
	}
}

func TestValidateGoalGitReposAcceptsLocalInit(t *testing.T) {
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	cfg := model.Config{
		Resources: map[string]model.Resource{
			"repo": {
				Type: "GitRepo",
				Realizations: []model.Realization{{
					Kind: "local_path",
					URI:  "file://.",
				}},
			},
		},
	}
	goal := model.Goal{Given: map[string]string{"repo": "repo"}}
	if err := gitrepo.ValidateGoalGitRepos(dir, cfg, goal); err != nil {
		t.Fatalf("expected local git repo to pass: %v", err)
	}
}
