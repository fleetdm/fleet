import PropTypes from "prop-types";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  description: PropTypes.string,
  type: PropTypes.string,
  disabled: PropTypes.bool,
  query_count: PropTypes.number,
  total_host_count: PropTypes.number,
  host_ids: PropTypes.arrayOf(PropTypes.number),
  label_ids: PropTypes.arrayOf(PropTypes.number),
  team_ids: PropTypes.arrayOf(PropTypes.number),
});

export interface IPack {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  type: string;
  disabled?: boolean;
  query_count: number;
  total_hosts_count: number;
  host_ids: number[];
  label_ids: number[];
  team_ids: number[];
}
