import PropTypes from "prop-types";

export default PropTypes.shape({
  author_email: PropTypes.string.isRequired,
  author_id: PropTypes.number.isRequired,
  author_name: PropTypes.string.isRequired,
  created_at: PropTypes.string.isRequired,
  description: PropTypes.string.isRequired,
  id: PropTypes.number.isRequired,
  name: PropTypes.string.isRequired,
  query: PropTypes.string.isRequired,
  resoluton: PropTypes.string.isRequired,
  response: PropTypes.string,
  team_id: PropTypes.number,
  updated_at: PropTypes.string.isRequired,
});

export interface IHostPolicy {
  author_email: string;
  author_id: number;
  author_name: string;
  created_at: string;
  description?: string;
  id: number;
  name: string;
  query: string;
  resolution: string;
  response: string;
  team_id?: number;
  updated_at: string;
}
