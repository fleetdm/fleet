import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import Linux from "components/icons/Linux";
import AcrobatReader from "./AcrobatReader";
import ChromeApp from "./ChromeApp";
import Excel from "./Excel";
import Extension from "./Extension";
import Firefox from "./Firefox";
import MacApp from "./MacApp";
import MacOS from "./MacOS";
import Package from "./Package";
import Safari from "./Safari";
import Slack from "./Slack";
import Teams from "./Teams";
import VisualStudioCode from "./VisualStudioCode";
import WindowsApp from "./WindowsApp";
import WindowsOS from "./WindowsOS";
import Word from "./Word";
import Zoom from "./Zoom";
import ChromeOS from "./ChromeOS";
import LinuxOS from "./LinuxOS";
import Falcon from "./Falcon";

// Maps all known Linux platforms to the LinuxOS icon
const LINUX_OS_NAME_TO_ICON_MAP = HOST_LINUX_PLATFORMS.reduce(
  (a, platform) => ({ ...a, [platform]: LinuxOS }),
  {}
);

// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "adobe acrobat reader": AcrobatReader,
  "google chrome": ChromeApp,
  "microsoft excel": Excel,
  falcon: Falcon,
  firefox: Firefox,
  package: Package,
  safari: Safari,
  slack: Slack,
  "microsoft teams": Teams,
  "visual studio code": VisualStudioCode,
  "microsoft word": Word,
  zoom: Zoom,
  darwin: MacOS,
  windows: WindowsOS,
  chrome: ChromeOS,
  ...LINUX_OS_NAME_TO_ICON_MAP,
} as const;

// SOFTWARE_SOURCE_TO_ICON_MAP maps different software sources to a defined
// icon.
export const SOFTWARE_SOURCE_TO_ICON_MAP = {
  package: Package,
  apt_sources: Package,
  deb_packages: Package,
  rpm_packages: Package,
  yum_sources: Package,
  npm_packages: Package,
  atom_packages: Package,
  python_packages: Package,
  homebrew_packages: Package,
  apps: MacApp,
  programs: WindowsApp,
  chrome_extensions: Extension,
  safari_extensions: Extension,
  firefox_addons: Extension,
  ie_extensions: Extension,
  chocolatey_packages: Package,
  pkg_packages: Package,
  vscode_extensions: Extension,
} as const;

export const SOFTWARE_ICON_SIZES: Record<string, string> = {
  medium: "24",
  meduim_large: "64", // TODO: rename this to large and update large to xlarge
  large: "96",
} as const;

export type SoftwareIconSizes = keyof typeof SOFTWARE_ICON_SIZES;

/**
 * List of software names that are exceptions to the software name-to-icon mapping. If one of the
 * listed exceptions is included in the software name, the SOFTWARE_NAME_TO_ICON_MAP should not be used.
 */
const SOFTWARE_NAME_TO_ICON_EXCEPTIONS = ["zoominfo"] as const;

/**
 *
 * Determines if the software name is an exception to the software icon mapping.
 * If the return value is true, it indicates to the caller that it should not use
 * the name-to-icon mapping for this software name.
 *
 * Example: ZoomInfo is an exception that would otherwise be matched to the Zoom icon.
 */
export const isSoftwareNameToIconException = (name: string) => {
  return SOFTWARE_NAME_TO_ICON_EXCEPTIONS.some((exception) =>
    name.trim().toLowerCase().includes(exception)
  );
};
