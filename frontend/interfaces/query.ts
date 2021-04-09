import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  name: PropTypes.string,
  query: PropTypes.string,
  id: PropTypes.number,
});

export interface IQuery {
  description: string;
  name: string;
  query: string;
  id: number;
}
