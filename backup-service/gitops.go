package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runGit executes a git command in dir (empty dir = current process cwd) and
// returns combined output.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Never prompt for credentials interactively.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// authURL injects the token into an https GitHub URL for non-interactive auth.
func authURL(repoURL, token string) string {
	const httpsPrefix = "https://"
	if strings.HasPrefix(repoURL, httpsPrefix) && !strings.Contains(repoURL, "@") {
		return httpsPrefix + "x-access-token:" + token + "@" + strings.TrimPrefix(repoURL, httpsPrefix)
	}
	return repoURL
}

// prepareRepo ensures a working clone exists at cfg.RepoWorkdir, refreshes
// remotes, and checks out the target branch (creating it if necessary).
func prepareRepo(cfg Config, log *Logger) error {
	gitDir := filepath.Join(cfg.RepoWorkdir, ".git")
	url := authURL(cfg.GitHubRepoURL, cfg.GitHubToken)

	if _, err := os.Stat(gitDir); err != nil {
		// Not a repo yet. If the path exists but is dirty, reset it.
		if entries, derr := os.ReadDir(cfg.RepoWorkdir); derr == nil && len(entries) > 0 {
			if rerr := os.RemoveAll(cfg.RepoWorkdir); rerr != nil {
				return fmt.Errorf("reset workdir: %w", rerr)
			}
		}
		if err := os.MkdirAll(filepath.Dir(cfg.RepoWorkdir), 0o755); err != nil {
			return err
		}
		if out, err := runGit("", "clone", url, cfg.RepoWorkdir); err != nil {
			return fmt.Errorf("clone repo: %w: %s", err, out)
		}
		log.Infof("cloned repository into %s", cfg.RepoWorkdir)
	}

	// Keep origin URL fresh so a rotated token takes effect.
	if out, err := runGit(cfg.RepoWorkdir, "remote", "set-url", "origin", url); err != nil {
		return fmt.Errorf("set remote: %w: %s", err, out)
	}
	if _, err := runGit(cfg.RepoWorkdir, "config", "user.name", cfg.GitAuthorName); err != nil {
		return err
	}
	if _, err := runGit(cfg.RepoWorkdir, "config", "user.email", cfg.GitAuthorEmail); err != nil {
		return err
	}

	// Best-effort fetch (an empty remote has nothing to fetch).
	if out, err := runGit(cfg.RepoWorkdir, "fetch", "origin"); err != nil {
		log.Warnf("git fetch reported: %s", strings.TrimSpace(out))
	}

	return checkoutBranch(cfg)
}

func checkoutBranch(cfg Config) error {
	branch := cfg.GitHubBranch
	// Remote already has the branch -> track it.
	if _, err := runGit(cfg.RepoWorkdir, "ls-remote", "--exit-code", "--heads", "origin", branch); err == nil {
		if out, err := runGit(cfg.RepoWorkdir, "checkout", "-B", branch, "origin/"+branch); err != nil {
			return fmt.Errorf("checkout tracking branch: %w: %s", err, out)
		}
		return nil
	}
	// Local repo has commits -> branch off current HEAD.
	if _, err := runGit(cfg.RepoWorkdir, "rev-parse", "--verify", "HEAD"); err == nil {
		if out, err := runGit(cfg.RepoWorkdir, "checkout", "-B", branch); err != nil {
			return fmt.Errorf("create branch: %w: %s", err, out)
		}
		return nil
	}
	// Empty repository (unborn HEAD) -> start an orphan branch.
	if out, err := runGit(cfg.RepoWorkdir, "checkout", "--orphan", branch); err != nil {
		return fmt.Errorf("create orphan branch: %w: %s", err, out)
	}
	return nil
}

// commitAndPush replaces the repo's tracked files with the freshly extracted
// code, commits on the backup branch, tags the snapshot, and pushes both.
func commitAndPush(cfg Config, codeDir, tag string, meta Metadata, log *Logger) error {
	if err := clearDirExcept(cfg.RepoWorkdir, ".git"); err != nil {
		return fmt.Errorf("clear repo: %w", err)
	}
	if err := copyTree(codeDir, cfg.RepoWorkdir); err != nil {
		return fmt.Errorf("copy code: %w", err)
	}

	if out, err := runGit(cfg.RepoWorkdir, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}

	msg := fmt.Sprintf("backup %s\n\nsource-commit: %s\nsource-date: %s\nschema: %s",
		tag, meta.GitCommit, meta.CreatedAt.Format("2006-01-02T15:04:05Z"), meta.SchemaVersion)
	// --allow-empty so a no-change cycle is still recorded (TZ §5.8).
	if out, err := runGit(cfg.RepoWorkdir, "commit", "--allow-empty", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	if out, err := runGit(cfg.RepoWorkdir, "tag", tag); err != nil {
		return fmt.Errorf("git tag: %w: %s", err, out)
	}
	if out, err := runGit(cfg.RepoWorkdir, "push", "origin", cfg.GitHubBranch); err != nil {
		return fmt.Errorf("git push branch: %w: %s", err, out)
	}
	if out, err := runGit(cfg.RepoWorkdir, "push", "origin", tag); err != nil {
		return fmt.Errorf("git push tag: %w: %s", err, out)
	}
	log.Infof("pushed branch %s and tag %s", cfg.GitHubBranch, tag)
	return nil
}

// clearDirExcept removes every entry in dir except the named one.
func clearDirExcept(dir, except string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == except {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyTree recursively copies the contents of src into dst.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
