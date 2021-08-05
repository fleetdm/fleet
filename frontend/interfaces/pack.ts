import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  detail_updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  platform: PropTypes.string,
  updated_at: PropTypes.string,
  host_count: PropTypes.number,
  total_query_count: PropTypes.number,
  disabled: PropTypes.bool,
});

export interface IPack {
  description: string;
  detail_updated_at: string;
  id: number;
  name: string;
  platform: string;
  updated_at: string;
  host_count: number;
  total_query_count: number;
  disabled: boolean;
}
