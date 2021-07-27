import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number.isRequired,
  interval: PropTypes.number.isRequired,
  name: PropTypes.string.isRequired,
  shard: PropTypes.number,
  query: PropTypes.string.isRequired,
  query_id: PropTypes.number.isRequired,
  removed: PropTypes.bool,
  snapshot: PropTypes.bool,
});

export interface IGlobalScheduledQuery {
  id: number;
  interval: number;
  name: string;
  shard?: number;
  query: string;
  query_id: number;
  removed: boolean;
  snapshot: boolean;
}
