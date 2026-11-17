// Loads the fleet-gitops skill that `fleetctl new` ships, from the repo's base
// branch, so the bot follows the same GitOps guidance as customers' agents
// instead of a copy that drifts.
const SKILL_DIR = "cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops";
const REFERENCES = ["profiles-apple", "profiles-windows", "profiles-android", "queries", "software"];
const REFRESH_MS = 60 * 60 * 1000;

// Drop the frontmatter and the agent-only "Task: $ARGUMENTS" line.
function stripSkillHeader(raw) {
  let body = raw;
  if (body.startsWith("---\n")) {
    const end = body.indexOf("\n---\n", 4);
    if (end !== -1) body = body.slice(end + 5);
  }
  return body.replace(/^Task: \$ARGUMENTS\n+/m, "").trim();
}

class SkillLoader {
  constructor(github) {
    this.github = github;
    this.skill = null;
    this.fetchedAt = 0;
  }

  // Returns the skill body, refreshing at most hourly. On a failed fetch it
  // keeps serving the last good copy (or null, so the bot still answers).
  async getSkill() {
    if (this.skill && Date.now() - this.fetchedAt < REFRESH_MS) return this.skill;
    try {
      const raw = await this.github.getFileContent(`${SKILL_DIR}/SKILL.md`);
      if (raw) {
        this.skill = stripSkillHeader(raw);
        this.fetchedAt = Date.now();
      } else {
        console.error(`[skill] ${SKILL_DIR}/SKILL.md not found`);
      }
    } catch (err) {
      console.error(`[skill] Failed to load SKILL.md: ${err.message}`);
    }
    return this.skill;
  }

  async getReference(name) {
    if (!REFERENCES.includes(name)) {
      return `Error: Unknown reference "${name}". Valid options: ${REFERENCES.join(", ")}`;
    }
    const content = await this.github.getFileContent(`${SKILL_DIR}/references/${name}.md`);
    return content ?? `Error: Reference not found: ${name}`;
  }
}

module.exports = { SkillLoader, REFERENCES, stripSkillHeader };
