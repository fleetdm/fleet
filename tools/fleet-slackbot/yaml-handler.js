const path = require("path");
const yaml = require("js-yaml");
const { createTwoFilesPatch, applyPatch } = require("diff");

/**
 * Parse YAML content into a JS object.
 */
function parseYaml(content) {
  return yaml.load(content);
}

/**
 * Dump a JS object back to YAML string.
 */
function dumpYaml(obj) {
  return yaml.dump(obj, {
    indent: 2,
    lineWidth: -1, // no line wrapping
    noRefs: true,
    quotingType: '"',
    forceQuotes: false,
  });
}

/**
 * Generate a unified diff between original and modified file content.
 * Returns a string suitable for display in Slack (truncated to maxLen).
 */
function generateDiffSummary(filePath, original, modified, maxLen = 2800) {
  if (original == null) {
    // New file — show the full content, truncated
    const lines = modified.split("\n");
    const preview = lines.slice(0, 30).join("\n");
    const result = lines.length > 30
      ? `${preview}\n... (${lines.length - 30} more lines)`
      : preview;
    return result.slice(0, maxLen);
  }

  const patch = createTwoFilesPatch(
    `a/${filePath}`,
    `b/${filePath}`,
    original,
    modified,
    "",
    "",
    { context: 3 }
  );

  // Strip the file header lines (first 4 lines) for cleaner display
  const lines = patch.split("\n");
  const diffBody = lines.slice(4).join("\n");

  if (diffBody.length > maxLen) {
    return diffBody.slice(0, maxLen) + "\n... (diff truncated)";
  }
  return diffBody;
}

/**
 * Validate a team YAML file has the expected structure.
 * Returns an array of error strings (empty if valid).
 */
function validateTeamYaml(content, filePath = "") {
  const errors = [];
  let data;
  try {
    data = yaml.load(content);
  } catch (err) {
    return [`Invalid YAML: ${err.message}`];
  }

  if (!data || typeof data !== "object") {
    return ["YAML did not parse to an object"];
  }

  // fleets/unassigned.yml intentionally omits name per the GitOps schema
  const isUnassigned = path.posix.basename(filePath) === "unassigned.yml";
  if (!data.name && !isUnassigned) errors.push("Missing required field: name");

  return errors;
}

/**
 * Validate a policy YAML file.
 * Returns an array of error strings (empty if valid).
 */
function validatePolicyYaml(content) {
  const errors = [];
  let data;
  try {
    data = yaml.load(content);
  } catch (err) {
    return [`Invalid YAML: ${err.message}`];
  }

  const policies = Array.isArray(data) ? data : [data];

  for (const policy of policies) {
    if (typeof policy !== "object" || policy === null || Array.isArray(policy)) {
      errors.push("Policy must be an object");
      continue;
    }
    if (!policy.name) errors.push("Policy missing required field: name");
    if (!policy.query) errors.push("Policy missing required field: query");
    if (typeof policy.critical === "undefined") errors.push("Policy missing required field: critical");
    if (!policy.description) errors.push("Policy missing required field: description");
    if (!policy.resolution) errors.push("Policy missing required field: resolution");
    if (!policy.platform) {
      errors.push("Policy missing required field: platform");
    } else if (!["darwin", "windows", "linux"].includes(policy.platform)) {
      errors.push(`Invalid platform "${policy.platform}". Must be: darwin, windows, or linux`);
    }
  }

  return errors;
}

const PLACEHOLDER_PATTERNS = [
  /^UNABLE_TO_GENERATE/i,
  /^CONTENT_TOO_LONG/i,
  /^PLACEHOLDER/i,
  /^\[.*content.*\]$/i,
  /^TODO/i,
];

/**
 * Validate proposed changes before committing: reject placeholders and
 * suspiciously short content.
 * Throws on the first invalid change found.
 */
function validateProposedChanges(changes) {
  for (const change of changes) {
    if (change.patch) {
      // For patch mode, check that added lines don't contain placeholder content
      const addedLines = change.patch.split("\n")
        .filter((l) => l.startsWith("+") && !l.startsWith("+++"))
        .map((l) => l.slice(1).trim())
        .join("\n")
        .trim();
      if (PLACEHOLDER_PATTERNS.some((p) => p.test(addedLines))) {
        throw new Error(`Refusing to commit: "${change.filePath}" patch contains placeholder content. Please try again.`);
      }
      continue;
    }
    const trimmed = (change.content || "").trim();
    if (PLACEHOLDER_PATTERNS.some((p) => p.test(trimmed)) || !trimmed) {
      throw new Error(`Refusing to commit: "${change.filePath}" has placeholder content instead of real file contents. Please try again.`);
    }
    if (!change.isNewFile && trimmed.length < 50) {
      throw new Error(`Refusing to commit: "${change.filePath}" content is only ${trimmed.length} chars — this likely means the full file was not generated. Please try again.`);
    }
  }
}

/**
 * Run YAML schema validation on resolved changes.
 * Returns an array of warning strings (empty if all valid).
 */
function validateResolvedChanges(changes) {
  const warnings = [];
  for (const change of changes) {
    if (change.relPath.startsWith("fleets/")) {
      const errs = validateTeamYaml(change.content, change.relPath);
      warnings.push(...errs.map((e) => `\`${change.relPath}\`: ${e}`));
    } else if (change.relPath.includes("/policies/")) {
      const errs = validatePolicyYaml(change.content);
      warnings.push(...errs.map((e) => `\`${change.relPath}\`: ${e}`));
    }
  }
  return warnings;
}

/**
 * Fix malformed @@ hunk headers in a unified diff.
 * LLMs often get the line counts wrong; this recalculates them
 * from the actual +/-/context lines in each hunk.
 */
function fixHunkHeaders(patch) {
  const lines = patch.split("\n");
  const result = [];
  for (let i = 0; i < lines.length; i++) {
    const hunkMatch = lines[i].match(/^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@(.*)$/);
    if (!hunkMatch) {
      result.push(lines[i]);
      continue;
    }
    let oldCount = 0;
    let newCount = 0;
    let j = i + 1;
    while (j < lines.length) {
      const line = lines[j];
      const ch = line[0];
      if (ch === "-") oldCount++;
      else if (ch === "+") newCount++;
      else if (ch === " ") { oldCount++; newCount++; }
      else break; // stop at @@, ---, empty lines, or anything else
      j++;
    }
    result.push(`@@ -${hunkMatch[1]},${oldCount} +${hunkMatch[2]},${newCount} @@${hunkMatch[3]}`);
  }
  return result.join("\n");
}

/**
 * Resolve the final content for a change object, handling both full-content
 * and unified-diff patch modes.
 *
 * @param {object} change - { filePath, content, patch }
 * @param {function} getContent - async (fullPath) => string|null, fetches current file content
 * @param {string} fullPath - full repo path for the file
 * @param {string} [logPrefix] - logging prefix
 * @returns {Promise<string>} resolved file content
 */
async function resolveChangeContent(change, getContent, fullPath, logPrefix = "") {
  if (change.patch) {
    const currentContent = await getContent(fullPath);
    if (currentContent === null) {
      console.log(`${logPrefix} File ${change.filePath} not found, cannot apply patch to non-existent file`);
      throw new Error(`Cannot apply patch to "${change.filePath}": file does not exist. Use full content mode for new files.`);
    }
    let result;
    try {
      result = applyPatch(currentContent, fixHunkHeaders(change.patch), { fuzzFactor: 2 });
    } catch (err) {
      console.error(`${logPrefix} Failed to parse/apply unified diff to ${change.filePath}: ${err.message}\n${change.patch}`);
      throw new Error(`Cannot apply patch to "${change.filePath}": ${err.message}`);
    }
    if (result === false) {
      console.error(`${logPrefix} Failed to apply unified diff to ${change.filePath}:\n${change.patch}`);
      throw new Error(`Cannot apply patch to "${change.filePath}": patch does not match the current file contents. The file may have changed since it was read.`);
    }
    console.log(`${logPrefix} Applied unified diff to ${change.filePath} (${currentContent.length} → ${result.length} chars)`);
    return result;
  }
  if (change.content) {
    return change.content;
  }
  throw new Error(`Change for "${change.filePath}" has neither content nor patch`);
}

module.exports = {
  parseYaml,
  dumpYaml,
  generateDiffSummary,
  validateTeamYaml,
  validatePolicyYaml,
  validateProposedChanges,
  validateResolvedChanges,
  resolveChangeContent,
};
