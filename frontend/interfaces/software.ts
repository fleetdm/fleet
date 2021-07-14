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
  type: string;
  name: string;
  version: string;
  source: string;
  id: number;
  vulnerabilities: IVulnerability[];
}
