import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  name: PropTypes.string,
  query: PropTypes.string,
  id: PropTypes.number,
  interval: PropTypes.number,
  last_excuted: PropTypes.string,
});

export interface IQuery {
  description: string;
  name: string;
  query: string;
  id: number;
  interval: number;
  last_excuted: string;
}
