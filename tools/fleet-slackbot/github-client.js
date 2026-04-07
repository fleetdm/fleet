const { Octokit } = require("@octokit/rest");

class GitHubClient {
  constructor({ token, repo, baseBranch = "main", gitopsBasePath = "it-and-security" }) {
    this.octokit = new Octokit({ auth: token });
    const [owner, repoName] = repo.split("/");
    this.owner = owner;
    this.repo = repoName;
    this.baseBranch = baseBranch;
    this.gitopsBasePath = gitopsBasePath;
  }

  /**
   * Fetch the content of a single file from the base branch.
   * Returns the file content as a string, or null if the file doesn't exist.
   */
  async getFileContent(filePath) {
    try {
      const { data } = await this.octokit.repos.getContent({
        owner: this.owner,
        repo: this.repo,
        path: filePath,
        ref: this.baseBranch,
      });
      if (!data.content) return null;
      return Buffer.from(data.content, "base64").toString("utf-8");
    } catch (err) {
      if (err.status === 404) return null;
      throw new Error(`Failed to fetch ${filePath}: ${err.message}`);
    }
  }

  /**
   * Get all file paths under a prefix using the Git tree API.
   * Returns an array of paths relative to the gitops base path.
   */
  async getRepoTreePaths() {
    const { data: refData } = await this.octokit.git.getRef({
      owner: this.owner,
      repo: this.repo,
      ref: `heads/${this.baseBranch}`,
    });

    // Fetch the root tree (non-recursive) to find the gitops subtree SHA
    const { data: rootTree } = await this.octokit.git.getTree({
      owner: this.owner,
      repo: this.repo,
      tree_sha: refData.object.sha,
    });

    const subtreeEntry = rootTree.tree.find(
      (item) => item.type === "tree" && item.path === this.gitopsBasePath
    );
    if (!subtreeEntry) {
      throw new Error(`GitOps directory "${this.gitopsBasePath}" not found in repo`);
    }

    // Fetch only the gitops subtree recursively
    const { data: subtree } = await this.octokit.git.getTree({
      owner: this.owner,
      repo: this.repo,
      tree_sha: subtreeEntry.sha,
      recursive: "true",
    });

    if (subtree.truncated) {
      throw new Error(`GitOps directory tree was truncated by GitHub API; too many files`);
    }

    return subtree.tree
      .filter((item) => item.type === "blob")
      .map((item) => item.path);
  }

  /**
   * Create a new branch from the head of the base branch.
   */
  async createBranch(branchName) {
    const { data: refData } = await this.octokit.git.getRef({
      owner: this.owner,
      repo: this.repo,
      ref: `heads/${this.baseBranch}`,
    });

    await this.octokit.git.createRef({
      owner: this.owner,
      repo: this.repo,
      ref: `refs/heads/${branchName}`,
      sha: refData.object.sha,
    });
  }

  /**
   * Commit one or more file changes to a branch as a single commit.
   * Uses the Git Data API for multi-file atomic commits.
   *
   * @param {string} branchName
   * @param {Array<{path: string, content: string}>} changes
   * @param {string} commitMessage
   * @returns {string} The new commit SHA
   */
  async commitChanges(branchName, changes, commitMessage) {
    // Get the current commit on the branch
    const { data: refData } = await this.octokit.git.getRef({
      owner: this.owner,
      repo: this.repo,
      ref: `heads/${branchName}`,
    });
    const baseCommitSha = refData.object.sha;

    const { data: baseCommit } = await this.octokit.git.getCommit({
      owner: this.owner,
      repo: this.repo,
      commit_sha: baseCommitSha,
    });

    // Create blobs for each changed file
    const treeItems = [];
    for (const change of changes) {
      const { data: blob } = await this.octokit.git.createBlob({
        owner: this.owner,
        repo: this.repo,
        content: Buffer.from(change.content).toString("base64"),
        encoding: "base64",
      });
      treeItems.push({
        path: change.path,
        mode: "100644",
        type: "blob",
        sha: blob.sha,
      });
    }

    // Create a new tree
    const { data: newTree } = await this.octokit.git.createTree({
      owner: this.owner,
      repo: this.repo,
      base_tree: baseCommit.tree.sha,
      tree: treeItems,
    });

    // Create the commit
    const { data: newCommit } = await this.octokit.git.createCommit({
      owner: this.owner,
      repo: this.repo,
      message: commitMessage,
      tree: newTree.sha,
      parents: [baseCommitSha],
    });

    // Update the branch ref
    await this.octokit.git.updateRef({
      owner: this.owner,
      repo: this.repo,
      ref: `heads/${branchName}`,
      sha: newCommit.sha,
    });

    return newCommit.sha;
  }

  async getFileContentFromRef(filePath, ref) {
    try {
      const { data } = await this.octokit.repos.getContent({
        owner: this.owner,
        repo: this.repo,
        path: filePath,
        ref,
      });
      if (!data.content) return null; // large files have no inline content
      return Buffer.from(data.content, "base64").toString("utf-8");
    } catch (err) {
      if (err.status === 404) return null;
      throw new Error(`Failed to fetch ${filePath} at ref ${ref}: ${err.message}`);
    }
  }

  async getPullRequest(pullNumber) {
    const { data } = await this.octokit.pulls.get({
      owner: this.owner,
      repo: this.repo,
      pull_number: pullNumber,
    });
    return {
      number: data.number,
      title: data.title,
      body: data.body,
      headBranch: data.head.ref,
      baseBranch: data.base.ref,
      state: data.state,
      url: data.html_url,
      authorAssociation: data.author_association,
    };
  }

  async getPullRequestFiles(pullNumber) {
    const data = await this.octokit.paginate(this.octokit.pulls.listFiles, {
      owner: this.owner,
      repo: this.repo,
      pull_number: pullNumber,
      per_page: 100,
    });
    return data.map((f) => ({
      filename: f.filename,
      status: f.status,
    }));
  }

  async addPullRequestComment(pullNumber, body) {
    const { data } = await this.octokit.issues.createComment({
      owner: this.owner,
      repo: this.repo,
      issue_number: pullNumber,
      body,
    });
    return { id: data.id, url: data.html_url };
  }

  async addIssueCommentReaction(commentId, reaction) {
    await this.octokit.reactions.createForIssueComment({
      owner: this.owner,
      repo: this.repo,
      comment_id: commentId,
      content: reaction,
    });
  }

  async addReviewCommentReaction(commentId, reaction) {
    await this.octokit.reactions.createForPullRequestReviewComment({
      owner: this.owner,
      repo: this.repo,
      comment_id: commentId,
      content: reaction,
    });
  }

  async getCommit(sha) {
    const { data } = await this.octokit.git.getCommit({
      owner: this.owner,
      repo: this.repo,
      commit_sha: sha,
    });
    return {
      sha: data.sha,
      message: data.message,
      authorName: data.author.name,
      authorEmail: data.author.email,
      parentSha: data.parents?.[0]?.sha || null,
    };
  }

  async getFailedJobLogs(headSha, checkName) {
    // Find workflow runs for this commit
    const { data: runs } = await this.octokit.actions.listWorkflowRunsForRepo({
      owner: this.owner,
      repo: this.repo,
      head_sha: headSha,
    });

    for (const run of runs.workflow_runs) {
      // List jobs for this run
      const { data: jobsData } = await this.octokit.actions.listJobsForWorkflowRun({
        owner: this.owner,
        repo: this.repo,
        run_id: run.id,
      });

      const failedJob = jobsData.jobs.find(
        (j) => j.name === checkName && j.conclusion === "failure"
      );
      if (!failedJob) continue;

      // Fetch the logs
      const { data: logs } = await this.octokit.actions.downloadJobLogsForWorkflowRun({
        owner: this.owner,
        repo: this.repo,
        job_id: failedJob.id,
      });
      return logs;
    }
    return null;
  }

  /**
   * Open a pull request from branchName into baseBranch.
   * @returns {{ url: string, number: number }}
   */
  async createPullRequest(branchName, title, body, { draft = false } = {}) {
    const { data: pr } = await this.octokit.pulls.create({
      owner: this.owner,
      repo: this.repo,
      title,
      body,
      head: branchName,
      base: this.baseBranch,
      draft,
    });
    return { url: pr.html_url, number: pr.number };
  }
}

module.exports = GitHubClient;
