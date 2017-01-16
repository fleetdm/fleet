import { PropTypes } from 'react';

export default PropTypes.shape({
  hosts_count: PropTypes.number,
  id: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
  title: PropTypes.string,
  type: PropTypes.string,
});
