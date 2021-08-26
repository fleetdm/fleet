import PropTypes from "prop-types";

export default PropTypes.arrayOf(
  PropTypes.shape({
    secret: PropTypes.string,
    created_at: PropTypes.string,
    team_id: PropTypes.number,
  })
);

export interface IEnrollSecret {
  secret: string;
  created_at: string;
  team_id?: number;
}
