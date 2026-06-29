const path = require("path");
const yaml = require("js-yaml");
const { createTwoFilesPatch } = require("diff");

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
    if (policy.platform) {
      const validPlatforms = ["darwin", "windows", "linux", "chrome"];
      const bad = policy.platform.split(",").map((t) => t.trim()).filter((t) => !validPlatforms.includes(t));
      if (bad.length > 0) {
        errors.push(`Invalid platform "${policy.platform}". Valid values: ${validPlatforms.join(", ")}`);
      }
    }
  }

  return errors;
}

const PLACEHOLDER_PATTERNS = [
  /^UNABLE_TO_GENERATE/im,
  /^CONTENT_TOO_LONG/im,
  /^PLACEHOLDER/im,
  /^\[.*content.*\]$/im,
  /^TODO/im,
];

/**
 * Validate proposed changes before committing: reject placeholders and
 * suspiciously short content.
 * Throws on the first invalid change found.
 */
function validateProposedChanges(changes) {
  for (const change of changes) {
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

module.exports = {
  parseYaml,
  dumpYaml,
  generateDiffSummary,
  validateTeamYaml,
  validatePolicyYaml,
  validateProposedChanges,
  validateResolvedChanges,
};
