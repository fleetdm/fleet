import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number.isRequired,
  interval: PropTypes.number.isRequired,
  name: PropTypes.string.isRequired,
  pack_id: PropTypes.number.isRequired,
  platform: PropTypes.string,
  query: PropTypes.string.isRequired,
  query_id: PropTypes.number.isRequired,
  removed: PropTypes.bool,
  snapshot: PropTypes.bool,
});

export interface IScheduledQuery {
  id: number;
  interval: number;
  name: string;
  pack_id: number;
  platform?: string;
  query: string;
  query_id: number;
  removed: boolean;
  snapshot: boolean;
}
