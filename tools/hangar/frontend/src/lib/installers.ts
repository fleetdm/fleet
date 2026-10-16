// Installers tab helpers. Process/log ids and platform keys mirror the Go side
// (internal/installers); file names come from the backend (PlannedFiles) so
// they're defined in one place.
import type { InstallerServerConfig, LogEntry } from "./ipc";

export const installersProcId = (serverId: string) => `installers:${serverId}`; // internal/installers.ProcID
export const installersChannel = (serverId: string) => `installers-${serverId}`; // internal/installers.LogChannel

// Platform keys must match internal/installers.Platforms.
export const INSTALLER_PLATFORMS: { key: string; label: string }[] = [
  { key: "macos", label: "macOS .pkg (universal)" },
  { key: "windows-amd64", label: "Windows x64 .msi" },
  { key: "windows-arm64", label: "Windows ARM64 .msi" },
  { key: "deb-amd64", label: "Linux x64 .deb" },
  { key: "deb-arm64", label: "Linux ARM64 .deb" },
  { key: "rpm-amd64", label: "Linux x64 .rpm" },
  { key: "rpm-arm64", label: "Linux ARM64 .rpm" },
];

// CUSTOM_TARGET is the build target for a Fleet server Hangar doesn't run
// (internal/installers.CustomTarget).
export const CUSTOM_TARGET = "custom";

export const UPDATE_OFFICIAL = "official";
export const UPDATE_LOCAL_TUF = "local-tuf";

// defaultInstallerServer mirrors settings.DefaultInstallerServer: every
// platform, Fleet Desktop on, official updates on stable.
export function defaultInstallerServer(): InstallerServerConfig {
  return {
    name: "",
    fleetctl: "",
    enroll_secret: "",
    extra_flags: "",
    context: "",
    platforms: INSTALLER_PLATFORMS.map((p) => p.key),
    fleet_desktop: true,
    debug: false,
    team_id: null,
    team_name: "",
    fleet_url: "",
    update_source: UPDATE_OFFICIAL,
    orbit_channel: "stable",
    desktop_channel: "stable",
    osqueryd_channel: "stable",
  };
}

export type JobStatus = "building" | "ok" | "failed";

// build.sh prints "==> [<key>] building <file>" / "==> [<key>] ok" /
// "==> [<key>] failed"; keep in sync with internal/installers/build.sh.
const STATUS_LINE = /^==> \[([a-z0-9-]+)\] (building|ok|failed)\b/;

// jobStatuses reads per-platform status from a build's log.
export function jobStatuses(entries: LogEntry[]): Record<string, JobStatus> {
  const out: Record<string, JobStatus> = {};
  for (const e of entries) {
    const m = STATUS_LINE.exec(e.message.trim());
    if (m) out[m[1]] = m[2] as JobStatus;
  }
  return out;
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}
