import PropTypes from "prop-types";

export default PropTypes.shape({
  type: PropTypes.string,
  name: PropTypes.string,
  version: PropTypes.string,
  id: PropTypes.number,
  vulnerabilities: PropTypes.arrayOf(
    PropTypes.shape({
      id: PropTypes.number,
      uid: PropTypes.number,
      username: PropTypes.string,
      type: PropTypes.string,
      groupname: PropTypes.string,
    })
  ),
});
