import PropTypes from "prop-types";

export default PropTypes.shape({
  loading_counts: PropTypes.bool,
  new_count: PropTypes.number,
  online_count: PropTypes.number,
  offline_count: PropTypes.number,
  mia_count: PropTypes.number,
});
