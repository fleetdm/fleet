import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  scheduled_query_name: PropTypes.string,
  interval: PropTypes.number,
  last_executed: PropTypes.string,
});

export interface IQueryStats {
  description: string;
  scheduled_query_name: string;
  interval: number;
  last_executed: string;
}
