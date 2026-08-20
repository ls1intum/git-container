package main

import (
	"container/heap"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	log "github.com/sirupsen/logrus"
)

var dir = "/opt/repositories/"

func isValidPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// isWithinBase reports whether target is the base directory itself or a
// descendant of it. It rejects clone destinations that use "../" traversal to
// escape the configured base directory, while still accepting valid descendants
// when the base is a relative "." or the filesystem root "/".
func isWithinBase(base, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolveRepoDir computes the clone destination for a repository below base.
// It uses repo.Path when set, otherwise derives the directory name from the
// last segment of repo.URL. The second return value reports whether the
// destination stays within base; a false value means the repository must be
// skipped instead of cloned outside base.
func resolveRepoDir(base string, repo Repository) (string, bool) {
	repodir := base
	if repo.Path != "" {
		repodir = path.Join(repodir, repo.Path)
	} else {
		parts := strings.Split(repo.URL, "/")
		if len(parts) > 0 {
			repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
			repodir = path.Join(repodir, repoName)
		}
	}
	return repodir, isWithinBase(base, repodir)
}

func main() {
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	if env_dir := os.Getenv("REPOSITORY_DIR"); env_dir != "" && isValidPath(env_dir) {
		dir = env_dir
	}
	log.Info("Starting HadesCloneContainer")
	log.Info("Cloning repositories to ", dir)

	repo_map := getReposFromEnv()
	repos := getReposFromMap(repo_map)
	for repos.Len() > 0 {
		repo := heap.Pop(&repos).(*Item).Repository
		log.Debugf("Cloning repository: %+v", repo)
		// URL is mandatory
		if repo.URL == "" {
			log.Warn("Skipping repository without URL")
			continue
		}
		clone_options := &git.CloneOptions{
			URL: repo.URL,
		}
		// Check if username and password are set and use them for authentication
		if repo.Username != "" && repo.Password != "" {
			clone_options.Auth = &http.BasicAuth{
				Username: repo.Username,
				Password: repo.Password,
			}
		}
		// Check if a branch is specified
		if repo.Branch != "" {
			clone_options.ReferenceName = plumbing.ReferenceName("refs/heads/" + repo.Branch)
		}

		// Guard against a HADES_<group>_PATH or a repository URL whose derived
		// name resolves the clone destination outside the base directory
		// (e.g. "../../outside").
		repodir, ok := resolveRepoDir(dir, repo)
		if !ok {
			log.Warnf("Skipping repository %q whose clone destination %q escapes the base directory", repo.URL, repodir)
			continue
		}

		// Execute the clone operation
		r, err := git.PlainClone(repodir, false, clone_options)
		if err != nil {
			log.WithError(err).Error("Failed to clone repository")
			continue
		}
		log.Infof("Cloned repository %s to %s", repo.URL, repodir)

		// Check out a specific commit if one is requested. This guarantees the
		// container tests the exact commit the job was scheduled for, even if a
		// newer commit was pushed to the branch after the job was queued.
		if repo.Commit != "" {
			if err := checkoutCommit(r, repo.Commit); err != nil {
				log.WithError(err).Errorf("Failed to check out commit %s", repo.Commit)
				if removeErr := os.RemoveAll(repodir); removeErr != nil {
					log.WithError(removeErr).Errorf("Failed to remove unchecked-out repository %s", repodir)
				}
				continue
			}
			log.Infof("Checked out commit %s in %s", repo.Commit, repodir)
		}
	}
}

// checkoutCommit checks out the given full 40-character SHA-1 commit hash in the
// worktree of the already cloned repository. The clone is full (no Depth or
// SingleBranch), so the target commit is present in the object database.
func checkoutCommit(r *git.Repository, commit string) error {
	if !plumbing.IsHash(commit) {
		return fmt.Errorf("invalid commit hash %q", commit)
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	return w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(commit)})
}
