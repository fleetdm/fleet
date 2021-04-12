import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number,
  name: PropTypes.string.isRequired,
  value: PropTypes.oneOfType([
    PropTypes.string,
    PropTypes.number,
    PropTypes.bool,
  ]),
  read_only: PropTypes.bool,
  type: PropTypes.string,
});
