import PropTypes from "prop-types";

export interface IStatusLabels {
  loading_counts: boolean;
  new_count: number;
  online_count: number;
  offline_count: number;
  mia_count: number;
}

export default PropTypes.shape({
  loading_counts: PropTypes.bool,
  new_count: PropTypes.number,
  online_count: PropTypes.number,
  offline_count: PropTypes.number,
  mia_count: PropTypes.number,
});
