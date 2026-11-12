package services

import "github.com/fleetdm/fleet/tools/hangar/internal/gitrepo"

// GitService exposes branch listing, status, and checkout. Mirrors git.rs.
type GitService struct{}

func (s *GitService) GitBranchStatus(repo string) (gitrepo.BranchStatus, error) {
	return gitrepo.BranchStatusFor(repo)
}
func (s *GitService) GitListBranches(repo, filter, query string, limit *uint32) ([]gitrepo.Branch, error) {
	return gitrepo.ListBranches(repo, filter, query, limit)
}
func (s *GitService) GitFetch(repo string) (string, error) { return gitrepo.Fetch(repo) }
func (s *GitService) GitPull(repo string) (string, error)  { return gitrepo.Pull(repo) }
func (s *GitService) GitCheckout(repo, branch string) (string, error) {
	return gitrepo.Checkout(repo, branch)
}
func (s *GitService) GitStashAndCheckout(repo, branch string) (string, error) {
	return gitrepo.StashAndCheckout(repo, branch)
}
func (s *GitService) GitDiscardAndCheckout(repo, branch string) (string, error) {
	return gitrepo.DiscardAndCheckout(repo, branch)
}

// Worktree management — used by multi-server to build/run different branches
// in parallel from one clone.
func (s *GitService) GitListWorktrees(repo string) ([]gitrepo.Worktree, error) {
	return gitrepo.ListWorktrees(repo)
}
func (s *GitService) GitAddWorktree(repo, path, ref string) (string, error) {
	return gitrepo.AddWorktree(repo, path, ref)
}
func (s *GitService) GitRemoveWorktree(repo, path string, force bool) (string, error) {
	return gitrepo.RemoveWorktree(repo, path, force)
}
