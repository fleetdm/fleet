import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number,
  query_id: PropTypes.number,
  query_name: PropTypes.string,
  response: PropTypes.string,
});

export interface IHostPolicy {
  id: number;
  query_id: number;
  query_name: string;
  response: string;
}
