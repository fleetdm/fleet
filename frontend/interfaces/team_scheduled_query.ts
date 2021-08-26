import PropTypes from "prop-types";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number.isRequired,
  pack_id: PropTypes.number,
  name: PropTypes.string.isRequired,
  query_id: PropTypes.number.isRequired,
  query: PropTypes.string.isRequired,
  query_name: PropTypes.string.isRequired,
  interval: PropTypes.number.isRequired,
  snapshot: PropTypes.bool,
  removed: PropTypes.bool,
  shard: PropTypes.number,
  platform: PropTypes.string,
  version: PropTypes.string,
});

export interface ITeamScheduledQuery {
  created_at: string;
  updated_at: string;
  id: number;
  pack_id: number;
  name: string;
  query_id: number;
  query: string;
  query_name: string;
  interval: number;
  snapshot: boolean;
  removed: boolean;
  platform?: string;
  version?: string;
  shard?: number;
}
