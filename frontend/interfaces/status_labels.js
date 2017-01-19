import { PropTypes } from 'react';

export default PropTypes.shape({
  loading_counts: PropTypes.bool,
  new_count: PropTypes.number,
  online_count: PropTypes.number,
  offline_count: PropTypes.number,
  mia_count: PropTypes.number,
});
