// Fleet's 5 core values. Source:
// https://fleetdm.com/handbook/company#values
//
// Order below matches the handbook's presentation order. The quiz accepts any
// order the user types; this array is just for canonical display and matching.

export const FLEET_VALUES = [
  { name: "Empathy", emoji: "🔴" },
  { name: "Ownership", emoji: "🟠" },
  { name: "Results", emoji: "🟢" },
  { name: "Objectivity", emoji: "🔵" },
  { name: "Openness", emoji: "🟣" },
] as const;

export type FleetValue = (typeof FLEET_VALUES)[number];

// Normalize a user answer for comparison: trim + lowercase. Accepts any
// capitalization or surrounding whitespace.
export function normalizeAnswer(s: string): string {
  return s.trim().toLowerCase();
}

const CANONICAL_SET = new Set(
  FLEET_VALUES.map((v) => normalizeAnswer(v.name)),
);

export function isCorrectValue(answer: string): boolean {
  return CANONICAL_SET.has(normalizeAnswer(answer));
}

// Returns the canonical value name for a user answer, or null if no match.
// Used so we can show "Empathy" even if the user typed "EMPATHY" or "  empathy ".
export function canonicalize(answer: string): string | null {
  const n = normalizeAnswer(answer);
  const match = FLEET_VALUES.find((v) => normalizeAnswer(v.name) === n);
  return match?.name ?? null;
}
