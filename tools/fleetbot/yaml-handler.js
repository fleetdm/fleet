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
    if (change.search && change.replace !== undefined) {
      const replaceTrimmed = change.replace.trim();
      if (PLACEHOLDER_PATTERNS.some((p) => p.test(replaceTrimmed))) {
        throw new Error(`Refusing to commit: "${change.filePath}" replace value is placeholder content. Please try again.`);
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
 * Resolve the final content for a change object, handling both full-content
 * and search/replace patch modes.
 *
 * @param {object} change - { filePath, content, search, replace }
 * @param {function} getContent - async (fullPath) => string|null, fetches current file content
 * @param {string} fullPath - full repo path for the file
 * @param {string} [logPrefix] - logging prefix
 * @returns {Promise<string>} resolved file content
 */
async function resolveChangeContent(change, getContent, fullPath, logPrefix = "") {
  if (change.search && change.replace !== null) {
    const currentContent = await getContent(fullPath);
    if (currentContent === null) {
      console.log(`${logPrefix} File ${change.filePath} not found, creating with replace content`);
      return change.replace;
    }
    if (currentContent.includes(change.search)) {
      const result = currentContent.replace(change.search, change.replace);
      console.log(`${logPrefix} Applied patch to ${change.filePath} (${currentContent.length} → ${result.length} chars)`);
      return result;
    }
    // Try whitespace-normalized matching as fallback
    const normalize = (s) => s.replace(/[ \t]+/g, " ").replace(/\r\n/g, "\n").trim();
    if (normalize(currentContent).includes(normalize(change.search))) {
      const searchLines = change.search.split("\n").map((l) => l.trim());
      const contentLines = currentContent.split("\n");
      let startIdx = -1;
      for (let i = 0; i <= contentLines.length - searchLines.length; i++) {
        let match = true;
        for (let j = 0; j < searchLines.length; j++) {
          if (contentLines[i + j].trim() !== searchLines[j]) {
            match = false;
            break;
          }
        }
        if (match) {
          startIdx = i;
          break;
        }
      }
      if (startIdx !== -1) {
        const before = contentLines.slice(0, startIdx);
        const after = contentLines.slice(startIdx + searchLines.length);
        console.log(`${logPrefix} Applied patch to ${change.filePath} with whitespace-normalized match`);
        return [...before, change.replace, ...after].join("\n");
      }
    }
    console.error(`${logPrefix} Search string not found in ${change.filePath}:\n---SEARCH---\n${change.search}\n---END---`);
    throw new Error(`Cannot apply patch to "${change.filePath}": search string not found in file. The file may have changed since it was read.`);
  }
  if (change.content) {
    return change.content;
  }
  throw new Error(`Change for "${change.filePath}" has neither content nor search/replace`);
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
