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
  if (!original) {
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
function validateTeamYaml(content) {
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

  if (!data.name) errors.push("Missing required field: name");

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

module.exports = {
  parseYaml,
  dumpYaml,
  generateDiffSummary,
  validateTeamYaml,
  validatePolicyYaml,
};
