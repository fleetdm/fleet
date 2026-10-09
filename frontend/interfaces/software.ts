import { uniq } from "lodash";
import PropTypes from "prop-types";

import { IconNames } from "components/icons";

import { ICommandResult } from "./command";
import { ILabelSoftwareTitle } from "./label";
import { HOST_APPLE_PLATFORMS, isLinuxLike, Platform } from "./platform";
import vulnerabilityInterface from "./vulnerability";

export default PropTypes.shape({
  type: PropTypes.string,
  name: PropTypes.string,
  version: PropTypes.string,
  source: PropTypes.string,
  id: PropTypes.number,
  vulnerabilities: PropTypes.arrayOf(vulnerabilityInterface),
});

export interface ISoftwareResponse {
  counts_updated_at: string;
  software: ISoftware[];
}

export interface ISoftwareCountResponse {
  count: number;
}

export interface IGetSoftwareByIdResponse {
  software: ISoftware;
}

// TODO: old software interface. replaced with ISoftwareVersion
// check to see if we still need this.
export interface ISoftware {
  id: number;
  /** All software names displayed by UI is ran through getDisplayedSoftwareName */
  name: string; // e.g., "Figma.app"
  /** Custom name set per team by admin */
  display_name?: string; // e.g. "Figma for Desktop"
  version: string; // e.g., "2.1.11"
  bundle_identifier?: string | null; // e.g., "com.figma.Desktop"
  application_id?: string | null; // e.g., "us.zoom.videomeetings" for Android apps
  source: string; // "apps" | "ipados_apps" | "ios_apps" | "programs" | "rpm_packages" | "deb_packages" | "android_apps" | ?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  last_opened_at?: string | null; // e.g., "2021-08-18T15:11:35Z”
  installed_paths?: string[];
  extension_for?: string;
  vendor?: string;
  release?: string;
  icon_url: string | null; // Only available on team view if an admin uploaded an icon to a team's software
  /** Fleet marked this software as an AI tool (Premium; always false on Free). */
  ai_tool?: boolean;
}

export type IVulnerabilitySoftware = Omit<
  ISoftware,
  "vulnerabilities" | "icon_url"
> & {
  resolved_in_version: string;
};

export interface ISoftwareTitleVersion {
  id: number;
  version: string;
  release?: string;
  vulnerabilities: string[] | null; // TODO: does this return null or is it omitted?
  hosts_count?: number;
}

export interface ISoftwarePatchPolicy {
  id: number;
  name: string;
  patch_when_closed: boolean;
  continuous_automations_enabled: boolean;
  notify_before_patching?: boolean;
}

export type SoftwareInstallPolicyType = "dynamic" | "patch";
export type SoftwareInstallPolicyTypeSet = Set<SoftwareInstallPolicyType>;

// A policy type returned from the API is set to:
// 1. dynamic if only auto install, and
// 2.patch if it's both auto install and patch policy
// This doesn't include patch alone, as policies set to patch only are under ISoftwarePackage.patch_policy
export interface ISoftwareInstallPolicy {
  id: number;
  name: string;
  type: SoftwareInstallPolicyType;
}

// A policy type in the UI uses a Set because a policy in
// Software Details > Policy can be both dynamic AND/OR patch
export interface ISoftwareInstallPolicyUI {
  id: number;
  name: string;
  type: SoftwareInstallPolicyTypeSet;
}

// Match allowedCategories in cmd/maintained-apps/main.go
export type SoftwareCategory =
  | "Browsers"
  | "Communication"
  | "Developer tools"
  | "Productivity"
  | "Security"
  | "Support"
  | "Utilities";

export interface ISoftwarePackageStatus {
  installed: number;
  pending_install: number;
  failed_install: number;
  pending_uninstall: number;
  failed_uninstall: number;
}

export interface ISoftwareAppStoreAppStatus {
  installed: number;
  pending: number;
  failed: number;
}

export interface IFleetMaintainedVersion {
  id: number;
  version: string;
  filename: string;
  uploaded_at: string;
}

export interface ISoftwarePackage {
  /** Per-installer id — distinct from `title_id`. Used by per-package edit
   * and delete endpoints so the request targets one specific package on a
   * title that may have several. */
  installer_id: number;
  name: string;
  /** Not included in SoftwareTitle software.software_package response, hoisted up one level
   * Custom name set per team by admin
   */
  display_name?: string;
  title_id: number;
  url: string;
  version: string;
  uploaded_at: string;
  install_script: string;
  uninstall_script: string;
  pre_install_query?: string;
  post_install_script?: string;
  automatic_install?: boolean; // POST only
  self_service: boolean;
  icon_url: string | null;
  status: ISoftwarePackageStatus;
  patch_policy?: ISoftwarePatchPolicy | null;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
  install_during_setup?: boolean;
  labels_include_any: ILabelSoftwareTitle[] | null;
  labels_include_all: ILabelSoftwareTitle[] | null;
  labels_exclude_any: ILabelSoftwareTitle[] | null;
  categories?: SoftwareCategory[] | null;
  fleet_maintained_app_id?: number | null;
  fleet_maintained_versions?: IFleetMaintainedVersion[] | null;
  /** Version pin: null/absent = Latest, exact version = exact pin, caret
   * ("^149") = major-version pin. */
  pinned_version?: string | null;
  hash_sha256?: string | null;
  /** XML plist string for iOS/iPadOS in-house .ipa managed app configuration. */
  configuration?: string;
}

export interface IAppStoreApp {
  name: string;
  /** Not included in SoftwareTitle software.app_store_app response, hoisted up one level
   * Custom name set per team by admin
   */
  display_name?: string;
  app_store_id: string; // API returns this as a string
  latest_version: string;
  created_at: string;
  icon_url: string;
  self_service: boolean;
  platform: typeof HOST_APPLE_PLATFORMS[number] | "android";
  status: ISoftwareAppStoreAppStatus;
  install_during_setup?: boolean;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
  automatic_install?: boolean;
  last_install?: IAppLastInstall | null;
  last_uninstall?: {
    script_execution_id: string;
    uninstalled_at: string;
  } | null;
  version?: string;
  labels_include_any: ILabelSoftwareTitle[] | null;
  labels_include_all: ILabelSoftwareTitle[] | null;
  labels_exclude_any: ILabelSoftwareTitle[] | null;
  categories?: SoftwareCategory[] | null;
  /** Typed as string but Android configs arrive as a parsed object at runtime
   * (backend sends json.RawMessage which Axios auto-parses). */
  configuration?: string;
}

/**
 * package: includes FMA, custom packages, and are defined under software_package
 * app-store: includes VPP, Google Play Store apps and are defined under app_store_app
 */
export type InstallerType = "package" | "app-store";

export const isSoftwarePackage = (
  data: ISoftwarePackage | IAppStoreApp
): data is ISoftwarePackage =>
  (data as ISoftwarePackage).install_script !== undefined;

export interface ISoftwareTitle {
  id: number;
  /** All software names displayed by UI is ran through getDisplayedSoftwareName */
  name: string;
  /** Custom name set per team by admin */
  display_name?: string;
  bundle_identifier?: string;
  icon_url: string | null;
  versions_count: number;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  /** First-added; mirrors packages[0]. Retained for back-compat. */
  software_package: ISoftwarePackage | null;
  /** All custom packages on this title (trimmed shape on list responses).
   * `null` when the title has no custom packages. */
  packages: ISoftwarePackage[] | null;
  app_store_app: IAppStoreApp | null;
  auto_update_enabled?: boolean;
  auto_update_window_start?: string;
  auto_update_window_end?: string;
  /** Fleet marked this software as an AI tool (Premium; always false on Free). */
  ai_tool?: boolean;
  /** @deprecated Use extension_for instead */
  browser?: string;
}

export interface ISoftwareTitleDetails {
  id: number;
  /** All software names displayed by UI is ran through getDisplayedSoftwareName */
  name: string;
  /** Custom name set per team by admin */
  display_name?: string;
  icon_url: string | null;
  /** First-added; mirrors packages[0]. Retained for back-compat. */
  software_package: ISoftwarePackage | null;
  /** All custom packages on this title, in first-added order (smallest
   * `installer_id` first). `null` when the title has no custom packages.
   * When present, treat as the source of truth; `software_package` is a
   * convenience alias to `packages[0]`. */
  packages: ISoftwarePackage[] | null;
  app_store_app: IAppStoreApp | null;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  counts_updated_at?: string;
  bundle_identifier?: string;
  versions_count?: number;
  auto_update_enabled?: boolean;
  auto_update_window_start?: string;
  auto_update_window_end?: string;
  /** Fleet marked this software as an AI tool (Premium; always false on Free). */
  ai_tool?: boolean;
  /** @deprecated Use extension_for instead */
  browser?: string;
}

export interface ISoftwareVulnerability {
  cve: string;
  details_link: string;
  cvss_score?: number | null;
  epss_probability?: number | null;
  cisa_known_exploit?: boolean | null;
  cve_published?: string | null;
  cve_description?: string | null;
  resolved_in_version?: string | null;
  created_at?: string | null;
}

export interface ISoftwareVersion {
  id: number;
  /** All software names displayed by UI is ran through getDisplayedSoftwareName */
  name: string; // e.g., "Figma.app"
  /** Custom name set per team by admin */
  display_name?: string; // e.g. "Figma for Desktop"
  version: string; // e.g., "2.1.11"
  bundle_identifier?: string; // e.g., "com.figma.Desktop"
  source: SoftwareSource;
  extension_for: SoftwareExtensionFor;
  /** OS release ("30.el7") or, for go_binaries, the Go toolchain version ("go1.26.1"). */
  release: string;
  vendor: string;
  arch: string; // e.g., "x86_64" // TODO: on software/verions/:id?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  /** Fleet marked this software as an AI tool (Premium; always false on Free). */
  ai_tool?: boolean;
  /** @deprecated Use extension_for instead */
  browser?: string;
}

export type SoftwareTypePlatform =
  | "darwin"
  | "windows"
  | "linux"
  | "chrome"
  | "ios"
  | "ipados"
  | "android";

const DESKTOP = ["darwin", "windows", "linux"] as const;

export const MACOS_APP_SOFTWARE_TYPE = "macos_app";

// Keys are used in the page URLs (`?types=`), so never rename them.
// IMPORTANT: Keep sources and extension_for values in sync with
// softwareTypeFilterSources in server/fleet/software.go.
const SOFTWARE_TYPE_VARIANTS = [
  {
    key: "adobe_plugin",
    displayName: "Adobe plugin",
    source: "adobe_plugins",
    platforms: ["darwin", "windows"],
  },
  {
    key: "ai_cli_tool",
    displayName: "AI CLI tool",
    source: "ai_clis",
    platforms: DESKTOP,
    premiumOnly: true,
  },
  {
    key: "ai_skill",
    displayName: "AI skill",
    source: "ai_skills",
    platforms: DESKTOP,
    premiumOnly: true,
  },
  {
    key: "android_app",
    displayName: "Android app",
    source: "android_apps",
    platforms: ["android"],
  },
  {
    key: "brave_extension",
    displayName: "Brave extension",
    source: "chrome_extensions",
    extensionFor: "brave",
    platforms: DESKTOP,
  },
  {
    key: "chocolatey_package",
    displayName: "Chocolatey package",
    source: "chocolatey_packages",
    platforms: ["windows"],
  },
  {
    key: "chrome_extension",
    displayName: "Chrome extension",
    source: "chrome_extensions",
    extensionFor: "chrome",
    platforms: [...DESKTOP, "chrome"],
  },
  {
    key: "chromium_extension",
    displayName: "Chromium extension",
    source: "chrome_extensions",
    extensionFor: "chromium",
    platforms: DESKTOP,
  },
  {
    key: "clion_extension",
    displayName: "CLion extension",
    source: "jetbrains_plugins",
    extensionFor: "clion",
    platforms: DESKTOP,
  },
  {
    key: "cursor_extension",
    displayName: "Cursor extension",
    source: "vscode_extensions",
    extensionFor: "cursor",
    platforms: DESKTOP,
  },
  {
    key: "datagrip_extension",
    displayName: "DataGrip extension",
    source: "jetbrains_plugins",
    extensionFor: "datagrip",
    platforms: DESKTOP,
  },
  {
    key: "deb_package",
    displayName: "deb package",
    source: "deb_packages",
    platforms: ["linux"],
  },
  {
    key: "edge_beta_extension",
    displayName: "Edge Beta extension",
    source: "chrome_extensions",
    extensionFor: "edge_beta",
    platforms: DESKTOP,
  },
  {
    key: "edge_extension",
    displayName: "Edge extension",
    source: "chrome_extensions",
    extensionFor: "edge",
    platforms: DESKTOP,
  },
  {
    key: "firefox_extension",
    displayName: "Firefox extension",
    source: "firefox_addons",
    extensionFor: "firefox",
    platforms: DESKTOP,
  },
  {
    key: "go_binary",
    displayName: "Go binary",
    source: "go_binaries",
    platforms: DESKTOP,
  },
  {
    key: "goland_extension",
    displayName: "GoLand extension",
    source: "jetbrains_plugins",
    extensionFor: "goland",
    platforms: DESKTOP,
  },
  {
    key: "homebrew_package",
    displayName: "Homebrew package",
    source: "homebrew_packages",
    platforms: ["darwin"],
  },
  {
    key: "intellij_idea_community_edition_extension",
    displayName: "IntelliJ IDEA Community Edition extension",
    source: "jetbrains_plugins",
    extensionFor: "intellij_idea_community_edition",
    platforms: DESKTOP,
  },
  {
    key: "intellij_idea_extension",
    displayName: "IntelliJ IDEA extension",
    source: "jetbrains_plugins",
    extensionFor: "intellij_idea",
    platforms: DESKTOP,
  },
  {
    key: "internet_explorer_extension",
    displayName: "Internet Explorer extension",
    source: "ie_extensions",
    platforms: ["windows"],
  },
  {
    key: "ios_app",
    displayName: "iOS app",
    source: "ios_apps",
    platforms: ["ios"],
  },
  {
    key: "ipados_app",
    displayName: "iPadOS app",
    source: "ipados_apps",
    platforms: ["ipados"],
  },
  {
    key: MACOS_APP_SOFTWARE_TYPE,
    displayName: "macOS app",
    source: "apps",
    platforms: ["darwin"],
  },
  {
    key: "macos_package_pkg",
    displayName: "macOS package (.pkg)",
    source: "pkg_packages",
    platforms: ["darwin"],
    installerOnly: true,
  },
  {
    key: "mcp_server",
    displayName: "MCP server",
    source: "mcp_servers",
    platforms: DESKTOP,
    premiumOnly: true,
  },
  {
    key: "nix_package",
    displayName: "Nix package",
    source: "nix_packages",
    platforms: ["linux"],
  },
  {
    key: "npm_package",
    displayName: "npm package",
    source: "npm_packages",
    platforms: ["darwin", "linux"],
  },
  {
    key: "opera_extension",
    displayName: "Opera extension",
    source: "chrome_extensions",
    extensionFor: "opera",
    platforms: DESKTOP,
  },
  {
    key: "pacman_package",
    displayName: "pacman package",
    source: "pacman_packages",
    platforms: ["linux"],
  },
  {
    key: "phpstorm_extension",
    displayName: "PhpStorm extension",
    source: "jetbrains_plugins",
    extensionFor: "phpstorm",
    platforms: DESKTOP,
  },
  {
    key: "portage_package",
    displayName: "Portage package",
    source: "portage_packages",
    platforms: ["linux"],
  },
  {
    key: "pycharm_extension",
    displayName: "PyCharm extension",
    source: "jetbrains_plugins",
    extensionFor: "pycharm",
    platforms: DESKTOP,
  },
  {
    key: "pycharm_community_edition_extension",
    displayName: "PyCharm Community Edition extension",
    source: "jetbrains_plugins",
    extensionFor: "pycharm_community_edition",
    platforms: DESKTOP,
  },
  {
    key: "python_package",
    displayName: "Python package",
    source: "python_packages",
    platforms: DESKTOP,
  },
  {
    key: "resharper_extension",
    displayName: "ReSharper extension",
    source: "jetbrains_plugins",
    extensionFor: "resharper",
    platforms: DESKTOP,
  },
  {
    key: "rider_extension",
    displayName: "Rider extension",
    source: "jetbrains_plugins",
    extensionFor: "rider",
    platforms: DESKTOP,
  },
  {
    key: "rpm_package",
    displayName: "RPM package",
    source: "rpm_packages",
    platforms: ["linux"],
  },
  {
    key: "rubymine_extension",
    displayName: "RubyMine extension",
    source: "jetbrains_plugins",
    extensionFor: "rubymine",
    platforms: DESKTOP,
  },
  {
    key: "rustrover_extension",
    displayName: "RustRover extension",
    source: "jetbrains_plugins",
    extensionFor: "rust_rov",
    platforms: DESKTOP,
  },
  {
    key: "safari_extension",
    displayName: "Safari extension",
    source: "safari_extensions",
    platforms: ["darwin"],
  },
  {
    key: "script_only_package_ps1",
    displayName: "Script-only package (.ps1)",
    source: "ps1_packages",
    platforms: ["windows"],
    installerOnly: true,
  },
  {
    key: "script_only_package_py",
    displayName: "Script-only package (.py)",
    source: "py_packages",
    platforms: ["darwin", "linux"],
    installerOnly: true,
  },
  {
    key: "script_only_package_sh",
    displayName: "Script-only package (.sh)",
    source: "sh_packages",
    platforms: ["darwin", "linux"],
    installerOnly: true,
  },
  {
    key: "tarball_tar_gz",
    displayName: "Tarball (.tar.gz)",
    source: "tgz_packages",
    platforms: ["linux"],
    installerOnly: true,
  },
  {
    key: "trae_extension",
    displayName: "Trae extension",
    source: "vscode_extensions",
    extensionFor: "trae",
    platforms: DESKTOP,
  },
  {
    key: "vs_code_extension",
    displayName: "VS Code extension",
    source: "vscode_extensions",
    extensionFor: "vscode",
    platforms: DESKTOP,
  },
  {
    key: "vs_code_insiders_extension",
    displayName: "VS Code Insiders extension",
    source: "vscode_extensions",
    extensionFor: "vscode_insiders",
    platforms: DESKTOP,
  },
  {
    key: "vscodium_extension",
    displayName: "VSCodium extension",
    source: "vscode_extensions",
    extensionFor: "vscodium",
    platforms: DESKTOP,
  },
  {
    key: "vscodium_insiders_extension",
    displayName: "VSCodium Insiders extension",
    source: "vscode_extensions",
    extensionFor: "vscodium_insiders",
    platforms: DESKTOP,
  },
  {
    key: "webstorm_extension",
    displayName: "WebStorm extension",
    source: "jetbrains_plugins",
    extensionFor: "webstorm",
    platforms: DESKTOP,
  },
  {
    key: "windows_app",
    displayName: "Windows app",
    source: "programs",
    platforms: ["windows"],
  },
  {
    key: "windsurf_extension",
    displayName: "Windsurf extension",
    source: "vscode_extensions",
    extensionFor: "windsurf",
    platforms: DESKTOP,
  },
  {
    key: "yandex_extension",
    displayName: "Yandex extension",
    source: "chrome_extensions",
    extensionFor: "yandex",
    platforms: DESKTOP,
  },
] as const;

// Sources Fleet no longer collects; their labels are kept for historical rows.
const LEGACY_SOURCE_TYPES = {
  apt_sources: "APT package",
  yum_sources: "YUM package",
  atom_packages: "Atom package",
} as const;

export type SoftwareSource =
  | typeof SOFTWARE_TYPE_VARIANTS[number]["source"]
  | keyof typeof LEGACY_SOURCE_TYPES;

export type SoftwareExtensionFor =
  | Extract<
      typeof SOFTWARE_TYPE_VARIANTS[number],
      { extensionFor: string }
    >["extensionFor"]
  | "";

export interface ISoftwareType {
  /** Stable key stored in the page URL (`?types=`). */
  key: string;
  displayName: string;
  source: SoftwareSource;
  extensionFor?: SoftwareExtensionFor;
  platforms: readonly SoftwareTypePlatform[];
  /** Installer-only sources never appear in a host's inventory. */
  installerOnly?: boolean;
  /** Hidden on Fleet Free, where the API rejects the source. */
  premiumOnly?: boolean;
}

const compareDisplayNames = (a: ISoftwareType, b: ISoftwareType) =>
  a.displayName.localeCompare(b.displayName, undefined, {
    sensitivity: "base",
  });

/** Every software type, sorted case-insensitively by display name. */
export const SOFTWARE_TYPES: readonly ISoftwareType[] = [
  ...SOFTWARE_TYPE_VARIANTS,
].sort(compareDisplayNames);

const SOFTWARE_TYPES_BY_KEY = new Map(SOFTWARE_TYPES.map((t) => [t.key, t]));

const softwareTypeLookupKey = (source: string, extensionFor?: string | null) =>
  `${source}/${extensionFor || ""}`;

const SOFTWARE_TYPES_BY_SOURCE = new Map(
  SOFTWARE_TYPES.map((t) => [
    softwareTypeLookupKey(t.source, t.extensionFor),
    t,
  ])
);

/** Labels for extension rows the catalog can't match because extension_for is
 * empty (ingested before Fleet recorded the browser). */
const EXTENSION_SOURCE_FALLBACK: Partial<Record<SoftwareSource, string>> = {
  chrome_extensions: "Browser extension",
  firefox_addons: "Browser extension",
  vscode_extensions: "IDE extension",
  jetbrains_plugins: "IDE extension",
};

const getSoftwareTypePlatform = (
  platform: string
): SoftwareTypePlatform | undefined => {
  switch (platform) {
    case "darwin":
    case "windows":
    case "chrome":
    case "ios":
    case "ipados":
    case "android":
      return platform;
    default:
      return isLinuxLike(platform) ? "linux" : undefined;
  }
};

/** Software types available on the license tier, sorted by display name. */
export const getSoftwareTypes = (opts?: {
  premium?: boolean;
}): ISoftwareType[] =>
  SOFTWARE_TYPES.filter((t) => opts?.premium || !t.premiumOnly);

/** Software types that apply to a host platform, sorted by display name.
 * Host pages hide installer-only types because hosts never report them. */
export const getSoftwareTypesForPlatform = (
  platform: string,
  opts?: { hostPage?: boolean; premium?: boolean }
): ISoftwareType[] => {
  const typePlatform = getSoftwareTypePlatform(platform);
  if (!typePlatform) return [];
  return getSoftwareTypes(opts).filter(
    (t) =>
      t.platforms.includes(typePlatform) && !(opts?.hostPage && t.installerOnly)
  );
};

/** Translates selected type keys into the software list endpoints' `source`
 * and `extension_for` params. Unknown keys are ignored. */
export const softwareTypesToApiParams = (
  keys: readonly string[]
): { source?: string; extension_for?: string } => {
  const selected = new Set(keys);
  const types = SOFTWARE_TYPES.filter((t) => selected.has(t.key));
  const sources = uniq(types.map((t) => t.source)).sort();
  const extensionFor = uniq(
    types
      .map((t) => t.extensionFor)
      .filter((ext): ext is SoftwareExtensionFor => !!ext)
  );
  return {
    ...(sources.length && { source: sources.join(",") }),
    ...(extensionFor.length && { extension_for: extensionFor.join(",") }),
  };
};

/** Parses the `types` URL param, dropping unknown keys. Returns undefined
 * when the param is absent. A repeated param arrives as an array. */
export const parseSoftwareTypesParam = (
  raw: string | string[] | undefined
): string[] | undefined => {
  if (raw === undefined) return undefined;
  const joined = Array.isArray(raw) ? raw.join(",") : raw;
  return uniq(joined.split(",").map((key) => key.trim())).filter((key) =>
    SOFTWARE_TYPES_BY_KEY.has(key)
  );
};

/** Map installable software source to platform  */
export const INSTALLABLE_SOURCE_PLATFORM_CONVERSION = {
  apt_sources: "linux",
  deb_packages: "linux",
  portage_packages: "linux",
  rpm_packages: "linux",
  yum_sources: "linux",
  pacman_packages: "linux",
  nix_packages: "linux",
  tgz_packages: "linux",
  npm_packages: null,
  atom_packages: null,
  python_packages: null,
  apps: "darwin",
  ios_apps: "ios",
  ipados_apps: "ipados",
  android_apps: "android", // 4.76 Currently hidden upstream as not installable
  chrome_extensions: null,
  firefox_addons: null,
  safari_extensions: null,
  homebrew_packages: "darwin",
  programs: "windows",
  ie_extensions: null,
  chocolatey_packages: "windows",
  pkg_packages: "darwin",
  vscode_extensions: null,
  sh_packages: "linux", // 4.76 Added support for Linux hosts only
  ps1_packages: "windows",
  py_packages: "linux", // stored as linux; also runs on macOS via the unix-like install exception
  jetbrains_plugins: null,
  go_binaries: null,
  adobe_plugins: null,
  ai_clis: null,
  ai_skills: null,
  mcp_servers: null,
} as const;

/** Look up an installable source's platform, normalizing the mapping's
 * `null` entries to `undefined` so callers can treat the return as an
 * optional `string`. */
export const getInstallablePlatform = (
  source?: SoftwareSource
): string | undefined => {
  if (!source) return undefined;
  return INSTALLABLE_SOURCE_PLATFORM_CONVERSION[source] ?? undefined;
};

export const SCRIPT_PACKAGE_SOURCES = [
  "sh_packages",
  "ps1_packages",
  "py_packages",
];

/** Mirrors `fleet.MaxPackagesPerTitle` in `server/fleet/software_installer.go`.
 * The backend rejects the upload past this cap with the `SoftwarePackageLimitMessage`
 * conflict error — the UI uses this constant to disable "+ Add package" and
 * surface a matching tooltip before the user hits the API. Keep in sync if
 * the backend limit changes. */
export const MAX_PACKAGES_PER_TITLE = 10;

/** Sources that don't map cleanly to versions or hosts in software inventory.
 * UI behavior for these sources:
 * - Never shows “Update available” (no version to compare against the package version).
 * - Skips showing recently updated and waiting for inventory  UI status/tooltip after successful install/uninstall (no inventory entry to await)
 * - Skips showing a host count (hosts cannot be mapped to the package).
 * - Skips showing a versions table (versions cannot be mapped to the package).
 * - Skips linking to “View all hosts” (hosts cannot be mapped to the package). */
export const NO_VERSION_OR_HOST_DATA_SOURCES = [
  "tgz_packages",
  ...SCRIPT_PACKAGE_SOURCES,
];

/** Sources that never report a version, with the reason shown on the "---" cell. */
export const NO_VERSION_TOOLTIP_BY_SOURCE: Partial<
  Record<SoftwareSource, string>
> = {
  mcp_servers:
    "Fleet doesn't detect MCP server versions yet. Most MCP servers run with npx or uvx, which pick the version at launch unless it's pinned.",
  ai_skills: "AI skills are markdown files, so they don't have versions.",
};

/** Sources Fleet never scans for vulnerabilities, with the reason shown on the "---" cell. */
export const NO_VULNERABILITIES_TOOLTIP_BY_SOURCE: Partial<
  Record<SoftwareSource, string>
> = {
  mcp_servers:
    "Currently, Fleet doesn't detect vulnerabilities for MCP servers.",
  ai_skills:
    "AI skills are markdown files, so they don't have vulnerabilities.",
};

export type InstallableSoftwareSource = keyof typeof INSTALLABLE_SOURCE_PLATFORM_CONVERSION;

/** For go_binaries the toolchain version is part of the row's identity, so it's shown
 * alongside the version. rpm_packages also populates `release` and must not be.
 * Version entries carry no source of their own; spread the entry and add the row's. */
export const formatSoftwareVersion = ({
  version,
  release,
  source,
}: {
  version: string;
  release?: string;
  source?: string;
}) =>
  source === "go_binaries" && release ? `${version} (${release})` : version;

export const formatSoftwareType = ({
  source,
  extension_for,
}: {
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
}) => {
  const extensionFallback = EXTENSION_SOURCE_FALLBACK[source];
  // Only extension sources are split by extension_for; ignore stray values on others.
  const match = SOFTWARE_TYPES_BY_SOURCE.get(
    softwareTypeLookupKey(source, extensionFallback ? extension_for : "")
  );
  if (match) return match.displayName;
  if (extensionFallback) {
    return extension_for ? `${extension_for} extension` : extensionFallback;
  }
  return (
    LEGACY_SOURCE_TYPES[source as keyof typeof LEGACY_SOURCE_TYPES] ?? "Unknown"
  );
};

/**
 * This list comprises all possible states of software install operations.
 */
export const SOFTWARE_UNINSTALL_STATUSES = [
  "uninstalled",
  "pending_uninstall",
  "failed_uninstall",
] as const;

export type SoftwareUninstallStatus = typeof SOFTWARE_UNINSTALL_STATUSES[number];

export const SOFTWARE_INSTALL_STATUSES = [
  "installed",
  "pending_install",
  "failed_install",
] as const;

// Script-only software statuses
export const SOFTWARE_SCRIPT_STATUSES = [
  "ran_script",
  "pending_script",
  "failed_script",
] as const;

export type SoftwareInstallStatus = typeof SOFTWARE_INSTALL_STATUSES[number];

export const SOFTWARE_INSTALL_UNINSTALL_STATUSES = [
  ...SOFTWARE_INSTALL_STATUSES,
  ...SOFTWARE_UNINSTALL_STATUSES,
  // Script-only software statuses use API's SOFTWARE_INSTALL_STATUSES
] as const;

/*
 * SoftwareInstallUninstallStatus represents the possible states of software install operations.
 */
export type SoftwareInstallUninstallStatus = typeof SOFTWARE_INSTALL_UNINSTALL_STATUSES[number];

/** Activity-backed install details can display a skipped state while the
 * persisted install result remains failed_install. */
export type SoftwareInstallDetailsStatus =
  | SoftwareInstallUninstallStatus
  | "skipped_install";

/** Include script-only software statuses */
export const ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES = [
  ...SOFTWARE_INSTALL_STATUSES,
  ...SOFTWARE_UNINSTALL_STATUSES,
  ...SOFTWARE_SCRIPT_STATUSES, // Script-only software
] as const;

/*
 * EnhancedSoftwareInstallUninstallStatus represents the possible states of software install operations including script-only software used in the UI.
 */
export type EnhancedSoftwareInstallUninstallStatus = typeof ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES[number];

export const isValidSoftwareInstallUninstallStatus = (
  s: string | undefined | null
): s is EnhancedSoftwareInstallUninstallStatus =>
  !!s &&
  ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES.includes(
    s as EnhancedSoftwareInstallUninstallStatus
  );

export const SOFTWARE_AGGREGATE_STATUSES = [
  "installed",
  "pending",
  "failed",
] as const;

export type SoftwareAggregateStatus = typeof SOFTWARE_AGGREGATE_STATUSES[number];

export const isValidSoftwareAggregateStatus = (
  s: string | undefined | null
): s is SoftwareAggregateStatus =>
  !!s && SOFTWARE_AGGREGATE_STATUSES.includes(s as SoftwareAggregateStatus);

export const isSoftwareUninstallStatus = (
  s: string | undefined | null
): s is SoftwareUninstallStatus =>
  !!s && SOFTWARE_UNINSTALL_STATUSES.includes(s as SoftwareUninstallStatus);

// not a typeguard, as above 2 functions are
export const isPendingStatus = (s: string | undefined | null) =>
  ["pending_install", "pending_uninstall"].includes(s || "");

export const resolveUninstallStatus = (
  activityStatus?: string
): SoftwareUninstallStatus => {
  let resolvedStatus = activityStatus;
  if (resolvedStatus === "pending") {
    resolvedStatus = "pending_uninstall";
  }
  if (resolvedStatus === "failed") {
    resolvedStatus = "failed_uninstall";
  }
  if (!isSoftwareUninstallStatus(resolvedStatus)) {
    console.warn(
      `Unexpected uninstall status "${activityStatus}" for activity. Defaulting to "pending_uninstall".`
    );
    resolvedStatus = "pending_uninstall";
  }
  return resolvedStatus as SoftwareUninstallStatus;
};

/**
 * ISoftwareInstallResult is the shape of a software install result object
 * returned by the Fleet API.
 */
export interface ISoftwareInstallResult {
  host_display_name?: string;
  install_uuid: string;
  software_title: string;
  software_display_name?: string | null;
  software_title_id: number;
  software_package: string;
  host_id: number;
  status: SoftwareInstallUninstallStatus;
  detail: string;
  output: string;
  pre_install_query_output: string;
  post_install_script_output: string;
  created_at: string;
  updated_at: string | null;
  self_service: boolean;
  /** SHA-256 of the installer package. Present when the payload was
   * hydrated from a package-backed install; absent for VPP / older results
   * whose backend join hasn't been extended. */
  hash_sha256?: string;
}

// Script results are only install results, never uninstall
export type ISoftwareScriptResult = Omit<ISoftwareInstallResult, "status"> & {
  status: SoftwareInstallStatus;
};

export interface ISoftwareInstallResults {
  results: ISoftwareInstallResult;
}

/** For Software .ipa installs, we use the install results API to return MDM command results */
export interface ISoftwareIpaInstallResults {
  results: ICommandResult;
}

// ISoftwareInstallerType defines the supported installer types for
// software uploaded by the IT admin.
export type ISoftwareInstallerType = "pkg" | "msi" | "deb" | "rpm" | "exe";

export interface ISoftwareLastInstall {
  install_uuid: string;
  installed_at: string;
}

export interface IAppLastInstall {
  command_uuid: string;
  installed_at: string;
}

interface SignatureInformation {
  installed_path: string;
  team_identifier: string;
  hash_sha256: string | null;
}
export interface ISoftwareLastUninstall {
  script_execution_id: string;
  uninstalled_at: string;
}

export interface ISoftwareInstallVersion {
  version: string;
  release?: string;
  bundle_identifier: string;
  last_opened_at?: string;
  vulnerabilities: string[] | null;
  installed_paths: string[];
  signature_information?: SignatureInformation[];
}

export interface IHostSoftwarePackage {
  name: string;
  self_service: boolean;
  icon_url: string | null;
  version: string;
  last_install: ISoftwareLastInstall | null;
  last_uninstall: ISoftwareLastUninstall | null;
  categories?: SoftwareCategory[] | null;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
  platform?: Platform;
  /** True when the installer has a non-empty uninstall script. Absent (not
   * `false`) for VPP and in-house apps, and absent on /software/titles
   * responses; only host software responses set it. Used to gate the
   * Uninstall action for script-only (.ps1/.sh/.py) and .tgz packages,
   * where the uninstall script is optional. */
  has_uninstall_script?: boolean;
}

export interface IHostAppStoreApp {
  app_store_id: string;
  platform: Platform;
  self_service: boolean;
  icon_url: string;
  version: string;
  last_install: IAppLastInstall | null;
  categories?: SoftwareCategory[] | null;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
}

export interface IHostSoftware {
  id: number;
  /** All software names displayed by UI is ran through getDisplayedSoftwareName */
  name: string; // e.g., "mock software.app"
  /** Custom name set per team by admin */
  display_name?: string; // e.g. "Mock Software"
  icon_url: string | null;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  bundle_identifier?: string;
  status: Exclude<SoftwareInstallUninstallStatus, "uninstalled"> | null;
  /**
   * True when the most recent install was a patch-when-closed skip (the target
   * app was open); `status` is then `failed_install`. Rendered as "Patch
   * skipped" rather than "Failed".
   */
  skipped_install?: boolean;
  installed_versions: ISoftwareInstallVersion[] | null;
  auto_update_enabled?: boolean;
  auto_update_window_start?: string;
  auto_update_window_end?: string;
  /** Fleet marked this software as an AI tool (Premium; always false on Free). */
  ai_tool?: boolean;
}

/**
 * Comprehensive list of possible UI software statuses for host > software > library/self-service.
 *
 * These are more detailed than the raw API `.status` and are determined by:
 * - Whether the host is online or offline
 * - If the fleet-installed version is newer than any in installed_versions
 * - Special handling for tarballs (tgz_packages)
 * - Cases where the software inventory has not yet updated to reflect a recent change
 *   (i.e., last_install date vs host software's updated_at date)
 */
// Error UI statuses
export const HOST_SOFTWARE_UI_ERROR_STATUSES = [
  "failed_install", // Install attempt failed
  "failed_install_installed", // Install attempt failed but version still present
  "failed_install_update_available", // Install/update failed; newer installer version available
  "failed_uninstall_installed", // Uninstall attempt failed but version still present
  "failed_uninstall", // Uninstall attempt failed
  "failed_uninstall_update_available", // Uninstall/update failed; newer installer version available
  "failed_script", // Script package failed to run
] as const;
export type HostSoftwareUiErrorStatus = typeof HOST_SOFTWARE_UI_ERROR_STATUSES[number];
export const isSoftwareErrorStatus = (
  status: IHostSoftwareUiStatus
): status is HostSoftwareUiErrorStatus =>
  HOST_SOFTWARE_UI_ERROR_STATUSES.includes(status as HostSoftwareUiErrorStatus);

// Pending UI statuses for OFFLINE hosts
export const HOST_SOFTWARE_UI_PENDING_STATUSES = [
  "pending_install", // Install scheduled (no newer installer version)
  "pending_uninstall", // Uninstall scheduled
  "pending_update", // Update scheduled (no newer installer version)
  "pending_script", // Fleet-initiated script run scheduled
] as const;
export type HostSoftwareUiPendingStatus = typeof HOST_SOFTWARE_UI_PENDING_STATUSES[number];
export const isSoftwarePendingStatus = (
  status: IHostSoftwareUiStatus
): status is HostSoftwareUiPendingStatus =>
  HOST_SOFTWARE_UI_PENDING_STATUSES.includes(
    status as HostSoftwareUiPendingStatus
  );

// In-progress UI statuses for ONLINE hosts
export const HOST_SOFTWARE_UI_IN_PROGRESS_STATUSES = [
  "installing", // Fleet-initiated install in progress
  "updating", // Update (install) in progress with newer fleet installer
  "uninstalling", // Fleet-initiated uninstall in progress
  "running_script", // Fleet-initiated script run in progress
] as const;
export type HostSoftwareUiInProgressStatus = typeof HOST_SOFTWARE_UI_IN_PROGRESS_STATUSES[number];
export const isSoftwareInProgressStatus = (
  status: IHostSoftwareUiStatus
): status is HostSoftwareUiInProgressStatus =>
  HOST_SOFTWARE_UI_IN_PROGRESS_STATUSES.includes(
    status as HostSoftwareUiInProgressStatus
  );

// Success/steady-state UI statuses
export const HOST_SOFTWARE_UI_SUCCESS_STATUSES = [
  "installed", // Present in inventory; no newer fleet installer version (tarballs: successful install only)
  "uninstalled", // Not present in inventory (tarballs: successful uninstall or never installed)
  // NOTE: Recently statuses cannot apply to tarballs as we cannot detect inventory
  "recently_updated", // Update applied (installer newer than inventory), but inventory not yet refreshed
  "recently_installed", // Install applied (installer NOT newer than inventory), but inventory not yet refreshed
  "recently_uninstalled", // Uninstall applied, but inventory not yet refreshed
  "ran_script", // Script package ran successfully
  "never_ran_script", // Script package never ran before
] as const;
export type HostSoftwareUiSuccessStatus = typeof HOST_SOFTWARE_UI_SUCCESS_STATUSES[number];
export const isSoftwareSuccessStatus = (
  status: IHostSoftwareUiStatus
): status is HostSoftwareUiSuccessStatus =>
  HOST_SOFTWARE_UI_SUCCESS_STATUSES.includes(
    status as HostSoftwareUiSuccessStatus
  );

// Update-available UI status
export const HOST_SOFTWARE_UI_UPDATE_AVAILABLE_STATUSES = [
  "update_available", // In inventory, but newer fleet installer version is available
  "skipped_install", // Patch-when-closed skip; renders as a deferred update
] as const;
export type HostSoftwareUiUpdateAvailableStatus = typeof HOST_SOFTWARE_UI_UPDATE_AVAILABLE_STATUSES[number];
export const isSoftwareUpdateAvailableStatus = (
  status: IHostSoftwareUiStatus
): status is HostSoftwareUiUpdateAvailableStatus =>
  HOST_SOFTWARE_UI_UPDATE_AVAILABLE_STATUSES.includes(
    status as HostSoftwareUiUpdateAvailableStatus
  );

// Master UI status type, combining all:
export type IHostSoftwareUiStatus =
  | HostSoftwareUiErrorStatus
  | HostSoftwareUiPendingStatus
  | HostSoftwareUiSuccessStatus
  | HostSoftwareUiInProgressStatus
  | HostSoftwareUiUpdateAvailableStatus;

/**
 * Extends IHostSoftware with a computed `ui_status` field.
 *
 * The `ui_status` categorizes software installation state for the UI by
 * combining the `status`, `installed_versions` info, and other factors
 * like host online state (via getUiStatus helper function), enabling
 * more detailed and status labels needed for the status and actions columns.
 */
export interface IHostSoftwareWithUiStatus extends IHostSoftware {
  ui_status: IHostSoftwareUiStatus;
}

/**
 * Allows unified data model for rendering of host VPP software installs and uninstalls
 * Optional as pending may not have a commandUuid
 */
export type IVPPHostSoftware = IHostSoftware & {
  commandUuid?: string;
};

export type IHostSoftwareUninstall = IHostSoftwareWithUiStatus & {
  scriptExecutionId: string;
};

export type IDeviceSoftware = IHostSoftware;
export type IDeviceSoftwareWithUiStatus = IHostSoftwareWithUiStatus;

const INSTALL_STATUS_PREDICATES: Record<
  EnhancedSoftwareInstallUninstallStatus | "pending",
  string
> = {
  pending: "pending",
  installed: "installed",
  uninstalled: "uninstalled",
  pending_install: "told Fleet to install",
  failed_install: "failed to install",
  pending_uninstall: "told Fleet to uninstall",
  failed_uninstall: "failed to uninstall",
  ran_script: "ran", // Script-only software
  failed_script: "failed to run", // Script-only software
  pending_script: "told Fleet to run", // Script-only software
} as const;

export const getInstallUninstallStatusPredicate = (
  status: string | undefined,
  isScriptPackage = false
) => {
  if (!status) {
    return INSTALL_STATUS_PREDICATES.pending;
  }

  // If it is a script package, map install statuses to script-specific predicates
  if (isScriptPackage) {
    switch (status.toLowerCase()) {
      case "installed":
        return INSTALL_STATUS_PREDICATES.ran_script;
      case "pending_install":
        return INSTALL_STATUS_PREDICATES.pending_script;
      case "failed_install":
        return INSTALL_STATUS_PREDICATES.failed_script;
      default:
        break;
    }
  }

  // For all other cases, return the matching predicate or default to pending
  return (
    INSTALL_STATUS_PREDICATES[
      status.toLowerCase() as keyof typeof INSTALL_STATUS_PREDICATES
    ] || INSTALL_STATUS_PREDICATES.pending
  );
};

// Passive-voice variants used for self-service activity rendering, where the
// activity reads "<software> was installed on this host (self-service)." with
// no actor.
const INSTALL_STATUS_PREDICATES_PASSIVE: Record<
  EnhancedSoftwareInstallUninstallStatus | "pending",
  string
> = {
  pending: "is pending",
  installed: "was installed",
  uninstalled: "was uninstalled",
  pending_install: "is pending install",
  failed_install: "installation failed",
  pending_uninstall: "is pending uninstall",
  failed_uninstall: "uninstallation failed",
  ran_script: "was run",
  failed_script: "run failed",
  pending_script: "is pending run",
} as const;

export const getInstallUninstallStatusPredicatePassive = (
  status: string | undefined,
  isScriptPackage = false
) => {
  if (!status) {
    return INSTALL_STATUS_PREDICATES_PASSIVE.pending;
  }

  if (isScriptPackage) {
    switch (status.toLowerCase()) {
      case "installed":
        return INSTALL_STATUS_PREDICATES_PASSIVE.ran_script;
      case "pending_install":
        return INSTALL_STATUS_PREDICATES_PASSIVE.pending_script;
      case "failed_install":
        return INSTALL_STATUS_PREDICATES_PASSIVE.failed_script;
      default:
        break;
    }
  }

  return (
    INSTALL_STATUS_PREDICATES_PASSIVE[
      status.toLowerCase() as keyof typeof INSTALL_STATUS_PREDICATES_PASSIVE
    ] || INSTALL_STATUS_PREDICATES_PASSIVE.pending
  );
};

export const aggregateInstallStatusCounts = (
  packageStatuses: ISoftwarePackage["status"]
) => ({
  installed: packageStatuses.installed,
  pending: packageStatuses.pending_install + packageStatuses.pending_uninstall,
  failed: packageStatuses.failed_install + packageStatuses.failed_uninstall,
});

export const INSTALL_STATUS_ICONS: Record<
  EnhancedSoftwareInstallUninstallStatus | "pending" | "failed",
  IconNames
> = {
  pending: "pending-outline",
  pending_install: "pending-outline",
  installed: "success-outline",
  uninstalled: "success-outline",
  failed: "error-outline",
  failed_install: "error-outline",
  pending_uninstall: "pending-outline",
  failed_uninstall: "error-outline",
  ran_script: "success-outline", // Script-only software
  failed_script: "error-outline", // Script-only software
  pending_script: "pending-outline", // Script-only software
} as const;

type IHostSoftwarePackageWithLastInstall = IHostSoftwarePackage & {
  last_install: ISoftwareLastInstall;
};

export const hasHostSoftwarePackageLastInstall = (
  software: IHostSoftware
): software is IHostSoftware & {
  software_package: IHostSoftwarePackageWithLastInstall;
} => {
  return !!software.software_package?.last_install;
};

type IHostAppWithLastInstall = IHostAppStoreApp & {
  last_install: IAppLastInstall;
};

export const hasHostSoftwareAppLastInstall = (
  software: IHostSoftware
): software is IHostSoftware & {
  app_store_app: IHostAppWithLastInstall;
} => {
  return !!software.app_store_app?.last_install;
};

export const isIpadOrIphoneSoftwareSource = (source: string) =>
  ["ios_apps", "ipados_apps"].includes(source);

export const isIpadOrIphoneSoftware = (platform: string) =>
  ["ios", "ipados"].includes(platform);

export const isAndroidSoftwareSource = (source: string) =>
  source === "android_apps";

export interface IFleetMaintainedApp {
  id: number;
  name: string;
  version: string;
  platform: FleetMaintainedAppPlatform;
  slug: string; // "<app-token>/<platform>", e.g. "figma/darwin"; the token uniquely identifies an app across its platform entries
  software_title_id?: number; // null unless the team already has the software added (as a Fleet-maintained app, App Store (app), or custom package)
}

export type FleetMaintainedAppPlatform = Extract<
  Platform,
  "darwin" | "windows"
>;

export interface ICombinedFMA {
  name: string;
  macos: Omit<IFleetMaintainedApp, "name"> | null;
  windows: Omit<IFleetMaintainedApp, "name"> | null;
}
export interface IFleetMaintainedAppDetails {
  id: number;
  name: string;
  version: string;
  platform: FleetMaintainedAppPlatform;
  pre_install_script: string;
  install_script: string;
  post_install_script: string;
  uninstall_script: string;
  automatic_install_query: string;
  url: string;
  slug: string;
  software_title_id?: number; // null unless the team already has the software added (as a Fleet-maintained app, App Store (app), or custom package)
  categories: SoftwareCategory[] | null;
}

export const ROLLING_ARCH_LINUX_NAMES = [
  "Arch Linux",
  "Arch Linux ARM",
  "Manjaro Linux",
  "Manjaro Linux ARM",
  "Manjaro ARM Linux",
  "CachyOS Linux",
];

export const ROLLING_ARCH_LINUX_VERSIONS = ROLLING_ARCH_LINUX_NAMES.map(
  (name) => `${name} rolling`
);
