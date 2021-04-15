import PropTypes from "prop-types";

export default PropTypes.arrayOf(
  PropTypes.shape({
    name: PropTypes.string,
    secret: PropTypes.string,
    active: PropTypes.bool,
    created_at: PropTypes.string,
  })
);
