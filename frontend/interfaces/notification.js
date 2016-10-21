import { PropTypes } from 'react';

export default PropTypes.shape({
  alertType: PropTypes.string,
  isVisible: PropTypes.bool,
  message: PropTypes.string,
  undoAction: PropTypes.func,
});
