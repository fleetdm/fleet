import PropTypes from "prop-types";

export default PropTypes.arrayOf(
  PropTypes.shape({
    name: PropTypes.string,
    secret: PropTypes.string,
    active: PropTypes.bool,
    created_at: PropTypes.string,
  })
);

export interface IEnrollSecret {
  name: string;
  secret: string;
  active: boolean;
  created_at: string;
}
