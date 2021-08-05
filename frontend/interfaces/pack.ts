import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  detail_updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  platform: PropTypes.string,
  updated_at: PropTypes.string,
  query_count: PropTypes.number,
  total_hosts_count: PropTypes.number,
  disabled: PropTypes.bool,
});

export interface IPack {
  description: string;
  detail_updated_at: string;
  id: number;
  name: string;
  platform: string;
  updated_at: string;
  query_count: number;
  total_hosts_count: number;
  disabled: boolean;
}
