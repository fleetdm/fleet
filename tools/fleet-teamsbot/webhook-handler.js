const crypto = require("crypto");
const path = require("path");
const { validateProposedChanges, validateResolvedChanges } = require("./yaml-handler");

// Known safe error prefixes that can be shown to users
const SAFE_ERROR_PREFIXES = [
  "Refusing to commit",
  "Invalid file path",
  "CI auto-fix failed",
];

function sanitizeErrorMessage(err) {
  const msg = err.message || "";
  if (msg.includes("Claude returned") || msg.includes("Raw response")) {
    return "Failed to process the AI response. Please try rephrasing your request.";
  }
  if (SAFE_ERROR_PREFIXES.some((p) => msg.startsWith(p))) {
    return msg;
  }
  if (err.status === 429 || msg.includes("rate_limit")) {
    return "Rate-limited by the AI service. Please wait a moment and try again.";
  }
  if (err.status === 529 || msg.includes("overloaded")) {
    return "The AI service is temporarily overloaded. Please try again in a minute.";
  }
  return "An unexpected error occurred. Please try again.";
}

function verifySignature(rawBody, signatureHeader, secret) {
  if (!signatureHeader) return false;
  const expected =
    "sha256=" +
    crypto.createHmac("sha256", secret).update(rawBody).digest("hex");
  try {
    return crypto.timingSafeEqual(
      Buffer.from(signatureHeader),
      Buffer.from(expected)
    );
  } catch {
    return false;
  }
}

function createWebhookHandler(config, github, claude) {
  return async function handleWebhook(req, res) {
    // Collect raw body from the IncomingMessage stream (capped at 1MB)
    const MAX_BODY_SIZE = 1024 * 1024;
    const buffers = [];
    let totalSize = 0;
    for await (const chunk of req) {
      totalSize += chunk.length;
      if (totalSize > MAX_BODY_SIZE) {
        res.writeHead(413);
        res.end("Request body too large");
        return;
      }
      buffers.push(chunk);
    }
    const rawBody = Buffer.concat(buffers).toString("utf-8");

    // Verify webhook signature
    const signature = req.headers["x-hub-signature-256"];
    if (!verifySignature(rawBody, signature, config.webhook.secret)) {
      console.log("[webhook] Invalid signature, rejecting");
      res.writeHead(401);
      res.end("Invalid signature");
      return;
    }

    const event = req.headers["x-github-event"];

    // Handle ping event
    if (event === "ping") {
      console.log("[webhook] Ping received");
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "pong" }));
      return;
    }

    let payload;
    try {
      payload = JSON.parse(rawBody);
    } catch (err) {
      console.error("[webhook] Invalid JSON payload:", err.message);
      res.writeHead(400, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: false, message: "Invalid JSON payload" }));
      return;
    }

    // Route check_run events for CI auto-fix
    if (event === "check_run") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "processing" }));

      if (config.ci.autoFix) {
        handleCheckRun(payload, config, github, claude).catch(async (err) => {
          console.error("[webhook] Error handling check_run:", err);
          const prNumber = payload.check_run?.pull_requests?.[0]?.number;
          if (prNumber) {
            try {
              await github.addPullRequestComment(prNumber, `🤖 **Fleet:** CI auto-fix failed: ${sanitizeErrorMessage(err)}`);
            } catch (replyErr) {
              console.error("[webhook] Failed to post CI error comment:", replyErr);
            }
          }
        });
      }
      return;
    }

    // Only handle comment events from here
    if (event !== "issue_comment" && event !== "pull_request_review_comment") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "ignored" }));
      return;
    }

    // Only handle new comments (not edits or deletions)
    if (payload.action !== "created") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "ignored" }));
      return;
    }

    // For issue_comment, only handle comments on PRs (not plain issues)
    if (event === "issue_comment" && !payload.issue.pull_request) {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "not a PR" }));
      return;
    }

    // Skip bot's own comments to prevent infinite loops
    const commentAuthor = payload.comment.user.login;
    const commentBody = payload.comment.body || "";
    if (
      commentAuthor === config.github.botUsername ||
      payload.comment.user.type === "Bot" ||
      commentBody.startsWith("🤖 **Fleet:**")
    ) {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "skipping bot comment" }));
      return;
    }

    // Authorization: only process comments from repo collaborators
    const TRUSTED_ASSOCIATIONS = new Set(["OWNER", "MEMBER", "COLLABORATOR"]);
    const authorAssociation = payload.comment.author_association;
    if (!TRUSTED_ASSOCIATIONS.has(authorAssociation)) {
      console.log(`[webhook] Ignoring comment from ${commentAuthor} (association: ${authorAssociation})`);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, message: "unauthorized author" }));
      return;
    }

    // Respond to GitHub immediately to avoid timeout
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ ok: true, message: "processing" }));

    // Extract PR number — different payload structure per event type
    const prNumber = event === "issue_comment"
      ? payload.issue.number
      : payload.pull_request.number;
    const commentId = payload.comment.id;
    console.log(`[webhook] PR #${prNumber} comment from ${commentAuthor}: "${commentBody.slice(0, 100)}"`);

    // Check if the bot was @mentioned
    const mentionedBot = config.github.botUsername &&
      commentBody.toLowerCase().includes(`@${config.github.botUsername.toLowerCase()}`);

    processComment({ prNumber, commentBody, commentId, event, mentionedBot }, config, github, claude).catch(
      async (err) => {
        console.error("[webhook] Error processing comment:", err);
        try {
          await github.addPullRequestComment(
            prNumber,
            `🤖 **Fleet:** Error processing your request: ${sanitizeErrorMessage(err)}`
          );
        } catch (replyErr) {
          console.error("[webhook] Failed to post error comment:", replyErr);
        }
      }
    );
  };
}

async function processComment({ prNumber, commentBody, commentId, event, mentionedBot }, config, github, claude) {
  // Only respond when explicitly @mentioned — avoid surprising users
  // by reacting to comments that weren't directed at the bot.
  if (!mentionedBot) {
    console.log(`[webhook] Ignoring PR #${prNumber} comment — bot was not @mentioned`);
    return;
  }

  // Fetch PR details
  console.log(`[webhook] Fetching PR #${prNumber} details...`);
  const pr = await github.getPullRequest(prNumber);

  if (pr.state !== "open") {
    console.log(`[webhook] Ignoring PR #${prNumber} — state is "${pr.state}"`);
    return;
  }

  // React with 👀 so the user knows we received their message
  try {
    if (event === "issue_comment") {
      await github.addIssueCommentReaction(commentId, "eyes");
    } else {
      await github.addReviewCommentReaction(commentId, "eyes");
    }
    console.log(`[webhook] Added 👀 reaction to comment ${commentId}`);
  } catch (err) {
    console.warn(`[webhook] Failed to add reaction: ${err.message}`);
  }

  // Fetch the files changed in the PR (only GitOps files)
  console.log(`[webhook] Fetching changed files for PR #${prNumber}...`);
  const prFiles = await github.getPullRequestFiles(prNumber);
  const gitopsPrefix = config.github.gitopsBasePath + "/";
  const activeFiles = prFiles.filter((f) => f.status !== "removed" && f.filename.startsWith(gitopsPrefix));
  console.log(`[webhook] ${activeFiles.length} GitOps files to read from branch ${pr.headBranch}`);

  // Read current contents of each file from the PR branch
  const currentFiles = {};
  for (const file of activeFiles) {
    const content = await github.getFileContentFromRef(file.filename, pr.headBranch);
    if (content !== null) {
      const relPath = file.filename.slice(gitopsPrefix.length);
      currentFiles[relPath] = content;
    }
  }
  console.log(`[webhook] Read ${Object.keys(currentFiles).length} files: ${Object.keys(currentFiles).join(", ")}`);

  // Send to Claude for revisions
  console.log("[webhook] Sending revision request to Claude...");
  const proposal = await claude.proposeRevisions(commentBody, currentFiles, pr.title);

  if (proposal.type === "info") {
    // Informational reply — no file changes, just post the answer
    console.log("[webhook] Claude returned an informational response");
    await github.addPullRequestComment(prNumber, `🤖 **Fleet:** ${proposal.text}`);
    console.log(`[webhook] PR #${prNumber} info reply posted. Done.`);
    return;
  }

  console.log(`[webhook] Claude proposed ${proposal.changes.length} changes`);

  // Guard: reject placeholder or suspiciously short content
  validateProposedChanges(proposal.changes);

  // Validate file paths: prevent traversal and enforce GitOps allowlist
  const changes = [];
  for (const c of proposal.changes) {
    const normalized = path.posix.normalize(c.filePath);
    if (normalized.startsWith("..") || path.posix.isAbsolute(normalized) ||
        !(normalized === "default.yml" || normalized.startsWith("fleets/") || normalized.startsWith("lib/"))) {
      throw new Error(`Invalid file path in response (must be under default.yml, fleets/, or lib/): ${c.filePath}`);
    }
    if (!c.content) {
      throw new Error(`Change for "${c.filePath}" is missing content`);
    }
    const fullPath = `${config.github.gitopsBasePath}/${normalized}`;
    changes.push({ path: fullPath, content: c.content, relPath: normalized });
  }

  // Validate YAML schema on resolved content
  const warnings = validateResolvedChanges(changes);
  if (warnings.length > 0) {
    console.warn(`[webhook] YAML validation warnings:\n${warnings.join("\n")}`);
  }

  console.log(`[webhook] Committing ${changes.length} file(s) to ${pr.headBranch}...`);
  await github.commitChanges(pr.headBranch, changes, `Update: ${proposal.summary}`);

  // Reply on the PR
  const fileList = proposal.changes
    .map((c) => `- \`${c.filePath}\` — ${c.changeDescription}`)
    .join("\n");
  const replyBody = `🤖 **Fleet:** Updated this PR based on your comment.\n\n**Summary:** ${proposal.summary}\n\n**Files changed:**\n${fileList}`;

  await github.addPullRequestComment(prNumber, replyBody);

  if (warnings.length > 0) {
    await github.addPullRequestComment(prNumber, `⚠️ **Validation warnings:**\n${warnings.map((w) => `- ${w}`).join("\n")}`);
  }

  console.log(`[webhook] PR #${prNumber} updated and reply posted. Done.`);
}

async function handleCheckRun(payload, config, github, claude) {
  const checkRun = payload.check_run;

  // Only handle completed, failed check runs matching our CI check name
  if (payload.action !== "completed" || checkRun.conclusion !== "failure") {
    return;
  }
  if (checkRun.name !== config.ci.checkName) {
    return;
  }

  // Need at least one associated PR
  const prRef = checkRun.pull_requests && checkRun.pull_requests[0];
  if (!prRef) {
    console.log(`[ci-fix] Check run "${checkRun.name}" failed but has no associated PR, skipping`);
    return;
  }

  const prNumber = prRef.number;
  const headSha = checkRun.head_sha;
  console.log(`[ci-fix] Check "${checkRun.name}" failed on PR #${prNumber} (sha: ${headSha.slice(0, 8)})`);

  // Loop prevention: allow up to 2 consecutive CI fix attempts, then stop
  try {
    const commit = await github.getCommit(headSha);
    if (commit.message.startsWith("CI fix:")) {
      // Check the parent commit too — if it's also a CI fix, we've already retried once
      const parentSha = commit.parentSha;
      if (parentSha) {
        const parent = await github.getCommit(parentSha);
        if (parent.message.startsWith("CI fix:")) {
          console.log(`[ci-fix] Skipping — already attempted CI fix twice`);
          return;
        }
      }
      console.log(`[ci-fix] Previous CI fix failed, retrying (attempt 2)...`);
    }
  } catch (err) {
    console.warn(`[ci-fix] Could not check commit history: ${err.message}, skipping as a safety precaution`);
    return;
  }

  // Fetch PR details
  const pr = await github.getPullRequest(prNumber);
  if (!pr.headBranch.startsWith("fleet/")) {
    console.log(`[ci-fix] Ignoring PR #${prNumber} — branch "${pr.headBranch}" is not a fleet branch`);
    return;
  }
  if (pr.state !== "open") {
    console.log(`[ci-fix] Ignoring PR #${prNumber} — state is "${pr.state}"`);
    return;
  }

  // Authorization: only auto-fix PRs from trusted authors
  const TRUSTED_ASSOCIATIONS = new Set(["OWNER", "MEMBER", "COLLABORATOR"]);
  if (!TRUSTED_ASSOCIATIONS.has(pr.authorAssociation)) {
    console.log(`[ci-fix] Ignoring PR #${prNumber} — author is not a trusted collaborator (${pr.authorAssociation})`);
    return;
  }

  // Fetch the failed job logs
  console.log(`[ci-fix] Fetching CI logs for sha ${headSha.slice(0, 8)}...`);
  const rawLogs = await github.getFailedJobLogs(headSha, config.ci.checkName);
  if (!rawLogs) {
    console.log("[ci-fix] Could not find failed job logs, skipping");
    return;
  }

  // Extract error lines from the logs
  const errorLines = extractErrors(rawLogs);
  if (!errorLines) {
    console.log("[ci-fix] No actionable errors found in logs, skipping");
    return;
  }
  console.log(`[ci-fix] Extracted errors:\n${errorLines}`);

  // Fetch current files from the PR branch (only GitOps files)
  const prFiles = await github.getPullRequestFiles(prNumber);
  const gitopsPrefix = config.github.gitopsBasePath + "/";
  const activeFiles = prFiles.filter((f) => f.status !== "removed" && f.filename.startsWith(gitopsPrefix));
  const currentFiles = {};
  for (const file of activeFiles) {
    const content = await github.getFileContentFromRef(file.filename, pr.headBranch);
    if (content !== null) {
      const relPath = file.filename.slice(gitopsPrefix.length);
      currentFiles[relPath] = content;
    }
  }
  console.log(`[ci-fix] Read ${Object.keys(currentFiles).length} files from ${pr.headBranch}`);

  // Send to Claude for a fix
  console.log("[ci-fix] Sending CI errors to Claude for auto-fix...");
  const proposal = await claude.proposeCiFix(errorLines, currentFiles, pr.title);

  if (proposal.type === "info") {
    console.log("[ci-fix] Claude returned info instead of a fix, posting as comment");
    await github.addPullRequestComment(prNumber, `🤖 **Fleet:** CI check \`${config.ci.checkName}\` failed. I analyzed the error but couldn't produce an automatic fix:\n\n${proposal.text}`);
    return;
  }

  console.log(`[ci-fix] Claude proposed ${proposal.changes.length} changes`);

  // Guard: reject placeholder or suspiciously short content
  validateProposedChanges(proposal.changes);

  // Validate file paths: prevent traversal and enforce GitOps allowlist
  const changes = [];
  for (const c of proposal.changes) {
    const normalized = path.posix.normalize(c.filePath);
    if (normalized.startsWith("..") || path.posix.isAbsolute(normalized) ||
        !(normalized === "default.yml" || normalized.startsWith("fleets/") || normalized.startsWith("lib/"))) {
      throw new Error(`Invalid file path in CI fix response (must be under default.yml, fleets/, or lib/): ${c.filePath}`);
    }
    if (!c.content) {
      throw new Error(`Change for "${c.filePath}" is missing content`);
    }
    const fullPath = `${config.github.gitopsBasePath}/${normalized}`;
    changes.push({ path: fullPath, content: c.content, relPath: normalized });
  }

  // Validate YAML schema on resolved content
  const ciWarnings = validateResolvedChanges(changes);
  if (ciWarnings.length > 0) {
    console.warn(`[ci-fix] YAML validation warnings:\n${ciWarnings.join("\n")}`);
  }
  console.log(`[ci-fix] Committing ${changes.length} file(s) to ${pr.headBranch}...`);
  await github.commitChanges(pr.headBranch, changes, `CI fix: ${proposal.summary}`);

  // Reply on the PR
  const fileList = proposal.changes
    .map((c) => `- \`${c.filePath}\` — ${c.changeDescription}`)
    .join("\n");
  const replyBody = `🤖 **Fleet:** CI check \`${config.ci.checkName}\` failed. I pushed a fix.\n\n**Errors:**\n\`\`\`\n${errorLines}\n\`\`\`\n\n**Summary:** ${proposal.summary}\n\n**Files changed:**\n${fileList}`;

  await github.addPullRequestComment(prNumber, replyBody);

  if (ciWarnings.length > 0) {
    await github.addPullRequestComment(prNumber, `⚠️ **Validation warnings:**\n${ciWarnings.map((w) => `- ${w}`).join("\n")}`);
  }

  console.log(`[ci-fix] PR #${prNumber} fix committed and comment posted. Done.`);
}

function extractErrors(rawLogs) {
  const lines = rawLogs.split("\n");
  const errorLines = [];
  for (let i = 0; i < lines.length; i++) {
    // Strip timestamp prefix (e.g. "2026-03-04T20:34:17.3948621Z ")
    const line = lines[i].replace(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z\s*/, "").trim();
    if (line.startsWith("Error:") || line.startsWith("error:") || line.startsWith("* ")) {
      errorLines.push(line);
    } else if (line.includes("##[error]")) {
      errorLines.push(line.replace("##[error]", "").trim());
    }
  }
  return errorLines.length > 0 ? errorLines.join("\n") : null;
}

module.exports = { createWebhookHandler };
