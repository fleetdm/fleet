import PropTypes, { string } from "prop-types";

export default PropTypes.shape({
  online_count: PropTypes.number,
  offline_count: PropTypes.number,
  mia_count: PropTypes.number,
  new_count: PropTypes.number,
});

export interface IHostSummary {
  online_count: number;
  offline_count: number;
  mia_count: number;
  new_count: number;
}
