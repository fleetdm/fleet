import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";

import AcrobatReader from "./AcrobatReader";
import ChromeApp from "./ChromeApp";
import Excel from "./Excel";
import Extension from "./Extension";
import Firefox from "./Firefox";
import AppleApp from "./AppleApp";
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
import AppStore from "./AppStore";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import TeamViewer from "./TeamViewer";
import Box from "./Box";
import Brave from "./Brave";
import Cloudflare from "./Cloudflare";
import Docker from "./Docker";
import Edge from "./Edge";
import Figma from "./Figma";
import Notion from "./Notion";
import WindowsDefender from "./WindowsDefender";
import WhatsApp from "./WhatsApp";
import Postman from "./Postman";
import OnePassword from "./OnePassword";

// Maps all known Linux platforms to the LinuxOS icon
const LINUX_OS_NAME_TO_ICON_MAP = HOST_LINUX_PLATFORMS.reduce(
  (a, platform) => ({ ...a, [platform]: LinuxOS }),
  {}
);

// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
const SOFTWARE_NAME_TO_ICON_MAP = {
  appStore: AppStore,
  "adobe acrobat reader": AcrobatReader,
  "microsoft excel": Excel,
  falcon: Falcon,
  firefox: Firefox,
  "mozilla firefox": Firefox,
  package: Package,
  safari: Safari,
  slack: Slack,
  "microsoft teams": Teams,
  "microsoft visual studio code": VisualStudioCode,
  "visual studio code": VisualStudioCode,
  "microsoft word": Word,
  "google chrome": ChromeApp,
  darwin: MacOS,
  windows: WindowsOS,
  chrome: ChromeOS,
  ios: iOS,
  ipados: iPadOS,
  whatsapp: WhatsApp,
  notion: Notion,
  figma: Figma,
  "microsoft edge": Edge,
  docker: Docker,
  cloudflare: Cloudflare,
  brave: Brave,
  box: Box,
  teamviewer: TeamViewer,
  "windows defender": WindowsDefender,
  postman: Postman,
  "1password": OnePassword,
  ...LINUX_OS_NAME_TO_ICON_MAP,
} as const;

// SOFTWARE_SOURCE_TO_ICON_MAP maps different software sources to a defined
// icon.
const SOFTWARE_SOURCE_TO_ICON_MAP = {
  package: Package,
  apt_sources: Package,
  deb_packages: Package,
  rpm_packages: Package,
  yum_sources: Package,
  npm_packages: Package,
  atom_packages: Package,
  python_packages: Package,
  homebrew_packages: Package,
  apps: AppleApp,
  ios_apps: AppleApp,
  ipados_apps: AppleApp,
  programs: WindowsApp,
  chrome_extensions: Extension,
  safari_extensions: Extension,
  firefox_addons: Extension,
  ie_extensions: Extension,
  chocolatey_packages: Package,
  pkg_packages: Package,
  vscode_extensions: Extension,
} as const;

/**
 * This attempts to loosely match the provided string to a key in a provided dictionary, returning the key if the
 * provided string starts with the key or undefined otherwise.
 */
const matchLoosePrefixToKey = <T extends Record<string, unknown>>(
  dict: T,
  s: string
) => {
  s = s.trim().toLowerCase();
  if (!s) {
    return undefined;
  }
  const match = Object.keys(dict).find((k) =>
    s.startsWith(k.trim().toLowerCase())
  );

  return match ? (match as keyof T) : undefined;
};

/**
 * This strictly matches the provided name and source to a software icon, returning the icon if a match is found or
 * null otherwise. It is intended to be used for special cases where a strict match is required
 * (e.g. Zoom). The caller should handle null cases by falling back to loose matching on name prefixes.
 */
const matchStrictNameSourceToIcon = ({
  name = "",
  source = "",
}: Pick<ISoftware, "name" | "source">) => {
  name = name.trim().toLowerCase();
  source = source.trim().toLowerCase();
  switch (true) {
    case name === "zoom.us.app" && source === "apps":
      return Zoom;
    case name === "zoom":
      return Zoom;
    default:
      return null;
  }
};

/**
 * This returns the icon component for a given software name and source. If a strict match is found,
 * it will be returned, otherwise it will fall back to loose matching on name and source prefixes.
 * If no match is found, the default package icon will be returned.
 */
const getMatchedSoftwareIcon = ({
  name = "",
  source = "",
}: Pick<ISoftware, "name" | "source">) => {
  // first, try strict matching on name and source
  let Icon = matchStrictNameSourceToIcon({
    name,
    source,
  });

  // if no match, try loose matching on name prefixes
  if (!Icon) {
    const matchedName = matchLoosePrefixToKey(SOFTWARE_NAME_TO_ICON_MAP, name);
    if (matchedName) {
      Icon = SOFTWARE_NAME_TO_ICON_MAP[matchedName];
    }
  }

  // if still no match, try loose matching on source prefixes
  if (!Icon) {
    const matchedSource = matchLoosePrefixToKey(
      SOFTWARE_SOURCE_TO_ICON_MAP,
      source
    );
    if (matchedSource) {
      Icon = SOFTWARE_SOURCE_TO_ICON_MAP[matchedSource];
    }
  }

  // if still no match, default to 'package'
  if (!Icon) {
    Icon = SOFTWARE_SOURCE_TO_ICON_MAP.package;
  }

  return Icon;
};

export default getMatchedSoftwareIcon;
