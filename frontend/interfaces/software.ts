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
