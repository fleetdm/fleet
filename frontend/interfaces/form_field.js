import PropTypes from "prop-types";

export default PropTypes.shape({
  error: PropTypes.string,
  name: PropTypes.string,
  onChange: PropTypes.func,
  value: PropTypes.oneOfType([
    PropTypes.array,
    PropTypes.bool,
    PropTypes.number,
    PropTypes.string,
  ]),
});
