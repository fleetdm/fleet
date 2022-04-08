import PropTypes from "prop-types";
import vulnerabilityInterface, { IVulnerability } from "./vulnerability";

export default PropTypes.shape({
  type: PropTypes.string,
  name: PropTypes.string,
  version: PropTypes.string,
  source: PropTypes.string,
  id: PropTypes.number,
  vulnerabilities: PropTypes.arrayOf(vulnerabilityInterface),
});

export interface ISoftware {
  hosts_count?: number;
  id: number;
  name: string; // e.g., "Figma.app"
  version: string; // e.g., "2.1.11"
  source: string; // e.g., "apps"
  generated_cpe: string;
  vulnerabilities: IVulnerability[] | null;
  last_opened_at?: string | null; // e.g., "2021-08-18T15:11:35Z”
  bundle_identifier?: string | null; // e.g., "com.figma.Desktop"
  // type: string;
}

export const TYPE_CONVERSION: Record<string, string> = {
  apt_sources: "Package (APT)",
  deb_packages: "Package (deb)",
  portage_packages: "Package (Portage)",
  rpm_packages: "Package (RPM)",
  yum_sources: "Package (YUM)",
  npm_packages: "Package (NPM)",
  atom_packages: "Package (Atom)",
  python_packages: "Package (Python)",
  apps: "Application (macOS)",
  chrome_extensions: "Browser plugin (Chrome)",
  firefox_addons: "Browser plugin (Firefox)",
  safari_extensions: "Browser plugin (Safari)",
  homebrew_packages: "Package (Homebrew)",
  programs: "Program (Windows)",
  ie_extensions: "Browser plugin (IE)",
  chocolatey_packages: "Package (Chocolatey)",
  pkg_packages: "Package (pkg)",
};

export const formatSoftwareType = (source: string): string => {
  const DICT = TYPE_CONVERSION;
  return DICT[source] || "Unknown";
};
