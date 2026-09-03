// SCEP helpers. A SCEP "profile" is one saved launch config for the shared
// in-repo scepserver binary — they differ by depot (CA), port, and challenge,
// so several run side by side to expose multiple Custom SCEP CAs to Fleet at
// once. Process/log ids mirror the Go side (internal/scep) so the frontend can
// address a running server's state and log ring without a round-trip.
import type { ScepProfile, Settings } from "./ipc";

// --- process / log namespacing (must match internal/scep ProcID/LogChannel) ---

export function scepProcId(profileId: string): string {
  return `scep:${profileId}`;
}

export function scepChannel(profileId: string): string {
  return `scep-${profileId}`;
}

// --- URLs / copy strings ---

/// The SCEP URL to paste into a Fleet Custom SCEP CA. Falls back to localhost
/// when the LAN IP isn't known yet.
export function scepUrl(lanIp: string, port: number): string {
  const host = lanIp || "localhost";
  return `http://${host}:${port}/scep`;
}

// --- settings mutation (returns new Settings; caller persists) ---

/// Inserts or replaces a profile by id, preserving order (existing → in place,
/// new → appended).
export function upsertScepProfile(
  settings: Settings,
  profile: ScepProfile,
): Settings {
  const exists = settings.scep_profiles.some((p) => p.id === profile.id);
  const scep_profiles = exists
    ? settings.scep_profiles.map((p) => (p.id === profile.id ? profile : p))
    : [...settings.scep_profiles, profile];
  return { ...settings, scep_profiles };
}

export function removeScepProfile(settings: Settings, id: string): Settings {
  return {
    ...settings,
    scep_profiles: settings.scep_profiles.filter((p) => p.id !== id),
  };
}

export function scepProfileById(
  settings: Settings,
  id: string,
): ScepProfile | undefined {
  return settings.scep_profiles.find((p) => p.id === id);
}

/// True when another profile already claims `port` (for a conflict warning).
export function scepPortConflict(
  settings: Settings,
  id: string,
  port: number,
): boolean {
  return settings.scep_profiles.some((p) => p.id !== id && p.port === port);
}
