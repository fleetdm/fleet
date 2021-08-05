import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  name: PropTypes.string,
  query: PropTypes.string,
  id: PropTypes.number,
  interval: PropTypes.number,
  last_excuted: PropTypes.string,
  observer_can_run: PropTypes.bool,
  author_name: PropTypes.string,
  updated_at: PropTypes.string,
});
export interface IQuery {
  description: string;
  name: string;
  query: string;
  id: number;
  interval: number;
  last_excuted: string;
  observer_can_run: boolean;
  author_name: string;
  updated_at: string;
}
