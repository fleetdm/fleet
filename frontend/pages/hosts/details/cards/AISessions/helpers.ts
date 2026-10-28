import { IHostAISession } from "interfaces/ai_sessions";
import { secondsToHms } from "utilities/helpers";

interface IAIAgentDisplay {
  name: string;
  /** Name the software icon is matched on. */
  iconName: string;
}

const AI_AGENTS: Record<string, IAIAgentDisplay> = {
  claude_code: { name: "Claude Code", iconName: "claude" },
  codex_cli: { name: "Codex CLI", iconName: "codex" },
  copilot_cli: { name: "GitHub Copilot CLI", iconName: "github copilot" },
  cursor: { name: "Cursor", iconName: "cursor" },
  gemini_cli: { name: "Gemini CLI", iconName: "google gemini" },
};

/** Shown wherever a session's events may be incomplete while Fleet collects
 * the calls it made before Fleet first stored it. */
export const AI_SESSION_BACKFILL_MESSAGE =
  "Fleet is still collecting earlier events from this session, so some may not appear yet.";

export const getAIAgentDisplay = (agent: string): IAIAgentDisplay =>
  AI_AGENTS[agent] ?? { name: agent, iconName: agent };

/** The account the agent is signed in with, or the local account when the
 * agent doesn't record one. */
export const getAISessionUserAccount = (
  session: Pick<IHostAISession, "account_identifier" | "username">
) => session.account_identifier || session.username;

/** The session's duration, e.g. "1 hr 5 mins" or "42 secs", or null when the
 * agent didn't record when it started or ended. */
export const getAISessionDuration = (
  session: Pick<IHostAISession, "started_at" | "ended_at">
): string | null => {
  if (!session.started_at || !session.ended_at) {
    return null;
  }
  const seconds = Math.floor(
    (Date.parse(session.ended_at) - Date.parse(session.started_at)) / 1000
  );
  if (Number.isNaN(seconds) || seconds < 0) {
    return null;
  }
  // Seconds are noise once a session runs for hours.
  const shown = seconds >= 3600 ? seconds - (seconds % 60) : seconds;
  return secondsToHms(shown).trim() || "0 secs";
};
