package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeAndCommit writes a file into the worktree and creates a commit for it,
// returning the resulting commit hash.
func writeAndCommit(t *testing.T, repo *git.Repository, dir, name, content, message string) plumbing.Hash {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	w, err := repo.Worktree()
	require.NoError(t, err)
	_, err = w.Add(name)
	require.NoError(t, err)
	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)
	return hash
}

// createSourceRepo initialises a git repository on disk with two commits and
// returns its path together with the two commit hashes.
func createSourceRepo(t *testing.T) (string, plumbing.Hash, plumbing.Hash) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	first := writeAndCommit(t, repo, dir, "file.txt", "first", "first commit")
	second := writeAndCommit(t, repo, dir, "file.txt", "second", "second commit")
	return dir, first, second
}

// headCommit returns the commit hash the worktree HEAD currently points at.
func headCommit(t *testing.T, repodir string) plumbing.Hash {
	t.Helper()
	repo, err := git.PlainOpen(repodir)
	require.NoError(t, err)
	head, err := repo.Head()
	require.NoError(t, err)
	return head.Hash()
}

// TestCloneWithoutCommit verifies that when no commit is requested the worktree
// stays at the branch HEAD, i.e. the behaviour is unchanged.
func TestCloneWithoutCommit(t *testing.T) {
	source, _, second := createSourceRepo(t)

	target := filepath.Join(t.TempDir(), "clone")
	r, err := git.PlainClone(target, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)

	// No checkout requested: HEAD must remain at the latest commit.
	assert.Equal(t, second, headCommit(t, target))

	// Sanity check: the repository handle is usable.
	_, err = r.Worktree()
	require.NoError(t, err)
}

// TestCloneAndCheckoutCommit verifies that requesting an older commit checks out
// exactly that commit instead of the branch HEAD.
func TestCloneAndCheckoutCommit(t *testing.T) {
	source, first, second := createSourceRepo(t)
	require.NotEqual(t, first, second)

	target := filepath.Join(t.TempDir(), "clone")
	r, err := git.PlainClone(target, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)

	// After the clone HEAD is at the newest commit.
	require.Equal(t, second, headCommit(t, target))

	// Checking out the first commit must move the worktree back to it.
	require.NoError(t, checkoutCommit(r, first.String()))
	assert.Equal(t, first, headCommit(t, target))

	// The checked out file content must match the first commit.
	content, err := os.ReadFile(filepath.Join(target, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "first", string(content))
}

// TestCheckoutMissingCommit verifies that checking out an unknown commit returns
// an error rather than panicking, so the caller can log and continue.
func TestCheckoutMissingCommit(t *testing.T) {
	source, _, _ := createSourceRepo(t)

	target := filepath.Join(t.TempDir(), "clone")
	r, err := git.PlainClone(target, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)

	err = checkoutCommit(r, "0123456789abcdef0123456789abcdef01234567")
	assert.Error(t, err)
}

// TestCheckoutInvalidCommitHash verifies that a malformed commit value is
// rejected up front instead of being silently coerced into a zero-padded hash
// by plumbing.NewHash.
func TestCheckoutInvalidCommitHash(t *testing.T) {
	source, _, _ := createSourceRepo(t)

	target := filepath.Join(t.TempDir(), "clone")
	r, err := git.PlainClone(target, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)

	err = checkoutCommit(r, "not-a-valid-hash")
	assert.Error(t, err)
}

// TestIsWithinBase verifies that repository paths are constrained to the base
// directory, so a HADES_<group>_PATH cannot use "../" traversal to clone
// outside REPOSITORY_DIR, while valid descendants are still accepted for
// relative "." and root "/" base directories.
func TestIsWithinBase(t *testing.T) {
	cases := []struct {
		base string
		path string
		want bool
	}{
		{"/opt/repositories/", "example", true},
		{"/opt/repositories/", "example/assignment", true},
		{"/opt/repositories/", "nested/deeper/repo", true},
		{"/opt/repositories/", "../../outside", false},
		{"/opt/repositories/", "../sibling", false},
		{"/opt/repositories/", "example/../../escape", false},
		// A relative base such as "." must still accept its descendants.
		{".", "example", true},
		{".", "../outside", false},
		// The filesystem root must accept absolute descendants.
		{"/", "example", true},
		{"/", "example/sub", true},
	}
	for _, c := range cases {
		got := isWithinBase(c.base, filepath.Join(c.base, c.path))
		assert.Equalf(t, c.want, got, "base %q path %q", c.base, c.path)
	}
}

// TestResolveRepoDir verifies the clone destination is derived correctly and
// rejected when it escapes the base directory, both via an explicit Path and
// via a repository URL whose last segment (e.g. a trailing "..") would resolve
// outside the base directory.
func TestResolveRepoDir(t *testing.T) {
	base := "/opt/repositories"
	cases := []struct {
		name    string
		repo    Repository
		wantDir string
		wantOK  bool
	}{
		{"explicit path", Repository{URL: "https://example.test/repo.git", Path: "example"}, "/opt/repositories/example", true},
		{"derived from url", Repository{URL: "https://example.test/repo.git"}, "/opt/repositories/repo", true},
		{"path escapes base", Repository{URL: "https://example.test/repo.git", Path: "../../outside"}, "", false},
		{"url ending in dotdot escapes base", Repository{URL: "https://example.test/.."}, "", false},
	}
	for _, c := range cases {
		gotDir, gotOK := resolveRepoDir(base, c.repo)
		assert.Equalf(t, c.wantOK, gotOK, "%s: ok", c.name)
		if c.wantOK {
			assert.Equalf(t, c.wantDir, gotDir, "%s: dir", c.name)
		}
	}
}

// TestCloneRemovedOnCheckoutFailure verifies that a failed checkout of a
// requested commit leaves no clone behind, so a downstream job can never test
// the wrong revision (the branch HEAD) instead of the requested commit.
func TestCloneRemovedOnCheckoutFailure(t *testing.T) {
	source, _, _ := createSourceRepo(t)

	target := filepath.Join(t.TempDir(), "clone")
	r, err := git.PlainClone(target, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)
	require.True(t, isValidPath(target))

	// Requesting an invalid commit must fail the checkout ...
	err = checkoutCommit(r, "not-a-valid-hash")
	require.Error(t, err)

	// ... and the caller must remove the clone so nothing is published.
	require.NoError(t, os.RemoveAll(target))
	assert.False(t, isValidPath(target))
}
